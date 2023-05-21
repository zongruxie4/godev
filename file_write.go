package godev

import (
	"bytes"
	"io"
	"log"
	"os"
)

// file_name ej: "theme/index.html"
func fileWrite(file_name string, data *bytes.Buffer) {

	dst, err := os.Create(file_name)
	if err != nil {
		log.Fatal("Error al crear archivo", err)
	}
	defer dst.Close()

	// Copy the uploaded File to the filesystem at the specified destination
	_, err = io.Copy(dst, data)
	if err != nil {
		log.Fatalf("Error no se logro escribir el archivo %v en el destino %v", file_name, err)
		return
	}

}
