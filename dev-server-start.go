package godev

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
)

// endpoint ej: "http://localhost:8080/level_3.html"
func (u ui) StartDevSERVER(endpoint string, mux *http.ServeMux) {

	mux.Handle("/", u)

	parsedUrl, err := url.Parse(endpoint)
	if err != nil {
		panic(err)
	}

	port := parsedUrl.Port()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		fmt.Println("Servidor: localhost:" + port)
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("fallo al iniciar servidor ", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go u.devBrowserSTART(endpoint, &wg)

	u.DevFileWatcherSTART(&wg)

	wg.Wait()
}
