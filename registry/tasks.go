package registry

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

type PurgeClient interface {
	RefreshCatalog()
	GetRepos() []string
	FetchAndCacheTagsForRepo(repoName string) []string
	GetImageInfo(imageRef string) (ImageInfo, error)
	DeleteTag(repoPath, tag string) error
}

type TagData struct {
	name    string
	created time.Time
}

func (t TagData) String() string {
	return fmt.Sprintf(`"%s <%s>"`, t.name, t.created.Format("2006-01-02 15:04:05"))
}

type timeSlice []TagData

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	// reverse sort tags on name if equal dates (OCI image case)
	// see https://github.com/Quiq/registry-ui/pull/62
	if p[i].created.Equal(p[j].created) {
		return p[i].name > p[j].name
	}
	return p[i].created.After(p[j].created)
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type PurgeRepoResult struct {
	CandidateTagCount   int      `json:"candidate_tag_count"`
	DeletedTagCount     int      `json:"deleted_tag_count"`
	EstimatedFreedBytes int64    `json:"estimated_freed_bytes"`
	Tags                []string `json:"tags"`
}

type PurgeRunResult struct {
	StartedAt           time.Time                    `json:"started_at"`
	FinishedAt          time.Time                    `json:"finished_at"`
	Success             bool                         `json:"success"`
	DryRun              bool                         `json:"dry_run"`
	CronExpr            string                       `json:"cron_expr"`
	RepoCount           int                          `json:"repo_count"`
	CandidateTagCount   int                          `json:"candidate_tag_count"`
	DeletedTagCount     int                          `json:"deleted_tag_count"`
	EstimatedFreedBytes int64                        `json:"estimated_freed_bytes"`
	Errors              []string                     `json:"errors"`
	Repositories        map[string]PurgeRepoResult   `json:"repositories"`
}

func (r PurgeRunResult) Duration() time.Duration {
	if r.FinishedAt.IsZero() || r.StartedAt.IsZero() {
		return 0
	}
	return r.FinishedAt.Sub(r.StartedAt)
}

