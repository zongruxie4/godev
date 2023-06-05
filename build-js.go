package godev

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tdewolff/minify"
	minjs "github.com/tdewolff/minify/js"
)

func (u *ui) BuildJS() {
	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	private_js := bytes.Buffer{}
	public_js := bytes.Buffer{}

	// fmt.Println(`0- agregamos js por defecto`)
	private_js.WriteString("'use strict';\n")

	// fmt.Println(`1- comenzamos con el js del tema`)
	readFiles(u.FolderPath()+"/js", ".js", &private_js)

	// fmt.Println(`2- leer js publico de los componentes`)
	components_dir := "ui/components"
	files, err := os.ReadDir(components_dir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				readFiles(components_dir+"/"+file.Name()+"/js_public", ".js", &private_js)
			}
		}
	}

	for _, c := range u.components {
		if c.JsGlobal != nil {
			private_js.Write([]byte(c.JsGlobal.JsGlobal()))
		}
	}

	u.addWasmJS(&private_js)

	// fmt.Println(`2- leer js publico de los módulos`)

	for _, module := range u.Modules() {
		// fmt.Println(`2.1 leer directorio "js_public" del module si existe`)
		dir_js := "modules/" + module.Name + "/js_public"

		readFiles(dir_js, ".js", &private_js)
	}

	if !u.AppInProduction() {
		// fmt.Println(`agregamos js test si existiesen`)
		readFiles(u.FolderPath()+"/js_test", ".js", &public_js)
	}
	// copiamos el js a publico hasta aquí
	public_js.Write(private_js.Bytes())

	// código js privado desde aca

	// fmt.Println(`3- construir módulos js`)
	for _, module := range u.Modules() {
		funtions := bytes.Buffer{}
		listener_add := bytes.Buffer{}
		listener_rem := bytes.Buffer{}

		attachJsFromComponentInToModule(module, &funtions, &listener_add, &listener_rem)

		u.attachFromJsObjectsToModule(module, &funtions, &listener_add, &listener_rem)
		// adjuntar js componentes registrados en el modulo
		for _, comp := range module.Components {
			// read files an parse dir ej: ui/components/form/js_module
			path_component := "components/" + comp.Name + "/js_module"
			// fmt.Println("PATH COMPONENT: ", path_component)
			attachFromJsFolderToModule(module, path_component, &funtions, &listener_add, &listener_rem)

		}

		dir_mod_js := `modules/` + module.Name + `/js_module`
		attachFromJsFolderToModule(module, dir_mod_js, &funtions, &listener_add, &listener_rem)

		// fmt.Println(`4- >>> escribiendo module JS: `, module.MainName)
		private_js.WriteString(fmt.Sprintf(u.ModuleJsTemplate(), module.Name, module.Name,
			funtions.String(),
			listener_add.String(),
			listener_rem.String(),
		))
	}

	if u.HotReload() {
		fmt.Println(">>> UI Hot Reload Activo <<<")
		// fmt.Println(`agregamos js test si existiesen`)
		readFiles(u.FolderPath()+"/js_test", ".js", &private_js)
	}

	if u.AppInProduction() {
		jsMinify(&private_js)
		jsMinify(&public_js)
	}

	fileWrite(u.build_folder+"/static/app.js", &private_js)
	fileWrite(u.build_folder+"/static/script.js", &public_js)

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
