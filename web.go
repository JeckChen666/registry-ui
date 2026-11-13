package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/labstack/echo/v4"
	"github.com/quiq/registry-ui/registry"
	"github.com/spf13/viper"
)

const usernameHTTPHeader = "X-WEBAUTH-USER"

func (a *apiClient) setUserPermissions(c echo.Context) jet.VarMap {
	user := c.Request().Header.Get(usernameHTTPHeader)

	data := jet.VarMap{}
	data.Set("user", user)
	admins := viper.GetStringSlice("access_control.admins")
	data.Set("eventsAllowed", viper.GetBool("access_control.anyone_can_view_events") || registry.ItemInSlice(user, admins))
	data.Set("deleteAllowed", viper.GetBool("access_control.anyone_can_delete_tags") || registry.ItemInSlice(user, admins))
	return data
}

func (a *apiClient) viewCatalog(c echo.Context) error {
	repoPath := strings.Trim(c.Param("repoPath"), "/")
	// fmt.Println("repoPath:", repoPath)

	if repoPath == "__event-log" {
		return c.NoContent(http.StatusNotFound)
	}

	data := a.setUserPermissions(c)
	data.Set("repoPath", repoPath)
	data.Set("notice", c.QueryParam("notice"))

	showTags := false
	showImageInfo := false
	allRepoPaths := a.client.GetRepos()
	repos := []string{}
	if repoPath == "" {
		// Show all repos
		for _, r := range allRepoPaths {
			repos = append(repos, strings.Split(r, "/")[0])
		}
	} else if strings.Contains(repoPath, ":") {
		// Show image info
		showImageInfo = true
	} else {
		for _, r := range allRepoPaths {
			if r == repoPath {
				// Show tags
				showTags = true
			}
			if strings.HasPrefix(r, repoPath+"/") {
				// Show sub-repos
				r = strings.TrimPrefix(r, repoPath+"/")
				repos = append(repos, strings.Split(r, "/")[0])
			}
		}
	}

	if showImageInfo {
		// Show image info
		imageInfo, err := a.client.GetImageInfo(repoPath)
		if err != nil {
			basePath := viper.GetString("uri_base_path")
			return c.Redirect(http.StatusSeeOther, basePath)
		}
		data.Set("ii", imageInfo)
		return c.Render(http.StatusOK, "image_info.html", data)
	} else {
		// Show repos, tags or both.
		repos = registry.UniqueSortedSlice(repos)
		tags := []string{}
		areTagsReady := false
		tagsRefreshInfo := ""
		if showTags {
			tags, areTagsReady = a.client.ListTags(repoPath)
			asyncInterval := viper.GetInt("performance.tags_async_refresh_interval")
			maxCount := viper.GetInt("performance.tags_online_refresh_max_count")
			if asyncInterval == 0 {
				tagsRefreshInfo = "浏览时实时刷新标签。"
			} else if maxCount == 0 {
				tagsRefreshInfo = fmt.Sprintf("标签每 %d 分钟后台刷新一次。", asyncInterval)
			} else if len(tags) <= maxCount {
				tagsRefreshInfo = fmt.Sprintf("浏览时实时刷新标签，也会每 %d 分钟后台刷新。", asyncInterval)
			} else {
				tagsRefreshInfo = fmt.Sprintf("标签数超过 %d，使用每 %d 分钟刷新的缓存。", maxCount, asyncInterval)
			}
		}
		data.Set("repos", repos)
		data.Set("showTags", showTags)
		data.Set("isCatalogReady", a.client.IsCatalogReady())
		data.Set("areTagsReady", areTagsReady)
		data.Set("tagsRefreshInfo", tagsRefreshInfo)
		data.Set("tagCounts", a.client.SubRepoTagCounts(repoPath, repos))
		data.Set("tags", tags)
		if data["eventsAllowed"].Bool() && repoPath != "" && (len(repos) > 0 || len(tags) > 0) {
			// Do not show events in the root of catalog.
			data.Set("events", a.eventListener.GetEvents(repoPath))
		}
		return c.Render(http.StatusOK, "catalog.html", data)
	}
}

func (a *apiClient) deleteTag(c echo.Context) error {
	repoPath := c.QueryParam("repoPath")
	tag := c.QueryParam("tag")

	data := a.setUserPermissions(c)
	notice := "delete_denied"
	if data["deleteAllowed"].Bool() {
		if err := a.client.DeleteTag(repoPath, tag); err != nil {
			notice = "delete_failed"
		} else {
			notice = "deleted"
		}
	}
	basePath := strings.TrimRight(viper.GetString("uri_base_path"), "/")
	return c.Redirect(http.StatusSeeOther, basePath+"/"+strings.TrimLeft(repoPath, "/")+"?"+url.Values{"notice": {notice}}.Encode())
}

func (a *apiClient) viewPurgeLog(c echo.Context) error {
	data := a.setUserPermissions(c)
	if !data["eventsAllowed"].Bool() {
		return c.Render(http.StatusOK, "purge_log.html", data)
	}
	data.Set("purgeRuns", a.eventListener.GetPurgeRuns(100))
	return c.Render(http.StatusOK, "purge_log.html", data)
}

