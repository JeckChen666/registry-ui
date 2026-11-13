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
	RefreshCatalog() error
	GetRepos() []string
	FetchTagsForPurge(repoName string) ([]string, error)
	GetPurgeImageInfo(imageRef string) (ImageInfo, error)
	DeletePurgeDigest(repoPath, digest string) error
}

type TagData struct {
	name    string
	created time.Time
	digest  string
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
	StartedAt           time.Time                  `json:"started_at"`
	FinishedAt          time.Time                  `json:"finished_at"`
	Success             bool                       `json:"success"`
	DryRun              bool                       `json:"dry_run"`
	CronExpr            string                     `json:"cron_expr"`
	RepoCount           int                        `json:"repo_count"`
	CandidateTagCount   int                        `json:"candidate_tag_count"`
	DeletedTagCount     int                        `json:"deleted_tag_count"`
	EstimatedFreedBytes int64                      `json:"estimated_freed_bytes"`
	Errors              []string                   `json:"errors"`
	Repositories        map[string]PurgeRepoResult `json:"repositories"`
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
	if keepDays < 0 || keepCount < 0 {
		result.Success = false
		result.Errors = append(result.Errors, "keep_days and keep_count must not be negative")
		result.FinishedAt = time.Now().UTC()
		return result
	}
	var keepPattern *regexp.Regexp
	if keepRegexp != "" {
		var err error
		keepPattern, err = regexp.Compile(keepRegexp)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("invalid keep regexp: %v", err))
			result.FinishedAt = time.Now().UTC()
			return result
		}
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
		if !gjson.ValidBytes(data) {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON in keep file %s", keepFromFile))
			result.FinishedAt = time.Now().UTC()
			return result
		}
		dataFromFile = gjson.ParseBytes(data)
		if !dataFromFile.IsObject() {
			result.Success = false
			result.Errors = append(result.Errors, "keep file must contain an object of repository names and tag arrays")
			result.FinishedAt = time.Now().UTC()
			return result
		}
		for repo, tags := range dataFromFile.Map() {
			if !tags.IsArray() {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("keep file entry for %s must be an array", repo))
				result.FinishedAt = time.Now().UTC()
				return result
			}
			for _, tag := range tags.Array() {
				if tag.Type != gjson.String {
					result.Success = false
					result.Errors = append(result.Errors, fmt.Sprintf("keep file entry for %s contains a non-string tag", repo))
					result.FinishedAt = time.Now().UTC()
					return result
				}
			}
		}
	}

	catalog := []string{}
	if purgeIncludeRepos != "" {
		logger.Infof("Including repositories: %s", purgeIncludeRepos)
		catalog = append(catalog, strings.Split(purgeIncludeRepos, ",")...)
	} else {
		if err := client.RefreshCatalog(); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("catalog error: %v", err))
			result.FinishedAt = time.Now().UTC()
			return result
		}
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
	snapshots := map[string]map[string]string{}
	for _, repo := range catalog {
		tags, err := client.FetchTagsForPurge(repo)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s tag listing error: %v", repo, err))
			continue
		}
		if len(tags) == 0 {
			continue
		}
		snapshot := map[string]string{}
		valid := true
		logger.Infof("[%s] scanning %d tags...", repo, len(tags))
		for _, tag := range tags {
			imageRef := repo + ":" + tag
			imageInfo, err := client.GetPurgeImageInfo(imageRef)
			if err != nil {
				logger.Warnf("[%s] cannot read image info for %s: %s", repo, tag, err)
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s:%s image info error: %s", repo, tag, err))
				valid = false
				continue
			}
			created := imageInfo.Created
			imageSizes[imageRef] = imageInfo.ImageSize
			if created.IsZero() || imageInfo.ImageRefDigest == "" {
				// Image manifest with zero creation time
				logger.Warnf("[%s] tag with zero creation time: %s", repo, tag)
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s:%s missing creation time or digest", repo, tag))
				valid = false
				continue
			}
			snapshot[tag] = imageInfo.ImageRefDigest
			repos[repo] = append(repos[repo], TagData{name: tag, created: created, digest: imageInfo.ImageRefDigest})
		}
		if !valid {
			delete(repos, repo)
			continue
		}
		snapshots[repo] = snapshot
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
		for _, i := range dataFromFile.Map()[repo].Array() {
			tagsFromFile = append(tagsFromFile, i.String())
		}

		// Filter out tags
		for _, tag := range repos[repo] {
			matchByRegexp := keepPattern != nil && keepPattern.MatchString(tag.name)

			if now.Sub(tag.created) > time.Duration(keepDays)*24*time.Hour && !matchByRegexp && !ItemInSlice(tag.name, tagsFromFile) {
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
		// Digest deletion also removes aliases, including those retained by count.
		protected := map[string]bool{}
		for _, tag := range keepTags[repo] {
			protected[snapshots[repo][tag]] = true
		}
		candidates := purgeTags[repo][:0]
		for _, tag := range purgeTags[repo] {
			if protected[snapshots[repo][tag]] {
				logger.Warnf("[%s] keeping %s because its image is shared with a retained tag", repo, tag)
				keepTags[repo] = append(keepTags[repo], tag)
				continue
			}
			candidates = append(candidates, tag)
		}
		purgeTags[repo] = candidates

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
		groups := map[string][]string{}
		for _, tag := range purgeTags[repo] {
			groups[snapshots[repo][tag]] = append(groups[snapshots[repo][tag]], tag)
		}
		// Recheck once per repository to avoid quadratic registry traffic.
		if err := verifyPurgeSnapshot(client, repo, snapshots[repo]); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s verification error: %v", repo, err))
			continue
		}
		for _, digest := range SortedMapKeys(groups) {
			if err := client.DeletePurgeDigest(repo, digest); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s:%v delete error: %s", repo, groups[digest], err))
				continue
			}
			remaining, err := client.FetchTagsForPurge(repo)
			if err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s could not verify deleted tags: %v", repo, err))
				break
			}
			stillPresent := false
			for _, tag := range groups[digest] {
				if ItemInSlice(tag, remaining) {
					stillPresent = true
				}
			}
			if stillPresent {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s tags still present after digest deletion: %v", repo, groups[digest]))
				break
			}
			for _, tag := range groups[digest] {
				delete(snapshots[repo], tag)
			}
			result.DeletedTagCount += len(groups[digest])
			repoResult.DeletedTagCount += len(groups[digest])
		}
		result.Repositories[repo] = repoResult
	}
	logger.Info("Done.")
	result.FinishedAt = time.Now().UTC()
	return result
}

func verifyPurgeSnapshot(client PurgeClient, repo string, snapshot map[string]string) error {
	tags, err := client.FetchTagsForPurge(repo)
	if err != nil {
		return err
	}
	if len(tags) != len(snapshot) {
		return fmt.Errorf("tags changed during purge")
	}
	for _, tag := range tags {
		expected, ok := snapshot[tag]
		if !ok {
			return fmt.Errorf("tags changed during purge")
		}
		info, err := client.GetPurgeImageInfo(repo + ":" + tag)
		if err != nil {
			return err
		}
		if info.ImageRefDigest != expected {
			return fmt.Errorf("tag %s changed during purge", tag)
		}
	}
	return nil
}
