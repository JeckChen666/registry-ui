package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudyKit/jet/v6"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func TestDeleteTagDenialReturnsToRepositoryUnderBasePath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("uri_base_path", "/registry/")
	viper.Set("access_control.anyone_can_delete_tags", false)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/registry/__delete-tag?repoPath=team/app&tag=latest", nil)
	rec := httptest.NewRecorder()
	if err := (&apiClient{}).deleteTag(e.NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/registry/team/app?notice=delete_denied" {
		t.Fatalf("unexpected redirect: %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestReactPageEmbedsEscapedDataAndBasePath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("uri_base_path", "/console/")
	data := jet.VarMap{}
	data.Set("user", "</script><script>alert(1)</script>")
	req := httptest.NewRequest(http.MethodGet, "/console/", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())
	var output bytes.Buffer
	if err := setupRenderer("/console").Render(&output, "catalog.html", data, c); err != nil {
		t.Fatal(err)
	}
	html := output.String()
	if !strings.Contains(html, "/console/static/ui/app.js") || !strings.Contains(html, `\u003c/script\u003e`) || strings.Contains(html, "</script><script>alert") {
		t.Fatalf("invalid embedded page data: %s", html)
	}
}

func TestOptionsPageDoesNotExposeDataWithoutPermission(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("access_control.anyone_can_view_events", false)
	viper.Set("registry.hostname", "internal-registry.example")
	e := echo.New()
	e.Renderer = setupRenderer("")
	rec := httptest.NewRecorder()
	if err := (&apiClient{}).viewOptions(e.NewContext(httptest.NewRequest(http.MethodGet, "/__options", nil), rec)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rec.Body.String(), `"sections"`) {
		t.Fatal("configuration was exposed to an unauthorized user")
	}
}
