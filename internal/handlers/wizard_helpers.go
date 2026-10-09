package handlers

import (
	"net/http"
	"net/url"

	"vallescentrales/internal/middleware"
	"vallescentrales/internal/models"
)

// getUser retrieves the authenticated user from the HTTP Request.
func getUser(r *http.Request) *models.User {
	if r == nil {
		return nil
	}
	return middleware.UserFromContext(r.Context())
}

// formToMap converts multi-value post parameters to a serialized step mapping.
func formToMap(form url.Values) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range form {
		if len(v) == 1 {
			out[k] = v[0]
		} else if len(v) > 1 {
			out[k] = v
		}
	}
	return out
}

func stringVal(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolVal(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
		if s, ok := v.(string); ok {
			return s == "1" || s == "true" || s == "on"
		}
	}
	return false
}

func stringSliceVal(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok && v != nil {
		if sl, ok := v.([]string); ok {
			return sl
		}
		if ifaceSl, ok := v.([]interface{}); ok {
			res := make([]string, 0, len(ifaceSl))
			for _, item := range ifaceSl {
				if s, ok := item.(string); ok {
					res = append(res, s)
				}
			}
			return res
		}
		if s, ok := v.(string); ok && s != "" {
			return []string{s}
		}
	}
	return nil
}
