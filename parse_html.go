package godev

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"
)

func (u ui) makeHtmlTemplate() (html bytes.Buffer) {

	data, err := os.ReadFile(u.PathTemplateIndexHTML())
	if err != nil {
		fmt.Println("Error al leer el archivo:", err)
	}
	t, err := template.New("").Parse(string(data))
	if err != nil {
		log.Fatal(err)
		return
	}

	err = t.Execute(&html, page_store)
	if err != nil {
		log.Fatal(err)
		return
	}

	page_store.UserArea = ""

	return
}
