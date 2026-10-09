package slug

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

var nonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

func Generate(s string) string {
	return Make(s)
}

func Make(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ä', 'â':
			b.WriteRune('a')
		case 'é', 'è', 'ë', 'ê':
			b.WriteRune('e')
		case 'í', 'ì', 'ï', 'î':
			b.WriteRune('i')
		case 'ó', 'ò', 'ö', 'ô':
			b.WriteRune('o')
		case 'ú', 'ù', 'ü', 'û':
			b.WriteRune('u')
		case 'ñ':
			b.WriteRune('n')
		case 'ç':
			b.WriteRune('c')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' {
				b.WriteRune(r)
			}
		}
	}

	slug := nonAlpha.ReplaceAllString(b.String(), "-")
	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = "propiedad"
	}

	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err == nil {
		slug = slug + "-" + hex.EncodeToString(suffix)
	}

	return slug
}
