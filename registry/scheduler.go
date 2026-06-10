package registry

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type PurgeScheduler struct {
	stop chan struct{}
}

type cronSchedule struct {
	minute  cronField
	hour    cronField
	day     cronField
	month   cronField
	weekday cronField
}

type fieldMatcher func(int) bool

type cronField struct {
	match    fieldMatcher
	wildcard bool
}

// StartPurgeScheduler starts the built-in purge cron scheduler.
func StartPurgeScheduler(cronExpr string, run func()) (*PurgeScheduler, error) {
	if cronExpr == "" {
		return nil, nil
	}

	schedule, err := parseCronSchedule(cronExpr)
	if err != nil {
		return nil, err
	}

	scheduler := &PurgeScheduler{stop: make(chan struct{})}
	var running atomic.Bool
	go func() {
		logger := SetupLogging("registry.scheduler")
		for {
			nextRun := schedule.Next(time.Now())
			timer := time.NewTimer(time.Until(nextRun))
			select {
			case <-timer.C:
				if !running.CompareAndSwap(false, true) {
					logger.Warn("Skipping purge run because the previous run is still in progress.")
					continue
				}
				func() {
					defer running.Store(false)
					run()
				}()
			case <-scheduler.stop:
				timer.Stop()
				return
			}
		}
	}()

	return scheduler, nil
}

func parseCronSchedule(expr string) (*cronSchedule, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("expected 5 cron fields, got %d", len(parts))
	}

	minute, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}
	hour, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}
	day, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}
	month, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}
	weekday, err := parseCronField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return &cronSchedule{
		minute:  minute,
		hour:    hour,
		day:     day,
		month:   month,
		weekday: weekday,
	}, nil
}

func (c *cronSchedule) Next(after time.Time) time.Time {
	candidate := after.Truncate(time.Minute).Add(time.Minute)
	limit := candidate.Add(366 * 24 * time.Hour)
	for !candidate.After(limit) {
		if c.matches(candidate) {
			return candidate
		}
		candidate = candidate.Add(time.Minute)
	}
	return after.Add(time.Minute)
}

func (c *cronSchedule) matches(t time.Time) bool {
	if !c.minute.match(t.Minute()) || !c.hour.match(t.Hour()) || !c.month.match(int(t.Month())) {
		return false
	}

	dayMatch := c.day.match(t.Day())
	weekdayMatch := c.weekday.match(int(t.Weekday()))
	switch {
	case c.day.wildcard && c.weekday.wildcard:
		return true
	case c.day.wildcard:
		return weekdayMatch
	case c.weekday.wildcard:
		return dayMatch
	default:
		return dayMatch || weekdayMatch
	}
}

func parseCronField(expr string, min, max int) (cronField, error) {
	allowed := map[int]bool{}
	wildcard := false
	parts := strings.Split(expr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return cronField{}, fmt.Errorf("empty token")
		}
		if part == "*" {
			wildcard = true
		}
		values, err := expandCronToken(part, min, max)
		if err != nil {
			return cronField{}, err
		}
		for _, value := range values {
			allowed[value] = true
		}
	}

	return cronField{
		match: func(v int) bool {
			return allowed[v]
		},
		wildcard: wildcard,
	}, nil
}

func expandCronToken(token string, min, max int) ([]int, error) {
	step := 1
	base := token
	if strings.Contains(token, "/") {
		parts := strings.Split(token, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid step syntax %q", token)
		}
		base = parts[0]
		parsedStep, err := strconv.Atoi(parts[1])
		if err != nil || parsedStep <= 0 {
			return nil, fmt.Errorf("invalid step %q", parts[1])
		}
		step = parsedStep
	}

	var start, end int
	switch {
	case base == "*" || base == "":
		start, end = min, max
	case strings.Contains(base, "-"):
		parts := strings.Split(base, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range %q", base)
		}
		var err error
		start, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid range start %q", parts[0])
		}
		end, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid range end %q", parts[1])
		}
	default:
		value, err := strconv.Atoi(base)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", base)
		}
		if max == 6 && value == 7 {
			value = 0
		}
		start, end = value, value
	}

	if start < min || end > max || start > end {
		return nil, fmt.Errorf("value %d-%d out of range %d-%d", start, end, min, max)
	}

	values := []int{}
	for value := start; value <= end; value += step {
		values = append(values, value)
	}
	return values, nil
}
