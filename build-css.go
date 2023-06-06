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

	private_css := bytes.Buffer{}
	public_css := bytes.Buffer{}

	// fmt.Println(`1- comenzamos con el css del tema`)

	err := readFiles(u.FolderPath()+"/css", ".css", &private_css)
	if err != nil {
		fmt.Println(err) // si hay error es por que no hay css en el tema
	}

	// fmt.Println(`2- leer CSS publico de los componentes registrados`)
	for _, c := range u.components {
		if c.CssGlobal != nil {
			private_css.Write([]byte(c.CssGlobal.CssGlobal()))
		}
	}

	// copiamos el css a publico hasta aquí
	public_css.Write(private_css.Bytes())

	// código css privado app desde aca

	for _, c := range u.components {
		if c.CssPrivate != nil {
			private_css.Write([]byte(c.CssPrivate.CssPrivate()))
		}
	}

	// fmt.Println(`3- construir css privado`)
	for _, m := range u.Modules() {

		dir := "modules/" + m.Name + "/css"
		readFiles(dir, ".css", &private_css)

	}

	// fmt.Println("4- >>> escribiendo archivos app.css y style.css")

	if u.AppInProduction() {
		cssMinify(&private_css)
		cssMinify(&public_css)
	}

	fileWrite(StaticFolder+"/app.css", &private_css)
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
