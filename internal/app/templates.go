package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vallescentrales/internal/services"

	"github.com/justinas/nosurf"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type TemplateRenderer struct {
	templates    map[string]*template.Template
	assetVersion string
	currencySvc  *services.CurrencyService
}

// titleCaser is the Unicode-aware replacement for the deprecated strings.Title.
// Initialized once at package level for performance.
var titleCaser = cases.Title(language.Spanish)

func formatCommaNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	var res []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			res = append(res, ',')
		}
		res = append(res, byte(c))
	}
	return string(res)
}

func toInt64(n any) int64 {
	switch v := n.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		return 0
	}
}

func NewTemplateRenderer(currencySvc *services.CurrencyService) (*TemplateRenderer, error) {
	tr := &TemplateRenderer{
		templates:    make(map[string]*template.Template),
		assetVersion: fmt.Sprintf("%d", time.Now().Unix()),
		currencySvc:  currencySvc,
	}

	funcMap := template.FuncMap{
		"formatPrice": func(n int64) string {
			return formatCommaNumber(n)
		},
		"formatMXN": func(n any) string {
			val := toInt64(n)
			return fmt.Sprintf("$%s MXN", formatCommaNumber(val))
		},
		"formatUSD": func(n any) string {
			val := toInt64(n)
			rate := 20.0
			if currencySvc != nil && currencySvc.Rate() > 0 {
				rate = currencySvc.Rate()
			}
			usd := int64(float64(val) / rate)
			return fmt.Sprintf("$%s USD", formatCommaNumber(usd))
		},
		"convertToUSD": func(n any) string {
			val := toInt64(n)
			rate := 20.0
			if currencySvc != nil && currencySvc.Rate() > 0 {
				rate = currencySvc.Rate()
			}
			usd := int64(float64(val) / rate)
			return fmt.Sprintf("$%s USD", formatCommaNumber(usd))
		},
		"convertMXNToUSD": func(n any) float64 {
			val := float64(toInt64(n))
			rate := 20.0
			if currencySvc != nil && currencySvc.Rate() > 0 {
				rate = currencySvc.Rate()
			}
			return val / rate
		},
		"formatNumber": func(n any) string {
			return formatCommaNumber(toInt64(n))
		},
		"formatArea": func(n int) string {
			return formatCommaNumber(int64(n))
		},
		"formatDate": func(t time.Time) string {
			return t.Format("02 Jan 2006")
		},
		"timeAgo": func(t time.Time) string {
			d := time.Since(t)
			switch {
			case d < time.Minute:
				return "hace un momento"
			case d < time.Hour:
				m := int(d.Minutes())
				if m == 1 {
					return "hace 1 minuto"
				}
				return fmt.Sprintf("hace %d minutos", m)
			case d < 24*time.Hour:
				h := int(d.Hours())
				if h == 1 {
					return "hace 1 hora"
				}
				return fmt.Sprintf("hace %d horas", h)
			case d < 30*24*time.Hour:
				days := int(d.Hours() / 24)
				if days == 1 {
					return "hace 1 día"
				}
				return fmt.Sprintf("hace %d días", days)
			default:
				return t.Format("02 Jan 2006")
			}
		},
		"operationLabel": func(s string) string {
			switch strings.ToLower(s) {
			case "sale", "venta":
				return "Venta"
			case "rent", "renta":
				return "Renta"
			case "temporary_rent", "renta_temporal":
				return "Renta Temporal"
			default:
				return titleCaser.String(s)
			}
		},
		"propertyTypeLabel": func(s string) string {
			switch strings.ToLower(s) {
			case "house", "casa":
				return "Casa"
			case "apartment", "departamento":
				return "Departamento"
			case "land", "terreno":
				return "Terreno"
			case "commercial", "local_comercial":
				return "Local Comercial"
			case "office", "oficina":
				return "Oficina"
			case "ranch", "rancho":
				return "Rancho / Finca"
			default:
				return titleCaser.String(s)
			}
		},
		"statusLabel": func(s string) string {
			switch strings.ToLower(s) {
			case "active", "activo":
				return "Activo"
			case "pending", "pendiente":
				return "En Revisión"
			case "draft", "borrador":
				return "Borrador"
			case "sold", "vendido":
				return "Vendido"
			case "rented", "rentado":
				return "Rentado"
			case "archived", "archivado":
				return "Archivado"
			default:
				return titleCaser.String(s)
			}
		},
		"legalRegimeLabel": func(s string) string {
			switch strings.ToLower(s) {
			case "escritura_publica", "public_deed":
				return "Escritura Pública"
			case "ejidal", "ejido", "comunal":
				return "Propiedad Ejidal / Comunal"
			case "posesion", "possession_certificate":
				return "Constancia de Posesión"
			default:
				return titleCaser.String(s)
			}
		},
		"userTypeLabel": func(s string) string {
			switch strings.ToLower(s) {
			case "buyer", "seeker":
				return "Comprador / Inversionista"
			case "owner":
				return "Propietario Particular"
			case "agent":
				return "Asesor Inmobiliario"
			case "agency":
				return "Agencia Inmobiliaria"
			default:
				return titleCaser.String(s)
			}
		},
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"contains":  strings.Contains,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     titleCaser.String,
		"seq": func(start, end int) []int {
			var res []int
			for i := start; i <= end; i++ {
				res = append(res, i)
			}
			return res
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"derefString": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"derefInt": func(i *int) int {
			if i == nil {
				return 0
			}
			return *i
		},
		"derefBool": func(b *bool) bool {
			if b == nil {
				return false
			}
			return *b
		},
		"derefFloat": func(f *float64) float64 {
			if f == nil {
				return 0
			}
			return *f
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires even number of arguments")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict key must be string, got %T", values[i])
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
	}

	// 1. Gather all shared partials
	var partials []string
	if err := filepath.Walk("templates/partials", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".tmpl") {
			partials = append(partials, path)
		}
		return nil
	}); err != nil {
		slog.Warn("could not walk partials directory", "error", err)
	}

	mainBase := filepath.Join("templates", "base.tmpl")
	wizardBase := filepath.Join("templates", "wizard", "base.tmpl")

	// 2. Walk templates recursively
	err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Normalize to forward slashes for reliable map lookups
		rel, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}
		cleanRel := filepath.ToSlash(rel)

		// Skip base layouts and partials from standalone compilation
		if cleanRel == "base.tmpl" || cleanRel == "wizard/base.tmpl" || strings.HasPrefix(cleanRel, "partials/") {
			return nil
		}

		// Gateway hub template renders standalone
		if cleanRel == "hub.tmpl" {
			tmpl, err := template.New("hub.tmpl").Funcs(funcMap).ParseFiles(path)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
			tr.templates["hub.tmpl"] = tmpl
			return nil
		}

		// Choose the appropriate base layout
		var files []string
		if strings.HasPrefix(cleanRel, "wizard/") {
			files = append([]string{wizardBase, path}, partials...)
		} else {
			files = append([]string{mainBase, path}, partials...)
		}

		tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", path, err)
		}

		// Register under relative path and basename for flexible handler lookups
		tr.templates[cleanRel] = tmpl
		tr.templates[filepath.Base(path)] = tmpl

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	return tr, nil
}

func (tr *TemplateRenderer) Render(w http.ResponseWriter, r *http.Request, name string, data any) {
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

	// Format templateData as map to inject custom tokens dynamically
	var templateData map[string]any
	if m, ok := data.(map[string]any); ok {
		templateData = m
	} else if data != nil {
		templateData = map[string]any{
			"Data": data,
		}
	} else {
		templateData = make(map[string]any)
	}

	// 1. Inject asset cache buster
	templateData["AssetVersion"] = tr.assetVersion

	// 2. CRYPTOGRAPHICALLY INJECT CSRF TOKEN AUTOMATICALLY
	templateData["CSRFToken"] = nosurf.Token(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	// Dynamic layout auto-detection using Lookup
	targetTemplate := "base"
	if tmpl.Lookup("hub") != nil {
		targetTemplate = "hub"
	} else if tmpl.Lookup("wizard/base") != nil {
		targetTemplate = "wizard/base"
	}

	if err := tmpl.ExecuteTemplate(w, targetTemplate, templateData); err != nil {
		slog.Error("template execution failed", "name", cleanName, "target", targetTemplate, "error", err)
		http.Error(w, "internal server error during template execution", http.StatusInternalServerError)
		return
	}
}
