package templatecheck

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"vallescentrales/internal/app"
)

// setupProjectRoot ensures the test runs with the project root as working
// directory so relative paths like "templates/" resolve correctly.
func setupProjectRoot(t *testing.T) {
	t.Helper()

	for i := 0; i < 5; i++ {
		if _, err := os.Stat("go.mod"); err == nil {
			return
		}
		if err := os.Chdir(".."); err != nil {
			t.Fatalf("failed to navigate to parent directory: %v", err)
		}
	}

	t.Fatal("could not locate project root (go.mod not found)")
}

// TestAllTemplatesParse verifies that every .tmpl file parses cleanly
// with the production funcMap. Catches syntax errors like missing {{end}}.
func TestAllTemplatesParse(t *testing.T) {
	setupProjectRoot(t)

	tmplMap, err := app.ParseTemplates(nil)
	if err != nil {
		t.Fatalf("expected all templates to parse cleanly, got error: %v", err)
	}

	if len(tmplMap) == 0 {
		t.Fatal("expected parsed templates map to not be empty")
	}

	criticalTemplates := []string{
		"hub.tmpl",
		"home.tmpl",
		"listings.tmpl",
		"listing_detail.tmpl",
		"dashboard.tmpl",
		"profile_edit.tmpl",
		"wizard/account/step1.tmpl",
		"wizard/account/step2.tmpl",
		"wizard/property/step1.tmpl",
		"wizard/property/step2.tmpl",
		"wizard/property/step3.tmpl",
	}

	for _, name := range criticalTemplates {
		cleanName := filepath.ToSlash(name)
		if _, ok := tmplMap[cleanName]; !ok {
			t.Errorf("critical template %q was not registered in parsed templates map", cleanName)
		}
	}

	t.Logf("✅ Successfully parsed %d registered template targets", len(tmplMap))
}

// TestCriticalTemplatesRender exercises each high-risk template with a
// realistic fake payload and asserts it renders without error. This is
// the primary defense against schema drift bugs where templates reference
// struct fields that have been renamed or removed.
//
// Each row in the table specifies:
//   - name:     human-readable subtest label
//   - template: the registered template key in the parsed map
//   - target:   the layout to execute ("base", "wizard/base", or the
//               template's own define name for standalone partials)
//   - data:     the fake payload to inject at render time
//   - mustContain: optional sanity-check substring in the output
func TestCriticalTemplatesRender(t *testing.T) {
	setupProjectRoot(t)

	tmplMap, err := app.ParseTemplates(nil)
	if err != nil {
		t.Fatalf("ParseTemplates failed: %v", err)
	}

	tests := []struct {
		name        string
		template    string
		target      string
		data        any
		mustContain string
	}{
		{
			name:        "listing_detail renders with full owner + viewer payload",
			template:    "listing_detail.tmpl",
			target:      "base",
			data:        fakeListingDetailData(),
			mustContain: "Casa de campo con terreno",
		},
		{
			name:        "dashboard renders with user and listings slice",
			template:    "dashboard.tmpl",
			target:      "base",
			data:        fakeDashboardData(),
			mustContain: "María",
		},
		{
			name:        "wizard property step1 renders with step data",
			template:    "wizard/property/step1.tmpl",
			target:      "wizard/base",
			data:        fakeWizardPropertyStep1Data(),
			mustContain: "", // no strict content assertion; render-success is enough
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, ok := tmplMap[tt.template]
			if !ok {
				t.Fatalf("template %q not found in parsed template map", tt.template)
			}

			// Inject the two values that production's Render() auto-injects.
			// Only do this if data is already a map; otherwise skip (partials
			// may receive a direct struct and have no need for them).
			if m, ok := tt.data.(map[string]any); ok {
				m["CSRFToken"] = "fake-csrf-token-for-testing"
				m["AssetVersion"] = "test-version"
			}

			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, tt.target, tt.data); err != nil {
				t.Fatalf("render failed: %v", err)
			}

			if buf.Len() == 0 {
				t.Fatal("render produced empty output")
			}

			if tt.mustContain != "" && !bytes.Contains(buf.Bytes(), []byte(tt.mustContain)) {
				t.Errorf("expected rendered output to contain %q, but it did not", tt.mustContain)
			}

			t.Logf("✅ %s rendered successfully (%d bytes)", tt.template, buf.Len())
		})
	}
}
