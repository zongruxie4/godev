package godev

import (
	"net/http"
)

func (ui) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	http.FileServer(http.Dir("ui/built")).ServeHTTP(w, r)
}
