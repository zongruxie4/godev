package godev

func (u ui) registerComponents() {

	for _, module := range u.Modules() {

		for _, comp := range module.Components {

			if _, no_exist := ui_store.registered[comp.Name]; !no_exist {

				ui_store.components = append(ui_store.components, comp)

				ui_store.registered[comp.Name] = struct{}{}

			}
		}
	}
}
