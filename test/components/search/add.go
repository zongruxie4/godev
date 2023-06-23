package search

import "github.com/cdvelop/model"

var Add = model.Component{
	Name:        "search",
	CssGlobal:   search{},
	CssPrivate:  nil,
	JsGlobal:    search{},
	JsPrivate:   search{},
	JsListeners: search{},
}

type search struct{}

func Check() search {
	return search{}
}

func (search) CssGlobal() string {
	return ".search-test-style{background:#ff0}"
}

func (search) JsGlobal() string {
	return "console.log('función componente search global')"
}

func (search) JsPrivate() string {
	return "console.log('función componente search privado modulo: {{.ModuleName}}')"
}

func (search) JsListeners() string {
	return "btn.addEventListener('click',MySearchFunction);"
}

// esta función es solo para comparar en el test ya que se crea de forma dinámica
func (search) RemoveEventListener() string {
	return "btn.removeEventListener('click',MySearchFunction);"
}
