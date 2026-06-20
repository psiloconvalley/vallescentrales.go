// internal/app/templates.go
// Template renderer — parses all templates at startup,
// executes them safely on every request.
// Rule 15: never add a template variable without verifying
// it exists in ALL render call sites.

package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
)

// TemplateRenderer holds all parsed templates.
// Parsed once at startup — never on each request.
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewTemplateRenderer parses all templates from the templates/ directory.
// Returns an error if any template fails to parse — app will not start.
func NewTemplateRenderer() (*TemplateRenderer, error) {
	templates := make(map[string]*template.Template)

	// Base layout — included in every page template
	base := filepath.Join("templates", "base.tmpl")
	partials, err := filepath.Glob(filepath.Join("templates", "partials", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob partials: %w", err)
	}

	// Page templates — each one includes base + all partials
	pages, err := filepath.Glob(filepath.Join("templates", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob pages: %w", err)
	}

	authPages, err := filepath.Glob(filepath.Join("templates", "auth", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob auth pages: %w", err)
	}

	pages = append(pages, authPages...)

	for _, page := range pages {
		name := filepath.Base(page)

		// Skip base.tmpl itself — it is always a dependency, never rendered directly
		if name == "base.tmpl" {
			continue
		}

		// Build file list: page + base + all partials
		files := []string{page, base}
		files = append(files, partials...)

		tmpl, err := template.New(name).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("templates: failed to parse %s: %w", name, err)
		}

		templates[name] = tmpl
		slog.Debug("template parsed", "name", name)
	}

	slog.Info("templates loaded", "count", len(templates))
	return &TemplateRenderer{templates: templates}, nil
}

// Render executes a named template and writes it to the response.
// Status 500 is written if the template is not found or execution fails.
func (tr *TemplateRenderer) Render(w http.ResponseWriter, r *http.Request, name string, data any) {
	tmpl, ok := tr.templates[name]
	if !ok {
		slog.Error("template not found", "name", name)
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		slog.Error("template execution failed", "name", name, "error", err)
		// Headers already sent — cannot change status code
		// Log the error and return
		return
	}
}
