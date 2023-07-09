package godev

import (
	"bytes"
	"log"
	"time"

	"github.com/tdewolff/minify"
	minjs "github.com/tdewolff/minify/js"
)

func (u *ui) BuildJS() {
	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	public_js := bytes.Buffer{}

	// fmt.Println(`0- agregamos js por defecto`)
	public_js.WriteString("'use strict';\n")

	// fmt.Println(`1- comenzamos con el js del tema`)
	readFiles(u.theme_folder+"/js", ".js", &public_js)

	// fmt.Println(`2- leer js publico de los componentes`)

	for _, comp := range u.components {

		if comp.JsGlobal != nil {
			public_js.Write([]byte(comp.JsGlobal.JsGlobal()))
		}

		readFiles(comp.Path.FolderPath()+"/js_global", ".js", &public_js)

	}

	u.addWasmJS(&public_js)

	// **** código js módulos desde aca

	// fmt.Println(`3- construir módulos js`)
	for _, module := range u.modules {
		funtions := bytes.Buffer{}
		listener_add := bytes.Buffer{}
		listener_rem := bytes.Buffer{}

		for _, comp := range u.components {
			if comp.Module != nil && comp.Module.MainName == module.MainName {
				// 1 adjuntar funciones de componentes
				u.attachJsToModuleFromGoCode(module, comp, &funtions, &listener_add, &listener_rem)

				attachJsToModuleFromFolder(comp, module.MainName, &funtions, &listener_add, &listener_rem)

			}
		}

		for _, obj := range u.objects {

			if obj.Module != nil && obj.Module.MainName == module.MainName {

				u.attachJsToModuleFromFieldsObject(module.MainName, obj, &funtions, &listener_add, &listener_rem)

				attachJsToModuleFromFolder(obj, module.MainName, &funtions, &listener_add, &listener_rem)

			}
		}

		if module.Path != nil && module.Path.FolderPath() != "" {

			readFiles(module.Path.FolderPath()+"/js_module", ".js", &public_js)

			// fmt.Println(`agregamos js test si existiesen`)
			readFiles(module.Path.FolderPath()+"/js_test", ".js", &public_js)
		}

		// fmt.Println(`4- >>> escribiendo module JS: `, module.MainName)
		public_js.WriteString(module.BuildModuleJS(funtions.String(), listener_add.String(), listener_rem.String()))

	}

	jsMinify(&public_js)

	fileWrite(STATIC_FOLDER+"/main.js", &public_js)

}

func jsMinify(data_in *bytes.Buffer) {

	m := minify.New()
	m.AddFunc("text/javascript", minjs.Minify)

	var temp_result bytes.Buffer
	err := m.Minify("text/javascript", &temp_result, data_in)

	if err != nil {
		log.Printf("Minification JS error: %v\n", err)
		return
	}

	data_in.Reset()
	*data_in = temp_result

}