func (r PurgeRunResult) SummaryJSON() string {
	if len(r.Repositories) == 0 {
		return "{}"
	}
	data, err := json.Marshal(r.Repositories)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// PurgeOldTags purge old tags.
func PurgeOldTags(client *Client, purgeDryRun bool, purgeIncludeRepos, purgeExcludeRepos string) PurgeRunResult {
	return RunPurgeOldTags(client, purgeDryRun, purgeIncludeRepos, purgeExcludeRepos)
}

// RunPurgeOldTags purge old tags and return a structured summary.
func RunPurgeOldTags(client PurgeClient, purgeDryRun bool, purgeIncludeRepos, purgeExcludeRepos string) PurgeRunResult {
	logger := SetupLogging("registry.tasks.PurgeOldTags")
	startedAt := time.Now().UTC()
	keepDays := viper.GetInt("purge_tags.keep_days")
	keepCount := viper.GetInt("purge_tags.keep_count")
	keepRegexp := viper.GetString("purge_tags.keep_regexp")
	keepFromFile := viper.GetString("purge_tags.keep_from_file")
	result := PurgeRunResult{
		StartedAt:    startedAt,
		DryRun:       purgeDryRun,
		CronExpr:     viper.GetString("purge_tags.cron"),
		Success:      true,
		Repositories: map[string]PurgeRepoResult{},
	}

	dryRunText := ""
	if purgeDryRun {
		logger.Warn("Dry-run mode enabled.")
		dryRunText = "skipped"
	}

	var dataFromFile gjson.Result
	if keepFromFile != "" {
		if _, err := os.Stat(keepFromFile); os.IsNotExist(err) {
			logger.Warnf("Cannot open %s: %s", keepFromFile, err)
			logger.Error("Not purging anything!")
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
			result.FinishedAt = time.Now().UTC()
			return result
		}
		data, err := os.ReadFile(keepFromFile)
		if err != nil {
			logger.Warnf("Cannot read %s: %s", keepFromFile, err)
			logger.Error("Not purging anything!")
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
			result.FinishedAt = time.Now().UTC()
			return result
		}
		dataFromFile = gjson.ParseBytes(data)
	}

	catalog := []string{}
	if purgeIncludeRepos != "" {
		logger.Infof("Including repositories: %s", purgeIncludeRepos)
		catalog = append(catalog, strings.Split(purgeIncludeRepos, ",")...)
	} else {
		client.RefreshCatalog()
		catalog = client.GetRepos()
	}
	if purgeExcludeRepos != "" {
		logger.Infof("Excluding repositories: %s", purgeExcludeRepos)
		tmpCatalog := []string{}
		for _, repo := range catalog {
			if !ItemInSlice(repo, strings.Split(purgeExcludeRepos, ",")) {
				tmpCatalog = append(tmpCatalog, repo)
			}
		}
		catalog = tmpCatalog
	}
	logger.Infof("Working on repositories: %s", catalog)
	result.RepoCount = len(catalog)

	now := time.Now().UTC()
	repos := map[string]timeSlice{}
	imageSizes := map[string]int64{}
	for _, repo := range catalog {
		tags := client.FetchAndCacheTagsForRepo(repo)
		if len(tags) == 0 {
			continue
		}
		logger.Infof("[%s] scanning %d tags...", repo, len(tags))
		for _, tag := range tags {
			imageRef := repo + ":" + tag
			imageInfo, err := client.GetImageInfo(imageRef)
			if err != nil {
				logger.Warnf("[%s] cannot read image info for %s: %s", repo, tag, err)
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s:%s image info error: %s", repo, tag, err))
				continue
			}
			created := imageInfo.Created
			imageSizes[imageRef] = imageInfo.ImageSize
			if created.IsZero() {
				// Image manifest with zero creation time
				logger.Warnf("[%s] tag with zero creation time: %s", repo, tag)
				continue
			}
			repos[repo] = append(repos[repo], TagData{name: tag, created: created})
		}
	}

	logger.Infof("Scanned %d repositories.", len(catalog))
	logger.Infof("Filtering out tags for purging: keep %d days, keep count %d", keepDays, keepCount)
	if keepRegexp != "" {
		logger.Infof("Keeping tags matching regexp: %s", keepRegexp)
	}
	if keepFromFile != "" {
		logger.Infof("Keeping tags from file: %+v", dataFromFile)
	}
	purgeTags := map[string][]string{}
	keepTags := map[string][]string{}
	purgeSizeByRepo := map[string]int64{}
	totalCandidates := 0
	for _, repo := range SortedMapKeys(repos) {
		// Sort tags by "created" from newest to oldest.
		sort.Sort(repos[repo])

		// Prep the list of tags to preserve if defined in the file
		tagsFromFile := []string{}
		for _, i := range dataFromFile.Get(repo).Array() {
			tagsFromFile = append(tagsFromFile, i.String())
		}

		// Filter out tags
		for _, tag := range repos[repo] {
			daysOld := int(now.Sub(tag.created).Hours() / 24)
			matchByRegexp := false
			if keepRegexp != "" {
				matchByRegexp, _ = regexp.MatchString(keepRegexp, tag.name)
			}

			if daysOld > keepDays && !matchByRegexp && !ItemInSlice(tag.name, tagsFromFile) {
				purgeTags[repo] = append(purgeTags[repo], tag.name)
			} else {
				keepTags[repo] = append(keepTags[repo], tag.name)
			}
		}

		// Keep minimal count of tags no matter how old they are.
		if len(keepTags[repo]) < keepCount {
			// At least "threshold"-"keep" but not more than available for "purge".
			takeFromPurge := int(math.Min(float64(keepCount-len(keepTags[repo])), float64(len(purgeTags[repo]))))
			keepTags[repo] = append(keepTags[repo], purgeTags[repo][:takeFromPurge]...)
			purgeTags[repo] = purgeTags[repo][takeFromPurge:]
		}

		for _, tag := range purgeTags[repo] {
			purgeSizeByRepo[repo] += imageSizes[repo+":"+tag]
		}

		totalCandidates += len(purgeTags[repo])
		result.Repositories[repo] = PurgeRepoResult{
			CandidateTagCount:   len(purgeTags[repo]),
			EstimatedFreedBytes: purgeSizeByRepo[repo],
			Tags:                append([]string{}, purgeTags[repo]...),
		}
		logger.Infof("[%s] All %d: %v", repo, len(repos[repo]), repos[repo])
		logger.Infof("[%s] Keep %d: %v", repo, len(keepTags[repo]), keepTags[repo])
		logger.Infof("[%s] Purge %d: %v", repo, len(purgeTags[repo]), purgeTags[repo])
	}

	result.CandidateTagCount = totalCandidates
	for _, repoResult := range result.Repositories {
		result.EstimatedFreedBytes += repoResult.EstimatedFreedBytes
	}
	logger.Infof("There are %d tags to purge.", totalCandidates)
	if totalCandidates > 0 {
		logger.Info("Purging old tags...")
	}

	for _, repo := range SortedMapKeys(purgeTags) {
		if len(purgeTags[repo]) == 0 {
			continue
		}
		logger.Infof("[%s] Purging %d tags... %s", repo, len(purgeTags[repo]), dryRunText)
		if purgeDryRun {
			continue
		}
		repoResult := result.Repositories[repo]
		for _, tag := range purgeTags[repo] {
			if err := client.DeleteTag(repo, tag); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s:%s delete error: %s", repo, tag, err))
				continue
			}
			result.DeletedTagCount++
			repoResult.DeletedTagCount++
		}
		result.Repositories[repo] = repoResult
	}
	logger.Info("Done.")
	result.FinishedAt = time.Now().UTC()
	return result
}
