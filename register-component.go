package godev

func (u *ui) registerComponentsAndObjects() {

	for _, m := range u.modules {

		for _, obj := range m.Objects {

			//los objetos (componentes) externos tienes ruta establecida los objetos internos no

			if obj.Path != nil && obj.Path.FolderPath() != "" { //objeto externo (componente)

				if _, no_exist := ui_store.comp_registered[obj.Name]; !no_exist {
					// fmt.Println("COMPONENTE: ", obj.Name, obj.Path)
					ui_store.components = append(ui_store.components, obj)

					// registrar su ubicación de la carpeta
					u.folders_watch = append(u.folders_watch, obj.FolderPath())

					ui_store.comp_registered[obj.Name] = struct{}{}

				}

			} else { // objeto interno modulo

				if _, no_exist := ui_store.obj_registered[obj.Name]; !no_exist {
					// fmt.Println("OBJETO: ", obj.Name, obj.Path)
					ui_store.objects = append(ui_store.objects, obj)

					ui_store.obj_registered[obj.Name] = struct{}{}
				}

			}

		}
	}

}
