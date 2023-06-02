package test

import (
	"log"
	"testing"

	"github.com/cdvelop/godev"
	"github.com/cdvelop/godev/test/setting"
	"github.com/cdvelop/godev/test/ui/components/search"
)

func Test_BuildingUI(t *testing.T) {
	const root = "ui/built"
	const static = "ui/built/static"

	deleteFiles(root, []string{".html"})
	deleteFiles(static, []string{".js", ".css"})

	// // registrar app
	ui := godev.RegisterApp(setting.App(), nil, false)

	ui.BuildHTML()

	ui.BuildJS()

	ui.BuildCSS()

	err := findFilesWithNonZeroSize(root, []string{"area_a.html", "area_t.html", "area_v.html", "app.js", "script.js", "app.css", "style.css"})
	if err != nil {
		log.Fatal("Error:", err)
	}

	if textExists(static+"/app.css", search.Check().CssGlobal()) == 0 {
		log.Fatalln("EN app.css NO EXISTE: ", search.Check().CssGlobal())
	}

	if textExists(static+"/style.css", search.Check().CssGlobal()) == 0 {
		log.Fatalln("EN style.css NO EXISTE: ", search.Check().CssGlobal())
	}

	if textExists(static+"/app.js", search.Check().JsGlobal()) == 0 {
		log.Fatalln("EN app.js NO EXISTE: ", search.Check().JsGlobal())
	}

	if textExists(static+"/script.js", search.Check().JsGlobal()) == 0 {
		log.Fatalln("EN script.js NO EXISTE: ", search.Check().JsGlobal())
	}

	if textExists(static+"/app.js", search.Check().JsListeners()) == 0 {
		log.Fatalln("EN app.js NO EXISTE: ", search.Check().JsListeners())
	}
	// removeEventListener se crea de forma dinámica
	if textExists(static+"/app.js", search.Check().RemoveEventListener()) == 0 {
		log.Fatalln("EN app.js NO EXISTE: ", search.Check().RemoveEventListener())
	}

}
