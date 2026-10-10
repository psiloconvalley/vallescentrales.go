// internal/app/templates.go
package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	wizardBase := filepath.Join("templates", "wizard", "base.tmpl")

	partials, err := filepath.Glob(filepath.Join("templates", "partials", "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("templates: failed to glob partials: %w", err)
	}

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

	// Walk recursively to capture all pages in templates, templates/auth, templates/wizard etc.
	err = filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only parse files ending in .tmpl
		if info.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Skip structural base templates and partials
		if info.Name() == "base.tmpl" || strings.Contains(path, "templates/partials") {
			return nil
		}

		// Relative path identifier (e.g., "auth/login.tmpl" or "wizard/account/step1.tmpl")
		rel, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		// Determine the base template for this rendering group
		currentBase := base
		if strings.HasPrefix(rel, "wizard/") {
			currentBase = wizardBase
		}

		files := []string{path, currentBase}
		files = append(files, partials...)

		tmpl, err := template.New(info.Name()).Funcs(funcs).ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("templates: failed to parse %s: %w", rel, err)
		}

		// Register the template by its filename AND its slash-formatted relative path
		templates[info.Name()] = tmpl
		templates[filepath.ToSlash(rel)] = tmpl

		slog.Debug("template registered", "rel", rel, "name", info.Name())
		return nil
	})

	if err != nil {
		return nil, err
	}

	version := strconv.FormatInt(time.Now().Unix(), 10)

	slog.Info("templates loaded recursively",
		"count", len(templates),
		"version", version,
	)

	return &TemplateRenderer{
		templates:    templates,
		assetVersion: version,
	}, nil
}

func (tr *TemplateRenderer) Render(w http.ResponseWriter, r *http.Request, name string, data any) {
	// Clean the name suffix for compatibility
	cleanName := name
	if !strings.HasSuffix(cleanName, ".tmpl") {
		cleanName += ".tmpl"
	}

	tmpl, ok := tr.templates[cleanName]
	if !ok {
		slog.Error("template not found", "name", cleanName)
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	if m, ok := data.(map[string]any); ok {
		m["AssetVersion"] = tr.assetVersion
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	targetTemplate := "base"
	if cleanName == "hub.tmpl" {
		targetTemplate = "hub"
	}

	if err := tmpl.ExecuteTemplate(w, targetTemplate, data); err != nil {
		slog.Error("template execution failed", "name", cleanName, "error", err)
		return
	}
}
