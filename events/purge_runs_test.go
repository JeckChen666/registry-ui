package events

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/quiq/registry-ui/registry"
	"github.com/spf13/viper"
)

func TestRecordAndListPurgeRuns(t *testing.T) {
	viper.Reset()
	viper.Set("event_listener.database_driver", "sqlite3")
	viper.Set("event_listener.database_location", filepath.Join(t.TempDir(), "events.db"))
	viper.Set("event_listener.retention_days", 7)
	viper.Set("event_listener.deletion_enabled", true)
	viper.Set("purge_tags.log_retention_days", 30)

	listener := NewEventListener()
	now := time.Now().UTC().Truncate(time.Second)
	err := listener.RecordPurgeRun(registry.PurgeRunResult{
		StartedAt:           now,
		FinishedAt:          now.Add(2 * time.Second),
		Success:             true,
		DryRun:              false,
		CronExpr:            "0 3 * * *",
		RepoCount:           2,
		CandidateTagCount:   4,
		DeletedTagCount:     3,
		EstimatedFreedBytes: 4096,
	})
	if err != nil {
		t.Fatalf("expected purge run to be recorded, got %v", err)
	}

	runs := listener.GetPurgeRuns(10)
	if len(runs) != 1 {
		t.Fatalf("expected one purge run, got %d", len(runs))
	}
	if runs[0].DeletedTagCount != 3 {
		t.Fatalf("expected deleted count 3, got %+v", runs[0])
	}
	if runs[0].EstimatedFreedBytes != 4096 {
		t.Fatalf("expected estimated freed bytes 4096, got %+v", runs[0])
	}
}
