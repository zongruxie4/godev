package godev

import (
	"net/http"
)

func (u ui) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	http.FileServer(http.Dir(BuiltFolder)).ServeHTTP(w, r)
}
