package main

import (
	"log"
	"net/http"
	"os"
)

var appProcess *os.Process

func main() {
	port := "8080" // Puerto predeterminado
	if len(os.Args) > 1 {
		port = os.Args[1] // Lee el puerto de los argumentos de línea de comandos si se proporciona
	}

	appName := getAppName() // Obtiene el nombre de la aplicación basado en el nombre de la carpeta actual

	http.HandleFunc("/restart", restartHandler)

	go func() {
		err := recompileApp()
		if err != nil {
			log.Println("Error al recompilar la aplicación:", err)
			return
		}

		err = restartApp()
		if err != nil {
			log.Println("Error al reiniciar la aplicación:", err)
			return
		}

		log.Println("Servidor escuchando en http://localhost:" + port + " App: " + appName)
		err = http.ListenAndServe(":"+port, nil)
		if err != nil {
			log.Fatal("Error al iniciar el servidor:", err)
		}
	}()

	// Espera indefinidamente
	select {}
}
