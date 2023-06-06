package godev

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/tdewolff/minify"
	minh "github.com/tdewolff/minify/html"
)

func (u *ui) BuildHTML() {
	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	for id_area, name_area := range u.Areas() {

		if id_area == 0 {

			page_store.StyleSheet = "static/style.css"
			page_store.Script = "static/script.js"

			page_store.UserName = ""
			page_store.UserArea = ""
			page_store.Message = ""
		} else {
			page_store.StyleSheet = "static/app.css"
			page_store.Script = "static/app.js"

			// preparamos las variables para usarlas posteriormente en el template
			page_store.UserName = `{{.UserName}}`
			page_store.UserArea = name_area
			page_store.Message = `{{.Message}}`
		}

		page_store.Menu = u.buildMenu(id_area)
		page_store.Modules = u.buildHtmlModule(id_area)

		template_html := u.makeHtmlTemplate()

		if u.AppInProduction() {
			htmlMinify(&template_html)
		}
		// crear archivo
		file_name := fmt.Sprintf(BuiltFolder+"/area_%c.html ", id_area)

		fileWrite(file_name, &template_html)

	}

}

func htmlMinify(data_in *bytes.Buffer) {

	m := minify.New()
	m.AddFunc("text/html", minh.Minify)

	var temp_result bytes.Buffer
	err := m.Minify("text/html", &temp_result, data_in)

	if err != nil {
		log.Printf("Minification HTML error: %v\n", err)
		return
	}

	data_in.Reset()
	*data_in = temp_result

}
