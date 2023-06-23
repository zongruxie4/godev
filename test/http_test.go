package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cdvelop/godev"
)

func Test_HttpUI(t *testing.T) {
	mux := http.NewServeMux()

	// // registrar app
	ui := godev.RegisterApp(App(), false, modules...)

	mux.Handle("/", ui)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Hacer petición GET al servidor con URL raíz "/"

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Error haciendo petición GET: %s", err)
	}
	defer resp.Body.Close()

	// Leer contenido de la respuesta
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error leyendo contenido de respuesta: %s", err)
	}

	// Verificar que se recibió código 200 OK
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Código de respuesta incorrecto. Esperado %d, recibido %d", http.StatusOK, resp.StatusCode)
	}

}
