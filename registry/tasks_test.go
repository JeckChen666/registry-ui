package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type fakePurgeClient struct {
	repos       []string
	tags        map[string][]string
	created     map[string]time.Time
	imageSizes  map[string]int64
	deletedRefs []string
	digests     map[string]string
	catalogErr  error
	listErr     error
	infoErrors  map[string]error
	listCalls   int
	onList      func(*fakePurgeClient)
}

func (f *fakePurgeClient) RefreshCatalog() error { return f.catalogErr }

func (f *fakePurgeClient) GetRepos() []string {
	return f.repos
}

func (f *fakePurgeClient) FetchTagsForPurge(repo string) ([]string, error) {
	f.listCalls++
	if f.onList != nil {
		f.onList(f)
	}
	return append([]string(nil), f.tags[repo]...), f.listErr
}

func (f *fakePurgeClient) GetPurgeImageInfo(imageRef string) (ImageInfo, error) {
	if err := f.infoErrors[imageRef]; err != nil {
		return ImageInfo{}, err
	}
	digest := f.digests[imageRef]
	if digest == "" {
		digest = imageRef
	}
	return ImageInfo{
		ImageSize:      f.imageSizes[imageRef],
		Created:        f.created[imageRef],
		ImageRefDigest: digest,
		IsImage:        true,
	}, nil
}

func (f *fakePurgeClient) DeletePurgeDigest(repoPath, digest string) error {
	remaining := []string{}
	for _, tag := range f.tags[repoPath] {
		ref := repoPath + ":" + tag
		info, _ := f.GetPurgeImageInfo(ref)
		if info.ImageRefDigest == digest {
			f.deletedRefs = append(f.deletedRefs, ref)
		} else {
			remaining = append(remaining, tag)
		}
	}
	f.tags[repoPath] = remaining
	return nil
}

func TestPurgeProtectsSharedDigestAndKeepsMinimum(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 21)
	viper.Set("purge_tags.keep_count", 1)
	old := time.Now().Add(-40 * 24 * time.Hour)
	client := &fakePurgeClient{
		repos:   []string{"app"},
		tags:    map[string][]string{"app": {"latest", "old", "unrelated"}},
		created: map[string]time.Time{"app:latest": old, "app:old": old, "app:unrelated": old.Add(-time.Hour)},
		digests: map[string]string{"app:latest": "shared", "app:old": "shared", "app:unrelated": "other"},
	}
	result := RunPurgeOldTags(client, false, "", "")
	if !result.Success || result.CandidateTagCount != 1 || result.DeletedTagCount != 1 || len(client.deletedRefs) != 1 || client.deletedRefs[0] != "app:unrelated" {
		t.Fatalf("shared retained digest must be preserved: result=%+v deleted=%v", result, client.deletedRefs)
	}
}

func TestPurgeSkipsIncompleteRepository(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 21)
	client := &fakePurgeClient{
		repos: []string{"app"}, tags: map[string][]string{"app": {"old", "unknown"}},
		created:    map[string]time.Time{"app:old": time.Now().Add(-40 * 24 * time.Hour)},
		infoErrors: map[string]error{"app:unknown": fmt.Errorf("registry unavailable")},
	}
	result := RunPurgeOldTags(client, false, "", "")
	if result.Success || result.DeletedTagCount != 0 || len(result.Errors) == 0 {
		t.Fatalf("expected failed scan without deletion: %+v", result)
	}
}

func TestPurgeReportsCatalogAndTagErrors(t *testing.T) {
	for _, test := range []struct {
		name          string
		catalog, list error
	}{
		{"catalog", fmt.Errorf("catalog unavailable"), nil},
		{"tags", nil, fmt.Errorf("tags unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			client := &fakePurgeClient{repos: []string{"app"}, catalogErr: test.catalog, listErr: test.list}
			result := RunPurgeOldTags(client, false, "", "")
			if result.Success || len(result.Errors) == 0 || result.DeletedTagCount != 0 {
				t.Fatalf("expected failure: %+v", result)
			}
		})
	}
}

func TestPurgeUsesExactDayThreshold(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 21)
	client := &fakePurgeClient{repos: []string{"app"}, tags: map[string][]string{"app": {"old", "new"}},
		created: map[string]time.Time{"app:old": time.Now().Add(-21*24*time.Hour - time.Hour), "app:new": time.Now().Add(-21*24*time.Hour + time.Hour)}}
	result := RunPurgeOldTags(client, true, "", "")
	if result.CandidateTagCount != 1 || result.Repositories["app"].Tags[0] != "old" {
		t.Fatalf("unexpected candidates: %+v", result)
	}
}

