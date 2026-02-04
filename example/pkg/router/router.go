package router

import (
	"net/http"

	"example/pkg/greet"
)

func RegisterRoutes(mux *http.ServeMux) {

	mux.HandleFunc("/api/greet", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(greet.Greet("API")))
	})
}
