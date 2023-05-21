package godev

import (
	"fmt"

	"github.com/cdvelop/model"
)

func (u ui) buildMenu(required_area byte) *string {

	// construir menu según nivel acceso

	var menu string
	var index int
	for _, module := range u.Modules() {

		for _, module_area := range module.Areas {
			if module_area == required_area {
				menu += u.buildMenuButton(index, module) + "\n"
				index++
				break
			}
		}
	}

	return new(string)
}

func (u ui) buildMenuButton(index int, m model.Module) string {
	return fmt.Sprintf(u.MenuButtonTemplate(),
		m.Name, index, m.Name, m.Icon, m.Title)
}
