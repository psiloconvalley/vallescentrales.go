package handlers

import "net/http"

type renderer interface {
	Render(w http.ResponseWriter, name string, data any) error
}
