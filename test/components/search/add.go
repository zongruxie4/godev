package search

import (
	"path/filepath"
	"runtime"

	"github.com/cdvelop/model"
)

type search struct{}

func Add() *model.Object {

	s := search{}

	return &model.Object{
		Name: "search",
		FrontendStaticCode: model.FrontendStaticCode{
			Css:         s,
			JsGlobal:    s,
			JsFunctions: s,
			JsListeners: s,
		},
		Path: s,
	}
}

func (search) FolderPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.ToSlash(dir)
}

func Check() search {
	return search{}
}

func (search) Css() string {
	return ".search-test-style{background:#ff0}"
}

func (search) JsGlobal() string {
	return "console.log('función componente search global')"
}

func (s search) JsFunctions() string {
	return "console.log('función componente search modulo: {{.ModuleName}}')"
}

func (s search) JsFunctionsExpected() string {
	return "console.log('función componente search modulo: module_product')"
}

func (search) JsListeners() string {
	return "btn.addEventListener('click',MySearchFunction);"
}

// esta función es solo para comparar en el test ya que se crea de forma dinámica
func (search) RemoveEventListener() string {
	return "btn.removeEventListener('click',MySearchFunction);"
}
