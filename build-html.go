package godev

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/cdvelop/model"
	"github.com/tdewolff/minify"
	minh "github.com/tdewolff/minify/html"
)

var page = model.Page{
	StyleSheet:  "",
	AppName:     "",
	AppVersion:  "",
	SpriteIcons: "",
	Menu:        "",
	UserName:    "",
	UserArea:    "",
	Message:     "",
	Modules:     "",
	Script:      "",
}

func (u *ui) BuildHTML() {
	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	for index, m := range u.modules {

		// si el proyecto usa webAssembly seteamos menu y módulos
		if !u.wasm_build {
			page.Menu += m.BuildMenuButton(index) + "\n"
			page.Modules += m.BuildHtmlModule() + "\n"
		}

		page.SpriteIcons += m.BuildSpriteIcon() + "\n"

	}

	page.StyleSheet = "static/style.css"
	page.Script = "static/main.js"

	template_html := u.makeHtmlTemplate()

	htmlMinify(&template_html)
	// crear archivo app html
	fileWrite(BUILT_FOLDER+"/index.html", &template_html)
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

	err = t.Execute(&html, page)
	if err != nil {
		log.Fatal(err)
		return
	}

	return
}
