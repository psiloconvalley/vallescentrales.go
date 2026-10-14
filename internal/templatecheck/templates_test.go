package templatecheck

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"vallescentrales/internal/app"
)

// setupProjectRoot ensures the test runs with the project root as working directory
// so relative paths like "templates/" resolve correctly.
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

// TestAllTemplatesParse verifies that every .tmpl file in the repository
// parses cleanly with the production funcMap without syntax errors.
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

// ─────────────────────────────────────────────────────────────────
// TestListingDetailRender renders listing_detail.tmpl with a realistic
// fake payload and asserts that execution completes without error.
// Catches schema drift bugs where templates reference removed fields.
// ─────────────────────────────────────────────────────────────────

func TestListingDetailRender(t *testing.T) {
	// ───── PHASE 1: ARRANGE ─────
	setupProjectRoot(t)

	tmplMap, err := app.ParseTemplates(nil)
	if err != nil {
		t.Fatalf("ParseTemplates failed: %v", err)
	}

	tmpl, ok := tmplMap["listing_detail.tmpl"]
	if !ok {
		t.Fatal("listing_detail.tmpl was not found in parsed template map")
	}

	data := fakeListingDetailData()

	// Inject the CSRF token and AssetVersion that Render() normally adds,
	// so the template doesn't crash looking for them.
	data["CSRFToken"] = "fake-csrf-token-for-testing"
	data["AssetVersion"] = "test-version"

	// ───── PHASE 2: ACT ─────
	// Render into a buffer so we don't need a real HTTP response writer.
	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "base", data)

	// ───── PHASE 3: ASSERT ─────
	if err != nil {
		t.Fatalf("listing_detail.tmpl failed to render: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("render produced empty output")
	}

	// Sanity check: the rendered HTML should mention the listing title.
	if !bytes.Contains(buf.Bytes(), []byte("Casa de campo con terreno")) {
		t.Error("rendered output did not contain the listing title")
	}

	t.Logf("✅ listing_detail.tmpl rendered successfully (%d bytes)", len(output))
}

// ─────────────────────────────────────────────────────────────────
// _ ensures the httptest import survives in case we add HTTP-shaped
// render tests later. Harmless no-op reference.
// ─────────────────────────────────────────────────────────────────
var _ = httptest.NewRecorder
var _ = http.StatusOK