func TestPurgeStopsIfRepositoryChangesBeforeDeletion(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 21)
	client := &fakePurgeClient{
		repos: []string{"app"}, tags: map[string][]string{"app": {"old"}},
		created: map[string]time.Time{"app:old": time.Now().Add(-40 * 24 * time.Hour)},
		onList: func(f *fakePurgeClient) {
			if f.listCalls == 2 {
				f.tags["app"] = append(f.tags["app"], "new")
			}
		},
	}
	result := RunPurgeOldTags(client, false, "", "")
	if result.Success || len(result.Errors) == 0 || len(client.deletedRefs) != 0 {
		t.Fatalf("expected changed repository to be skipped: %+v", result)
	}
}

func TestPurgeDeletesSharedCandidatesOnce(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 21)
	client := &fakePurgeClient{
		repos: []string{"app"}, tags: map[string][]string{"app": {"one", "two"}},
		created: map[string]time.Time{"app:one": time.Now().Add(-40 * 24 * time.Hour), "app:two": time.Now().Add(-40 * 24 * time.Hour)},
		digests: map[string]string{"app:one": "shared", "app:two": "shared"},
	}
	result := RunPurgeOldTags(client, false, "", "")
	if !result.Success || result.DeletedTagCount != 2 || len(client.deletedRefs) != 2 {
		t.Fatalf("expected both aliases to be removed: %+v", result)
	}
}

func TestRunPurgeOldTagsDryRunSummary(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 30)
	viper.Set("purge_tags.keep_count", 1)
	viper.Set("purge_tags.keep_regexp", "")
	viper.Set("purge_tags.keep_from_file", "")

	now := time.Now().UTC()
	client := &fakePurgeClient{
		repos: []string{"app"},
		tags: map[string][]string{
			"app": {"new", "old"},
		},
		created: map[string]time.Time{
			"app:new": now.Add(-10 * 24 * time.Hour),
			"app:old": now.Add(-60 * 24 * time.Hour),
		},
		imageSizes: map[string]int64{
			"app:old": 512,
		},
	}

	result := RunPurgeOldTags(client, true, "", "")

	if !result.DryRun {
		t.Fatalf("expected dry run result")
	}
	if result.CandidateTagCount != 1 {
		t.Fatalf("expected 1 purge candidate, got %d", result.CandidateTagCount)
	}
	if result.DeletedTagCount != 0 {
		t.Fatalf("expected 0 deleted tags in dry-run, got %d", result.DeletedTagCount)
	}
	if result.EstimatedFreedBytes != 512 {
		t.Fatalf("expected estimated freed bytes 512, got %d", result.EstimatedFreedBytes)
	}
	if len(client.deletedRefs) != 0 {
		t.Fatalf("expected no delete operations in dry-run, got %v", client.deletedRefs)
	}
	if repo, ok := result.Repositories["app"]; !ok || repo.CandidateTagCount != 1 {
		t.Fatalf("expected app repo summary with one candidate, got %+v", result.Repositories)
	}
}

func TestRunPurgeOldTagsDeleteExecution(t *testing.T) {
	viper.Reset()
	viper.Set("purge_tags.keep_days", 30)
	viper.Set("purge_tags.keep_count", 1)
	viper.Set("purge_tags.keep_regexp", "")
	viper.Set("purge_tags.keep_from_file", "")

	now := time.Now().UTC()
	client := &fakePurgeClient{
		repos: []string{"app"},
		tags: map[string][]string{
			"app": {"new", "old"},
		},
		created: map[string]time.Time{
			"app:new": now.Add(-10 * 24 * time.Hour),
			"app:old": now.Add(-60 * 24 * time.Hour),
		},
		imageSizes: map[string]int64{
			"app:old": 2048,
		},
	}

	result := RunPurgeOldTags(client, false, "", "")

	if result.CandidateTagCount != 1 {
		t.Fatalf("expected 1 purge candidate, got %d", result.CandidateTagCount)
	}
	if result.DeletedTagCount != 1 {
		t.Fatalf("expected 1 deleted tag, got %d", result.DeletedTagCount)
	}
	if len(client.deletedRefs) != 1 || client.deletedRefs[0] != "app:old" {
		t.Fatalf("expected app:old delete call, got %v", client.deletedRefs)
	}
	if !result.Success {
		t.Fatalf("expected successful purge result, got %+v", result)
	}
}
