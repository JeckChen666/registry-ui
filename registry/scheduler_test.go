package registry

import "testing"

func TestStartPurgeSchedulerDisabled(t *testing.T) {
	scheduler, err := StartPurgeScheduler("", func() {})
	if err != nil {
		t.Fatalf("expected no error for disabled scheduler, got %v", err)
	}
	if scheduler != nil {
		t.Fatalf("expected nil scheduler when cron is empty")
	}
}

func TestStartPurgeSchedulerInvalidCron(t *testing.T) {
	scheduler, err := StartPurgeScheduler("not-a-cron", func() {})
	if err == nil {
		t.Fatal("expected invalid cron expression error")
	}
	if scheduler != nil {
		t.Fatalf("expected nil scheduler for invalid cron, got %+v", scheduler)
	}
}
