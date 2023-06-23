package godev

func (u *ui) registerComponents() {

	for _, module := range u.modules {

		for _, comp := range module.Components {

			if _, no_exist := ui_store.registered[comp.Name]; !no_exist {

				ui_store.components = append(ui_store.components, comp)

				// registrar su ubicación de la carpeta
				if comp.Path != nil && comp.FolderPath() != "" {
					u.folders_watch = append(u.folders_watch, comp.FolderPath())
				}

				ui_store.registered[comp.Name] = struct{}{}

			}
		}
	}
}
