package registry

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	regserver "github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/viper"
)

func TestPurgeReportsRegistryThatLeavesTagsAfterDigestDeletion(t *testing.T) {
	server := httptest.NewServer(regserver.New())
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")
	viper.Reset()
	viper.Set("registry.hostname", host)
	viper.Set("registry.insecure", true)
	viper.Set("registry.password", "unused")
	viper.Set("performance.catalog_page_size", 100)
	viper.Set("purge_tags.keep_days", 21)
	viper.Set("purge_tags.keep_count", 1)

	makeImage := func(created time.Time) v1.Image {
		t.Helper()
		cfg, err := empty.Image.ConfigFile()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Created = v1.Time{Time: created}
		img, err := mutate.ConfigFile(empty.Image, cfg)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	ref := func(tag string) name.Tag {
		t.Helper()
		r, err := name.NewTag(host+"/app:"+tag, name.Insecure)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	old := time.Now().UTC().Add(-40 * 24 * time.Hour).Truncate(time.Second)
	shared := makeImage(old)
	for _, tag := range []string{"latest", "old"} {
		if err := remote.Write(ref(tag), shared); err != nil {
			t.Fatal(err)
		}
	}
	if err := remote.Write(ref("other"), makeImage(old.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	client := NewClient()
	result := RunPurgeOldTags(client, false, "", "")
	// The in-memory test registry deletes only the digest entry and leaves
	// tag references accessible. This must not be reported as a successful purge.
	if result.Success || result.DeletedTagCount != 0 || len(result.Errors) == 0 {
		t.Fatalf("unexpected purge result: %+v", result)
	}
	for _, tag := range []string{"latest", "old"} {
		if _, err := remote.Get(ref(tag)); err != nil {
			t.Fatalf("retained tag %s: %v", tag, err)
		}
	}
	if _, err := remote.Get(ref("other")); err != nil {
		t.Fatalf("expected test registry to retain tag: %v", err)
	}
}

func TestPurgeReadsLatestPlatformCreationTime(t *testing.T) {
	server := httptest.NewServer(regserver.New())
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")
	viper.Reset()
	viper.Set("registry.hostname", host)
	viper.Set("registry.insecure", true)
	viper.Set("registry.password", "unused")
	makeImage := func(created time.Time) v1.Image {
		t.Helper()
		cfg, err := empty.Image.ConfigFile()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Created = v1.Time{Time: created}
		img, err := mutate.ConfigFile(empty.Image, cfg)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	older := time.Now().UTC().Add(-40 * 24 * time.Hour).Truncate(time.Second)
	newer := older.Add(24 * time.Hour)
	index := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{Add: makeImage(older)},
		mutate.IndexAddendum{Add: makeImage(newer)},
	)
	ref, err := name.NewTag(host+"/app:multi", name.Insecure)
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.WriteIndex(ref, index); err != nil {
		t.Fatal(err)
	}
	info, err := NewClient().GetPurgeImageInfo("app:multi")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Created.Equal(newer) {
		t.Fatalf("expected latest platform time %v, got %v", newer, info.Created)
	}
}
