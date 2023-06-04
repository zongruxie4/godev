package godev

import (
	"fmt"
	"log"
	"net/http"
)

// endpoint ej: "http://localhost:8080/level_3.html"
func (u ui) StartDevSERVER() {

	u.http_server_mux.Handle("/", u)

	srv := &http.Server{
		Addr:    ":" + u.AppPort(),
		Handler: u.http_server_mux,
	}

	go func() {
		fmt.Println("Servidor localhost:" + u.AppPort())
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("fallo al iniciar servidor ", err)
		}
	}()

	u.DevFileWatcherSTART()

	sendTcpMessage("server_start")
}
