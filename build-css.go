package godev

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/tdewolff/minify"
	mincss "github.com/tdewolff/minify/css"
)

func (u ui) BuildCSS() {
	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	public_css := bytes.Buffer{}

	// fmt.Println(`1- comenzamos con el css del tema`)
	err := readFiles(u.theme_folder+"/css", ".css", &public_css)
	if err != nil {
		fmt.Println(err) // si hay error es por que no hay css en el tema
	}

	for _, c := range u.components {
		if c.Css != nil {
			public_css.Write([]byte(c.Css.Css()))
		}

		if c.Path != nil {
			readFiles(c.Path.FolderPath()+"/css", ".css", &public_css)
		}

	}

	for _, c := range u.objects {
		if c.Css != nil {
			public_css.Write([]byte(c.Css.Css()))
		}

		if c.Path != nil {
			readFiles(c.Path.FolderPath()+"/css", ".css", &public_css)
		}
	}

	// fmt.Println("4- >>> escribiendo archivos app.css y style.css")
	cssMinify(&public_css)

	fileWrite(StaticFolder+"/style.css", &public_css)

}

func cssMinify(data_in *bytes.Buffer) {

	m := minify.New()
	m.AddFunc("text/css", mincss.Minify)

	var temp_result bytes.Buffer
	err := m.Minify("text/css", &temp_result, data_in)

	if err != nil {
		log.Printf("Minification CSS error: %v\n", err)
		return
	}

	data_in.Reset()
	*data_in = temp_result

}