// viewStatistics view registry statistics.
func (a *apiClient) viewStatistics(c echo.Context) error {
	data := a.setUserPermissions(c)
	if !data["eventsAllowed"].Bool() {
		return c.Render(http.StatusOK, "statistics.html", data)
	}
	data.Set("isCatalogReady", a.client.IsCatalogReady())
	data.Set("repoCount", len(a.client.GetRepos()))
	data.Set("tagCount", a.client.GetTotalTagCount())
	data.Set("eventCount", a.eventListener.GetEventCount())
	data.Set("topRepos", a.client.GetTopReposByTagCount(10))
	catalogJobInfo, tagsJobInfo := a.client.GetJobInfo()
	data.Set("catalogJobInfo", catalogJobInfo)
	data.Set("tagsJobInfo", tagsJobInfo)
	data.Set("latestPurgeRun", a.eventListener.GetLatestPurgeRun())
	data.Set("purgeCronExpr", viper.GetString("purge_tags.cron"))
	return c.Render(http.StatusOK, "statistics.html", data)
}

type configSection struct {
	Name    string
	Options [][2]string // key, value pairs
}

// viewOptions view configuration options.
func (a *apiClient) viewOptions(c echo.Context) error {
	data := a.setUserPermissions(c)
	if !data["eventsAllowed"].Bool() {
		return c.Render(http.StatusOK, "options.html", data)
	}
	sectionLabels := map[string]string{
		"general": "常规", "performance": "后台刷新", "registry": "镜像仓库连接",
		"access_control": "访问权限", "event_listener": "事件记录", "purge_tags": "自动清理",
		"auth": "登录认证", "debug": "调试",
	}
	optionLabels := map[string]string{
		"listen_addr": "监听地址", "uri_base_path": "访问路径",
		"performance.catalog_page_size": "仓库列表分页大小", "performance.catalog_refresh_interval": "仓库刷新间隔（分钟）",
		"performance.tags_async_refresh_interval": "标签刷新间隔（分钟）", "performance.tags_online_refresh_max_count": "标签实时刷新上限",
		"registry.hostname": "仓库地址", "registry.insecure": "允许非 HTTPS 连接", "registry.username": "仓库用户名",
		"registry.password": "仓库密码", "registry.password_file": "密码文件", "registry.auth_with_keychain": "使用系统凭据",
		"access_control.anyone_can_view_events": "所有用户可查看记录", "access_control.anyone_can_delete_tags": "所有用户可删除标签", "access_control.admins": "管理员",
		"event_listener.bearer_token": "事件接收令牌", "event_listener.retention_days": "事件保留天数", "event_listener.database_driver": "事件存储类型",
		"event_listener.database_location": "事件存储位置", "event_listener.deletion_enabled": "自动删除过期事件",
		"purge_tags.cron": "清理计划", "purge_tags.dry_run": "仅模拟清理", "purge_tags.log_retention_days": "清理记录保留天数",
		"purge_tags.keep_days": "标签保留天数", "purge_tags.keep_count": "每个仓库最少保留数", "purge_tags.keep_regexp": "永久保留标签规则", "purge_tags.keep_from_file": "保留标签文件",
		"auth.enabled": "启用登录", "auth.username": "登录用户名", "auth.password": "登录密码", "auth.secret": "会话密钥",
		"auth.max_attempts": "登录失败次数上限", "auth.lockout_minutes": "登录锁定时长（分钟）", "debug.templates": "模板调试",
	}

	sensitiveKeys := map[string]bool{
		"registry.password":                true,
		"registry.password_file":           true,
		"event_listener.bearer_token":      true,
		"event_listener.database_location": true,
		"auth.password":                    true,
		"auth.secret":                      true,
	}

	settings := viper.AllSettings()
	sectionMap := map[string][][2]string{}
	for sectionName, v := range settings {
		if nested, ok := v.(map[string]interface{}); ok {
			keys := make([]string, 0, len(nested))
			for k := range nested {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				val := fmt.Sprintf("%v", nested[k])
				if enabled, ok := nested[k].(bool); ok {
					if enabled {
						val = "已启用"
					} else {
						val = "未启用"
					}
				}
				if sensitiveKeys[sectionName+"."+k] {
					val = "***"
				}
				label := optionLabels[sectionName+"."+k]
				if label == "" {
					label = k
				}
				if val == "" {
					val = "—"
				}
				sectionMap[sectionName] = append(sectionMap[sectionName], [2]string{label, val})
			}
		} else {
			label := optionLabels[sectionName]
			if label == "" {
				label = sectionName
			}
			sectionMap["general"] = append(sectionMap["general"], [2]string{label, fmt.Sprintf("%v", v)})
		}
	}

	sectionNames := make([]string, 0, len(sectionMap))
	for name := range sectionMap {
		sectionNames = append(sectionNames, name)
	}
	sort.Strings(sectionNames)

	sections := make([]configSection, 0, len(sectionNames))
	for _, name := range sectionNames {
		label := sectionLabels[name]
		if label == "" {
			label = name
		}
		sections = append(sections, configSection{Name: label, Options: sectionMap[name]})
	}

	data.Set("sections", sections)
	return c.Render(http.StatusOK, "options.html", data)
}

// receiveEvents receive events.
func (a *apiClient) receiveEvents(c echo.Context) error {
	a.eventListener.ProcessEvents(c.Request())
	return c.String(http.StatusOK, "OK")
}
