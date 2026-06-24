// internal/app/templates.go
// Template renderer — parses all templates at startup,
// executes them safely on every request.
// Injects AssetVersion on every render for cache busting.
// Registers currency helpers for consistent MXN/USD display.

package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"vallescentrales/internal/services"
)

// TemplateRenderer holds all parsed templates and the asset version.
// Parsed once at startup — never on each request.
type TemplateRenderer struct {
	templates    map[string]*template.Template
	assetVersion string
}

// NewTemplateRenderer parses all templates from the templates/ directory.
// AssetVersion changes every deploy/startup.
// CurrencyService is optional but recommended.
func NewTemplateRenderer(currency *services.CurrencyService) (*TemplateRenderer, error) {
	templates := make(map[string]*template.Template)

	base := filepath.Join("templates", "base.tmpl")
	partials, err := filepath.Glob(filepath.Join("templates", "partials", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob partials: %w", err)
	}

	pages, err := filepath.Glob(filepath.Join("templates", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob pages: %w", err)
	}

	authPages, err := filepath.Glob(filepath.Join("templates", "auth", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob auth pages: %w", err)
	}

	pages = append(pages, authPages...)

	funcs := template.FuncMap{
		"formatMXN": services.FormatMXN,
		"formatUSD": services.FormatUSD,
		"convertToUSD": func(mxn float64) float64 {
			if currency == nil {
				return 0
			}
			return currency.ConvertMXNToUSD(mxn)
		},
	}

	for _, page := range pages {
		name := filepath.Base(page)

		if name == "base.tmpl" {
			continue
		}

		files := []string{page, base}
		files = append(files, partials...)

		tmpl, err := template.New(name).Funcs(funcs).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("templates: failed to parse %s: %w", name, err)
		}

		templates[name] = tmpl
		slog.Debug("template parsed", "name", name)
	}

	version := strconv.FormatInt(time.Now().Unix(), 10)

	slog.Info("templates loaded",
		"count", len(templates),
		"asset_version", version,
	)

	return &TemplateRenderer{
		templates:    templates,
		assetVersion: version,
	}, nil
}

// Render executes a named template and writes it to the response.
func (tr *TemplateRenderer) Render(w http.ResponseWriter, r *http.Request, name string, data any) {
	tmpl, ok := tr.templates[name]
	if !ok {
		slog.Error("template not found", "name", name)
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	if m, ok := data.(map[string]any); ok {
		m["AssetVersion"] = tr.assetVersion
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		slog.Error("template execution failed", "name", name, "error", err)
		return
	}
}
