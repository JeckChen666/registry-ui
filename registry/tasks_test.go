package registry

import (
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
}

func (f *fakePurgeClient) RefreshCatalog() {}

func (f *fakePurgeClient) GetRepos() []string {
	return f.repos
}

func (f *fakePurgeClient) FetchAndCacheTagsForRepo(repo string) []string {
	return f.tags[repo]
}

func (f *fakePurgeClient) GetImageInfo(imageRef string) (ImageInfo, error) {
	return ImageInfo{
		ImageSize: f.imageSizes[imageRef],
		Created:   f.created[imageRef],
		IsImage:   true,
	}, nil
}

func (f *fakePurgeClient) DeleteTag(repoPath, tag string) error {
	f.deletedRefs = append(f.deletedRefs, repoPath+":"+tag)
	return nil
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
