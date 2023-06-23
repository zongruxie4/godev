package godev

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/tdewolff/minify"
	minh "github.com/tdewolff/minify/html"
)

func (u *ui) BuildHTML() {
	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	var public_icons, private_icons string

	for _, m := range u.modules {
		public_found, private_found := m.ContainsTypeAreas()
		// fmt.Println("Modulo: ", m.Name, "tiene areas publica: ", public_found, " privada: ", private_found)

		if public_found {
			public_icons += m.BuildSpriteIcon()

		}
		if private_found {
			private_icons += m.BuildSpriteIcon()

		}
		// fmt.Println("icono publico: ", public_icons)
		// fmt.Println("icono privado: ", private_icons)

	}

	public_menu, private_menu := u.buildMenu()

	public_modules, private_modules := u.buildModules()

	// construir una pagina de nombre app.html privada y otra index.html publica

	// PUBLIC

	page_store.SpriteIcons = public_icons

	page_store.UserName = ""
	page_store.UserArea = ""
	page_store.Message = ""

	page_store.StyleSheet = "static/style.css"
	page_store.Script = "static/script.js"

	u.writeHtml(true, public_menu, public_modules)

	// PRIVATE

	page_store.SpriteIcons = private_icons

	// preparamos las variables para usarlas posteriormente en el template
	page_store.UserName = `{{.UserName}}`
	page_store.UserArea = `{{.UserArea}}`
	page_store.Message = `{{.Message}}`

	page_store.StyleSheet = "static/app.css"
	page_store.Script = "static/app.js"

	u.writeHtml(false, private_menu, private_modules)

}

func (u *ui) writeHtml(public bool, menu, modules string) {
	file_name := "/app.html"
	if public {
		file_name = "/index.html"
	}

	// si el proyecto no usa webAssembly creamos los menu y módulos
	if !u.wasm_build {
		page_store.Menu = menu
		page_store.Modules = modules
	}

	template_html := u.makeHtmlTemplate()
	htmlMinify(&template_html)
	// crear archivo app html
	fileWrite(BuiltFolder+file_name, &template_html)
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

func (u ui) makeHtmlTemplate() (html bytes.Buffer) {

	data, err := os.ReadFile(u.theme_folder + "/index.html")
	if err != nil {
		fmt.Println("THEME FOLDER: ", u.theme_folder)
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

	return
}
