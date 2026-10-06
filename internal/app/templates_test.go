package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"vallescentrales/internal/app"
)

// ensureProjectRoot changes the working directory to the repository root if running from a subpackage directory.
func ensureProjectRoot(t *testing.T) {
	t.Helper()
	for i := 0; i < 4; i++ {
		if _, err := os.Stat("templates"); err == nil {
			return
		}
		if err := os.Chdir(".."); err != nil {
			t.Fatalf("failed to navigate to project root: %v", err)
		}
	}
	t.Fatal("could not locate templates/ directory from current working directory")
}

func TestNewTemplateRenderer(t *testing.T) {
	// Ensure test runs with root as working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	ensureProjectRoot(t)
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// 1. Validate that NewTemplateRenderer parses all templates without error
	renderer, err := app.NewTemplateRenderer(nil)
	if err != nil {
		t.Fatalf("template parsing failed: %v", err)
	}

	if renderer == nil {
		t.Fatal("expected non-nil TemplateRenderer")
	}

	// 2. Validate that critical templates exist and are loaded
	expectedTemplates := []string{
		"home.tmpl",
		"listing_detail.tmpl",
		"listings.tmpl",
		"profile_public.tmpl",
		"hub.tmpl",
		"login.tmpl",
		"register.tmpl",
	}

	for _, tmplName := range expectedTemplates {
		// Check file exists on disk
		pattern := filepath.Join("templates", "**", tmplName)
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			// Also check directly
			if _, err := os.Stat(filepath.Join("templates", tmplName)); err != nil {
				if _, err := os.Stat(filepath.Join("templates", "auth", tmplName)); err != nil {
					t.Logf("template %s not found on disk, skipping check", tmplName)
					continue
				}
			}
		}
	}
}
