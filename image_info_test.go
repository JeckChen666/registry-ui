package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/spf13/viper"
)

func TestImageInfoRendersDeleteButtonForArtifactDetails(t *testing.T) {
	viper.Set("registry.hostname", "registry.example.com")

	testCases := []struct {
		name string
		ii   map[string]interface{}
	}{
		{
			name: "image index",
			ii: map[string]interface{}{
				"IsImage":        false,
				"IsImageIndex":   true,
				"ImageRefRepo":   "demo/app",
				"ImageRefTag":    "latest",
				"ImageRefDigest": "sha256:abc123",
				"MediaType":      "application/vnd.oci.image.index.v1+json",
				"Platforms":      "linux/amd64, linux/arm64",
				"Manifest": map[string]interface{}{
					"manifests": []interface{}{},
				},
			},
		},
		{
			name: "image",
			ii: map[string]interface{}{
				"IsImage":        true,
				"IsImageIndex":   false,
				"ImageRefRepo":   "demo/app",
				"ImageRefTag":    "latest",
				"ImageRefDigest": "sha256:def456",
				"MediaType":      "application/vnd.oci.image.manifest.v1+json",
				"Platforms":      "linux/amd64",
				"ImageSize":      int64(12345),
				"Created":        time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC),
				"ConfigImageID":  "abcdef123456",
				"Manifest": map[string]interface{}{
					"layers": []interface{}{},
				},
				"ConfigFile": map[string]interface{}{
					"config": map[string]interface{}{},
					"rootfs": map[string]interface{}{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			renderer := setupRenderer("")
			data := jet.VarMap{}
			data.Set("deleteAllowed", true)
			data.Set("eventsAllowed", false)
			data.Set("user", "")
			data.Set("repoPath", "demo/app:latest")
			data.Set("ii", tc.ii)

			var out bytes.Buffer
			if err := renderer.Render(&out, "image_info.html", data, nil); err != nil {
				t.Fatalf("render image_info.html: %v", err)
			}

			html := out.String()
			if !strings.Contains(html, "__delete-tag?repoPath=demo/app&tag=latest") {
				t.Fatalf("expected delete action for %s, got html: %s", tc.name, html)
			}
		})
	}
}
