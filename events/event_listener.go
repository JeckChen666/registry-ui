package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/quiq/registry-ui/registry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	// 🐒 patching of "database/sql".
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tidwall/gjson"
)

const (
	userAgent    = "registry-ui"
	schemaSQLite = `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action CHAR(5) NULL,
		repository VARCHAR(100) NULL,
		tag VARCHAR(100) NULL,
		ip VARCHAR(45) NULL,
		user VARCHAR(50) NULL,
		created DATETIME NULL
	);
`
	schemaPurgeRunsSQLite = `
	CREATE TABLE IF NOT EXISTS purge_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at DATETIME NULL,
		finished_at DATETIME NULL,
		success BOOLEAN NULL,
		dry_run BOOLEAN NULL,
		cron_expr VARCHAR(100) NULL,
		repo_count INTEGER NULL,
		candidate_tag_count INTEGER NULL,
		deleted_tag_count INTEGER NULL,
		estimated_freed_bytes BIGINT NULL,
		summary TEXT NULL,
		error_message TEXT NULL,
		created DATETIME NULL
	);
`
)

// EventListener event listener
type EventListener struct {
	databaseDriver   string
	databaseLocation string
	retention        int
	eventDeletion    bool
	logger           *logrus.Entry
}

type eventData struct {
	Events []interface{} `json:"events"`
}

// EventRow event row from sqlite
type EventRow struct {
	ID         int
	Action     string
	Repository string
	Tag        string
	IP         string
	User       string
	Created    string
}

type PurgeRunRow struct {
	ID                  int
	StartedAt           string
	FinishedAt          string
	Success             bool
	DryRun              bool
	CronExpr            string
	RepoCount           int
	CandidateTagCount   int
	DeletedTagCount     int
	EstimatedFreedBytes int64
	Summary             string
	ErrorMessage        string
	Created             string
}

// NewEventListener initialize EventListener.
func NewEventListener() *EventListener {
	databaseDriver := viper.GetString("event_listener.database_driver")
	databaseLocation := viper.GetString("event_listener.database_location")
	retention := viper.GetInt("event_listener.retention_days")
	eventDeletion := viper.GetBool("event_listener.deletion_enabled")

	if databaseDriver != "sqlite3" && databaseDriver != "mysql" {
		panic(fmt.Errorf("event_database_driver should be either sqlite3 or mysql"))
	}

	return &EventListener{
		databaseDriver:   databaseDriver,
		databaseLocation: databaseLocation,
		retention:        retention,
		eventDeletion:    eventDeletion,
		logger:           registry.SetupLogging("events.event_listener"),
	}
}

// ProcessEvents parse and store registry events
func (e *EventListener) ProcessEvents(request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	var t eventData
	if err := decoder.Decode(&t); err != nil {
		e.logger.Errorf("Problem decoding event from request: %+v", request)
		return
	}
	e.logger.Debugf("Received event: %+v", t)
	j, _ := json.Marshal(t)

	db, err := e.getDatabaseHandler()
	if err != nil {
		e.logger.Error(err)
		return
	}
	defer db.Close()

	now := "DateTime('now')"
	if e.databaseDriver == "mysql" {
		now = "NOW()"
	}
	stmt, _ := db.Prepare("INSERT INTO events(action, repository, tag, ip, user, created) values(?,?,?,?,?," + now + ")")
	for _, i := range gjson.GetBytes(j, "events").Array() {
		// Ignore calls by registry-ui itself.
		if strings.HasPrefix(i.Get("request.useragent").String(), userAgent) {
			continue
		}
		action := i.Get("action").String()
		repository := i.Get("target.repository").String()
		tag := i.Get("target.tag").String()
		// Tag is empty in case of signed pull.
		if tag == "" {
			tag = i.Get("target.digest").String()
		}
		ip := i.Get("request.addr").String()
		if x, _, _ := net.SplitHostPort(ip); x != "" {
			ip = x
		}
		user := i.Get("actor.name").String()
		e.logger.Debugf("Parsed event data: %s %s:%s %s %s ", action, repository, tag, ip, user)

		res, err := stmt.Exec(action, repository, tag, ip, user)
		if err != nil {
			e.logger.Error("Error inserting a row: ", err)
			return
		}
		id, _ := res.LastInsertId()
		e.logger.Debug("New event added with id ", id)
	}

	// Purge old records.
	if !e.eventDeletion {
		return
	}
	var res sql.Result
	if e.databaseDriver == "mysql" {
		stmt, _ := db.Prepare("DELETE FROM events WHERE created < DATE_SUB(NOW(), INTERVAL ? DAY)")
		res, _ = stmt.Exec(e.retention)
	} else {
		stmt, _ := db.Prepare("DELETE FROM events WHERE created < DateTime('now',?)")
		res, _ = stmt.Exec(fmt.Sprintf("-%d day", e.retention))
	}
	count, _ := res.RowsAffected()
	e.logger.Debug("Rows deleted: ", count)
}

