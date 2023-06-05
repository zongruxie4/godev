package godev

import (
	"fmt"
	"log"
	"net/http"
)

// endpoint ej: "http://localhost:8080/level_3.html"
func (a Args) StartDevSERVER() {

	mux := http.NewServeMux()

	mux.Handle("/", ui_store)

	srv := &http.Server{
		Addr:    ":1234",
		Handler: mux,
	}

	go func() {
		fmt.Println("Static Dev File Server localhost:1234")
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("fallo al iniciar servidor ", err)
		}
	}()

}
