package main

import (
	"log"
	"net/http"
)

func restartHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Se recibió una señal de reinicio. Reiniciando la aplicación...")

	err := recompileApp()
	if err != nil {
		log.Println("Error al recompilar la aplicación:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al reiniciar la aplicación"))
		return
	}

	err = restartApp()
	if err != nil {
		log.Println("Error al reiniciar la aplicación:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al reiniciar la aplicación"))
		return
	}

	log.Println("La aplicación se reinició correctamente.")
	w.Write([]byte("La aplicación se reinició correctamente"))
}
