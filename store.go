package godev

import "github.com/cdvelop/model"

type ui struct {
	app
	//componentes registrados
	registered map[string]struct{}
	components []model.Component
	reload     chan bool
}

var ui_store = ui{
	app:        nil,
	registered: map[string]struct{}{},
	components: []model.Component{},
	reload:     make(chan bool, 1),
}
