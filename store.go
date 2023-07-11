package godev

import (
	"github.com/cdvelop/model"
)

type ui struct {
	theme_folder string
	//módulos registrados
	modules []*model.Module

	//componentes registrados externos al modulo
	comp_registered map[string]struct{}
	components      []*model.Object

	// objetos registrados propios del modulo
	obj_registered map[string]struct{}
	objects        []*model.Object

	packages_watch []string // ej: "modules", "ui\\theme"

	wasm_build  bool
	with_tinyGo bool
}

const WORK_FOLDER = "frontend"
const BUILT_FOLDER = "frontend/built"
const STATIC_FOLDER = "frontend/built/static"

var ui_store = ui{
	theme_folder:    "",
	modules:         []*model.Module{},
	comp_registered: map[string]struct{}{},
	components:      []*model.Object{},
	obj_registered:  map[string]struct{}{},
	objects:         []*model.Object{},
	packages_watch:  []string{},
	with_tinyGo:     false,
}