// GetEvents retrieve events from sqlite db
func (e *EventListener) GetEvents(repository string) []EventRow {
	var events []EventRow

	db, err := e.getDatabaseHandler()
	if err != nil {
		e.logger.Error(err)
		return events
	}
	defer db.Close()

	query := "SELECT * FROM events ORDER BY id DESC LIMIT 1000"
	if repository != "" {
		query = fmt.Sprintf("SELECT * FROM events WHERE repository='%s' OR repository LIKE '%s/%%' ORDER BY id DESC LIMIT 5",
			repository, repository)
	}
	rows, err := db.Query(query)
	if err != nil {
		e.logger.Error("Error selecting from table: ", err)
		return events
	}
	defer rows.Close()

	for rows.Next() {
		var row EventRow
		rows.Scan(&row.ID, &row.Action, &row.Repository, &row.Tag, &row.IP, &row.User, &row.Created)
		events = append(events, row)
	}
	return events
}

// GetEventCount returns the total number of events in the database.
func (e *EventListener) GetEventCount() int {
	db, err := e.getDatabaseHandler()
	if err != nil {
		e.logger.Error(err)
		return 0
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		e.logger.Error("Error counting events: ", err)
		return 0
	}
	return count
}

// RecordPurgeRun persists a purge execution summary.
func (e *EventListener) RecordPurgeRun(run registry.PurgeRunResult) error {
	db, err := e.getDatabaseHandler()
	if err != nil {
		e.logger.Error(err)
		return err
	}
	defer db.Close()

	now := "DateTime('now')"
	if e.databaseDriver == "mysql" {
		now = "NOW()"
	}

	stmt, err := db.Prepare("INSERT INTO purge_runs(started_at, finished_at, success, dry_run, cron_expr, repo_count, candidate_tag_count, deleted_tag_count, estimated_freed_bytes, summary, error_message, created) values(?,?,?,?,?,?,?,?,?,?,?," + now + ")")
	if err != nil {
		e.logger.Error("Error preparing purge run insert: ", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		run.StartedAt.UTC().Format("2006-01-02 15:04:05"),
		run.FinishedAt.UTC().Format("2006-01-02 15:04:05"),
		run.Success,
		run.DryRun,
		run.CronExpr,
		run.RepoCount,
		run.CandidateTagCount,
		run.DeletedTagCount,
		run.EstimatedFreedBytes,
		run.SummaryJSON(),
		strings.Join(run.Errors, "\n"),
	)
	if err != nil {
		e.logger.Error("Error inserting purge run: ", err)
		return err
	}

	retention := viper.GetInt("purge_tags.log_retention_days")
	if retention <= 0 {
		return nil
	}
	if e.databaseDriver == "mysql" {
		stmt, err = db.Prepare("DELETE FROM purge_runs WHERE created < DATE_SUB(NOW(), INTERVAL ? DAY)")
		if err != nil {
			return err
		}
		defer stmt.Close()
		_, err = stmt.Exec(retention)
		return err
	}

	stmt, err = db.Prepare("DELETE FROM purge_runs WHERE created < DateTime('now',?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(fmt.Sprintf("-%d day", retention))
	return err
}

// GetPurgeRuns returns recent purge execution logs.
func (e *EventListener) GetPurgeRuns(limit int) []PurgeRunRow {
	runs := []PurgeRunRow{}
	if limit <= 0 {
		limit = 50
	}

	db, err := e.getDatabaseHandler()
	if err != nil {
		e.logger.Error(err)
		return runs
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT id, started_at, finished_at, success, dry_run, cron_expr, repo_count, candidate_tag_count, deleted_tag_count, estimated_freed_bytes, summary, error_message, created FROM purge_runs ORDER BY id DESC LIMIT %d", limit)
	rows, err := db.Query(query)
	if err != nil {
		e.logger.Error("Error selecting purge runs: ", err)
		return runs
	}
	defer rows.Close()

	for rows.Next() {
		var row PurgeRunRow
		rows.Scan(
			&row.ID,
			&row.StartedAt,
			&row.FinishedAt,
			&row.Success,
			&row.DryRun,
			&row.CronExpr,
			&row.RepoCount,
			&row.CandidateTagCount,
			&row.DeletedTagCount,
			&row.EstimatedFreedBytes,
			&row.Summary,
			&row.ErrorMessage,
			&row.Created,
		)
		runs = append(runs, row)
	}
	return runs
}

// GetLatestPurgeRun returns the most recent purge execution log.
func (e *EventListener) GetLatestPurgeRun() *PurgeRunRow {
	runs := e.GetPurgeRuns(1)
	if len(runs) == 0 {
		return nil
	}
	return &runs[0]
}

func (e *EventListener) getDatabaseHandler() (*sql.DB, error) {
	schema := schemaSQLite
	purgeSchema := schemaPurgeRunsSQLite
	if e.databaseDriver == "sqlite3" {
		if _, err := os.Stat(e.databaseLocation); os.IsNotExist(err) {
			dir := filepath.Dir(e.databaseLocation)
			if dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0o755)
			}
		}
	}

	// Open db connection.
	db, err := sql.Open(e.databaseDriver, e.databaseLocation)
	if err != nil {
		return nil, fmt.Errorf("Error opening %s db: %s", e.databaseDriver, err)
	}

	if e.databaseDriver == "mysql" {
		schema = strings.Replace(schema, "AUTOINCREMENT", "AUTO_INCREMENT", 1)
		purgeSchema = strings.Replace(purgeSchema, "AUTOINCREMENT", "AUTO_INCREMENT", 1)
	}

	if _, err = db.Exec(schema); err != nil {
		return nil, fmt.Errorf("Error creating events table: %s", err)
	}
	if _, err = db.Exec(purgeSchema); err != nil {
		return nil, fmt.Errorf("Error creating purge_runs table: %s", err)
	}
	return db, nil
}
