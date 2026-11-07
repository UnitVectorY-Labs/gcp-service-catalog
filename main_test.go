package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// generationFixture uses real templates/assets with bounded, escaped source data.
// It never crawls, writes into the checkout, or uses external credentials.
func generationFixture(t *testing.T) string {
	t.Helper()
	source, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	for _, dir := range []string{"templates", "assets"} {
		if err := os.CopyFS(filepath.Join(work, dir), os.DirFS(filepath.Join(source, dir))); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range map[string]string{
		"services.json":  `[{"name":"sample.googleapis.com","title":"Sample <script>unsafe</script>","documentation":"Preserve exact & safe values","domain":"googleapis.com"}]`,
		"directory.json": `{"items":[{"id":"sample:v1beta1","name":"sample","title":"Sample API","version":"v1beta1","preferred":false,"description":"A <script>unsafe</script> description","documentationLink":"https://example.com/docs","discoveryRestUrl":"https://example.com/discovery"}]}`,
	} {
		if err := os.WriteFile(filepath.Join(work, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(work)
	t.Setenv("WEBSITE", "https://catalog.example")
	return work
}

func TestGeneratedPagesShipSelfContainedDesignAssets(t *testing.T) {
	work := generationFixture(t)
	if err := generateHTML(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"uvy-tokens.css", "uvy-theme.js", "uvy-elements.css", "LICENSE", "TABLER-ICONS-LICENSE"} {
		source, err := os.ReadFile(filepath.Join(work, "assets", "uvy-design", name))
		if err != nil {
			t.Fatal(err)
		}
		deployed, err := os.ReadFile(filepath.Join(work, "html", "uvy-design", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(source, deployed) {
			t.Errorf("deployed %s differs from pinned source", name)
		}
	}
	for _, route := range []string{"index.html", "services.html", "apis.html", "bydomain.html", "domain/domain-googleapis.com.html", "service/sample.googleapis.com.html", "api/sample_v1beta1.html"} {
		t.Run(route, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(work, "html", route))
			if err != nil {
				t.Fatal(err)
			}
			html := string(data)
			prefix := ""
			if strings.Contains(route, "/") {
				prefix = "../"
			}
			for _, want := range []string{`<html lang="en">`, `src="` + prefix + `uvy-design/uvy-theme.js"`, `href="` + prefix + `uvy-design/uvy-tokens.css"`, `data-uvy-theme-toggle`, `aria-label="Main navigation"`, `Open source on GitHub`, `MIT License`, `<svg class="icon" aria-hidden="true"`} {
				if !strings.Contains(html, want) {
					t.Errorf("%s missing %q", route, want)
				}
			}
			if strings.Index(html, `uvy-theme.js`) > strings.Index(html, `uvy-tokens.css`) {
				t.Error("theme must initialize before token styles")
			}
			for _, remoteAsset := range []string{`<script src="https://`, `<link rel="stylesheet" href="https://`} {
				if strings.Contains(html, remoteAsset) {
					t.Errorf("remote runtime dependency in %s", route)
				}
			}
			if strings.Contains(html, `<script>unsafe</script>`) {
				t.Error("source content was not escaped")
			}
			if !strings.Contains(html, `href="`+prefix+`services.html"`) {
				t.Error("existing services navigation missing")
			}
		})
	}
	api, err := os.ReadFile(filepath.Join(work, "html", "apis.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(api), "Not preferred") {
		t.Error("non-preferred version requires textual context")
	}
	if !strings.Contains(string(api), "&lt;script&gt;unsafe&lt;/script&gt;") {
		t.Error("API description did not preserve escaped source text")
	}
	for _, name := range []string{"sitemap.xml", "robots.txt"} {
		if _, err := os.Stat(filepath.Join(work, "html", name)); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerationRejectsMissingSharedRuntime(t *testing.T) {
	work := generationFixture(t)
	if err := os.Remove(filepath.Join(work, "assets", "uvy-design", "uvy-theme.js")); err != nil {
		t.Fatal(err)
	}
	if err := generateHTML(); err == nil || !strings.Contains(err.Error(), "copy shared design file uvy-theme.js") {
		t.Fatalf("expected actionable missing runtime error, got %v", err)
	}
}

func TestGenerationRejectsBlockedSharedAssetDestination(t *testing.T) {
	work := generationFixture(t)
	if err := os.MkdirAll(filepath.Join(work, "html"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "html", "uvy-design"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateHTML(); err == nil {
		t.Fatal("generation silently accepted an unusable shared asset directory")
	}
}
