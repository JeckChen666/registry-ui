package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/quiq/registry-ui/events"
	"github.com/quiq/registry-ui/registry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type apiClient struct {
	client        *registry.Client
	eventListener *events.EventListener
}

func main() {
	var (
		a apiClient

		configFile, loggingLevel             string
		purgeTags, purgeDryRun               bool
		purgeIncludeRepos, purgeExcludeRepos string
	)
	flag.StringVar(&configFile, "config-file", "config.yml", "path to the config file")
	flag.StringVar(&loggingLevel, "log-level", "info", "logging level")

	flag.BoolVar(&purgeTags, "purge-tags", false, "purge old tags instead of running a web server")
	flag.BoolVar(&purgeDryRun, "dry-run", false, "dry-run for purging task, does not delete anything")
	flag.StringVar(&purgeIncludeRepos, "purge-include-repos", "", "comma-separated list of repos to purge tags from, otherwise all")
	flag.StringVar(&purgeExcludeRepos, "purge-exclude-repos", "", "comma-separated list of repos to skip from purging tags, otherwise none")
	flag.Parse()

	// Setup logging
	if loggingLevel != "info" {
		if level, err := logrus.ParseLevel(loggingLevel); err == nil {
			logrus.SetLevel(level)
		}
	}

	// Read config file
	viper.SetConfigName(strings.Split(filepath.Base(configFile), ".")[0])
	viper.AddConfigPath(filepath.Dir(configFile))
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error reading config file: %w", err))
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Init registry API client.
	a.client = registry.NewClient()
	a.eventListener = events.NewEventListener()

	// Execute CLI task and exit.
	if purgeTags {
		run := registry.PurgeOldTags(a.client, purgeDryRun, purgeIncludeRepos, purgeExcludeRepos)
		if err := a.eventListener.RecordPurgeRun(run); err != nil {
			logrus.Errorf("failed to record purge run: %v", err)
		}
		return
	}

	go a.client.StartBackgroundJobs()
	if _, err := registry.StartPurgeScheduler(viper.GetString("purge_tags.cron"), func() {
		run := registry.PurgeOldTags(a.client, viper.GetBool("purge_tags.dry_run"), "", "")
		if err := a.eventListener.RecordPurgeRun(run); err != nil {
			logrus.Errorf("failed to record scheduled purge run: %v", err)
		}
	}); err != nil {
		logrus.Fatalf("invalid purge cron expression %q: %v", viper.GetString("purge_tags.cron"), err)
	}

	// Template engine init.
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(loggingMiddleware())
	e.Use(recoverMiddleware())

	basePath := viper.GetString("uri_base_path")
	// Normalize base path.
	basePath = strings.Trim(basePath, "/")
	if basePath != "" {
		basePath = "/" + basePath
	}
	e.Renderer = setupRenderer(basePath)

	// Web routes.
	e.File("/favicon.ico", "static/favicon.ico")
	e.Static(basePath+"/static", "static")

	// Auth routes (no middleware).
	e.GET(basePath+"/login", loginHandler)
	e.POST(basePath+"/login", loginHandler)
	e.GET(basePath+"/logout", logoutHandler)

	// Protected routes with auth middleware.
	p := e.Group(basePath)
	p.Use(authMiddleware())
	p.GET("", a.viewCatalog)
	p.GET("/", a.viewCatalog)
	p.GET("/:repoPath", a.viewCatalog)
	p.GET("/__event-log", a.viewEventLog)
	p.GET("/__purge-log", a.viewPurgeLog)
	p.GET("/__statistics", a.viewStatistics)
	p.GET("/__options", a.viewOptions)
	p.GET("/__delete-tag", a.deleteTag)

	// Protected event listener.
	pp := e.Group("/event-receiver")
	pp.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: middleware.KeyAuthValidator(func(token string, c echo.Context) (bool, error) {
			return token == viper.GetString("event_listener.bearer_token"), nil
		}),
	}))
	pp.POST("", a.receiveEvents)

	e.Logger.Fatal(e.Start(viper.GetString("listen_addr")))
}
