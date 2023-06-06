package test

import (
	"log"
	"testing"

	"github.com/cdvelop/godev"
	"github.com/cdvelop/godev/test/components/search"
	"github.com/cdvelop/godev/test/setting"
)

func Test_BuildingUI(t *testing.T) {

	deleteFiles(godev.BuiltFolder, []string{".html"})
	deleteFiles(godev.StaticFolder, []string{".js", ".css", ".wasm"})

	// // registrar app
	ui := godev.RegisterApp(setting.App(), false)

	ui.BuildHTML()

	ui.BuildJS()

	ui.BuildCSS()

	ui.BuildWASM()

	err := findFilesWithNonZeroSize(godev.BuiltFolder, []string{"app.wasm", "area_a.html", "area_t.html", "area_v.html", "app.js", "script.js", "app.css", "style.css"})
	if err != nil {
		log.Fatal("Error:", err)
	}

	if textExists(godev.StaticFolder+"/app.css", search.Check().CssGlobal()) == 0 {
		log.Fatalln("EN app.css NO EXISTE: ", search.Check().CssGlobal())
	}

	if textExists(godev.StaticFolder+"/style.css", search.Check().CssGlobal()) == 0 {
		log.Fatalln("EN style.css NO EXISTE: ", search.Check().CssGlobal())
	}

	if textExists(godev.StaticFolder+"/app.js", search.Check().JsGlobal()) == 0 {
		log.Fatalln("EN app.js NO EXISTE: ", search.Check().JsGlobal())
	}

	if textExists(godev.StaticFolder+"/script.js", search.Check().JsGlobal()) == 0 {
		log.Fatalln("EN script.js NO EXISTE: ", search.Check().JsGlobal())
	}

	if textExists(godev.StaticFolder+"/app.js", search.Check().JsListeners()) == 0 {
		log.Fatalln("EN app.js NO EXISTE: ", search.Check().JsListeners())
	}
	// removeEventListener se crea de forma dinámica
	if textExists(godev.StaticFolder+"/app.js", search.Check().RemoveEventListener()) == 0 {
		log.Fatalln("EN app.js NO EXISTE: ", search.Check().RemoveEventListener())
	}

}
