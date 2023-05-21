package product_test

import (
	"github.com/cdvelop/godev/test/ui/components/search"
	"github.com/cdvelop/model"
)

var Add = model.Module{
	Name:       "product_test",
	Title:      "Productos TEST",
	Icon:       "icon-products",
	Areas:      []byte{'a', 't'},
	Components: []model.Component{search.Add},
	Objects:    []model.Object{},
}
