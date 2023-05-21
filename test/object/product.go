package object

import (
	"github.com/cdvelop/input"
	"github.com/cdvelop/model"
)

var Product = model.Object{
	Name:           "",
	TextFieldNames: []string{},
	Fields: []model.Field{
		{Name: "name", Legend: "Nombre", Input: input.Text()},
		{Name: "mail", Legend: "Nombre", Input: input.Text()},
	},
}
