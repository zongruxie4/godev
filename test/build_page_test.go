package test

import (
	"log"
	"testing"

	"github.com/cdvelop/godev"
	"github.com/cdvelop/godev/test/components/search"
	"github.com/cdvelop/gotools"
)

func Test_BuildingUI(t *testing.T) {

	deleteFiles(godev.BuiltFolder, []string{".html"})
	deleteFiles(godev.StaticFolder, []string{".js", ".css", ".wasm"})

	// registrar app
	ui := godev.RegisterApp("app", "0.0.0", false, modules...)

	ui.BuildHTML()

	ui.BuildJS()

	ui.BuildCSS()

	ui.BuildWASM()

	err := gotools.FindFilesWithNonZeroSize(godev.BuiltFolder, []string{"index.html", "style.css", "main.js", "app.wasm"})
	if err != nil {
		log.Fatal("Error:", err)
	}

	if textExists(godev.StaticFolder+"/style.css", search.Check().Css()) == 0 {
		log.Fatalln("EN style.css NO EXISTE: ", search.Check().Css())
	}

	if textExists(godev.StaticFolder+"/main.js", search.Check().JsGlobal()) == 0 {
		log.Fatalln("EN main.js NO EXISTE: ", search.Check().JsGlobal())
	}

	if textExists(godev.StaticFolder+"/main.js", search.Check().JsFunctionsExpected()) == 0 {
		log.Fatalln("EN main.js NO EXISTE: ", search.Check().JsFunctionsExpected())
	}

	if textExists(godev.StaticFolder+"/main.js", search.Check().JsListeners()) == 0 {
		log.Fatalln("EN main.js NO EXISTE: ", search.Check().JsListeners())
	}
	// removeEventListener se crea de forma dinámica
	if textExists(godev.StaticFolder+"/main.js", search.Check().RemoveEventListener()) == 0 {
		log.Fatalln("EN main.js NO EXISTE: ", search.Check().RemoveEventListener())
	}

	//comprobar símbolos svg en html
	if textExists(godev.BuiltFolder+"/index.html", info_module.Icon.Id) == 0 {
		log.Fatalln("EN index.html NO SE CREO EL SÍMBOLO SVG ID : ", info_module.Icon.Id)
	}

	if textExists(godev.BuiltFolder+"/index.html", info_module.Icon.Id) > 1 {
		log.Fatalln("EN index.html icono repetido SÍMBOLO SVG ID : ", info_module.Icon.Id)
	}

	if textExists(godev.BuiltFolder+"/app.html", product_module.Icon.Id) == 0 {
		log.Fatalln("EN app.html NO SE CREO EL SÍMBOLO SVG ID : ", product_module.Icon.Id)
	}

	if textExists(godev.BuiltFolder+"/app.html", product_module.Icon.Id) > 1 {
		log.Fatalln("EN app.html icono repetido SÍMBOLO SVG ID : ", product_module.Icon.Id)
	}

}
