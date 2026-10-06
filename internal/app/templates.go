// internal/app/templates.go
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
type TemplateRenderer struct {
	templates    map[string]*template.Template
	assetVersion string
}

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
		"add": func(a, b int) int {
			return a + b
		},
		"formatMXN": services.FormatMXN,
		"formatUSD": services.FormatUSD,
		"convertToUSD": func(mxn float64) float64 {
			if currency == nil {
				return 0
			}
			return currency.ConvertMXNToUSD(mxn)
		},
		"formatPrice": func(amount float64, curr string) string {
			if curr == "USD" {
				return services.FormatUSD(amount)
			}
			return services.FormatMXN(amount)
		},
		"formatNumber": func(v any) string {
			switch n := v.(type) {
			case float64:
				return fmt.Sprintf("%.2f", n)
			case float32:
				return fmt.Sprintf("%.2f", n)
			case int, int64, int32:
				return fmt.Sprintf("%d", n)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		"operationLabel": func(op string) string {
			switch op {
			case "sale":
				return "Venta"
			case "rent":
				return "Renta"
			case "temporary_rental":
				return "Renta Temporal"
			default:
				return op
			}
		},
		"propertyLabel": func(pt string) string {
			switch pt {
			case "land":
				return "Terreno"
			case "house":
				return "Casa"
			case "apartment":
				return "Departamento"
			case "commercial":
				return "Local Comercial"
			case "office":
				return "Oficina"
			default:
				return pt
			}
		},
		"propertyRegimeLabel": func(r string) string {
			switch r {
			case "escritura_publica":
				return "Escritura Pública"
			case "comunal":
				return "Comunal"
			case "ejidal":
				return "Ejidal"
			default:
				return r
			}
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
		"version", version,
	)

	return &TemplateRenderer{
		templates:    templates,
		assetVersion: version,
	}, nil
}

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

	targetTemplate := "base"
	if name == "hub.tmpl" {
		targetTemplate = "hub_layout"
	}

	if err := tmpl.ExecuteTemplate(w, targetTemplate, data); err != nil {
		slog.Error("template execution failed", "name", name, "error", err)
		return
	}
}
