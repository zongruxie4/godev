package godev

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cdvelop/gomod"
	"github.com/cdvelop/model"
)

func (a Args) RegisterFoldersPackages(current_dir string) {
	module := filepath.Base(current_dir)

	//1 leer go mod file si existe
	if cont, err := gomod.Exist(); err == nil {

		owner := gomod.GetRepositoryOwner(cont)
		if owner != "" {
			pkgs_names := gomod.GetUsedPackageNames(owner, cont)

			fmt.Println("DUEÑO REPO go.mod: ", owner, " paquetes: ", pkgs_names)

			// Retrocede al directorio paquetes
			err = os.Chdir("..")
			if err != nil {
				ShowErrorAndExit("Error al retroceder al directorio padre: " + err.Error())
				return
			}

			// Obtiene el nuevo directorio de trabajo actual
			pkgs_dir, err := os.Getwd()
			if err != nil {
				ShowErrorAndExit("Error al obtener directorio paquetes: " + err.Error())
				return
			}

			fmt.Println("directorio paquetes:", pkgs_dir)

			// Lee los elementos del directorio
			// dirEntries, err := os.ReadDir(pkgs_dir)
			// if err != nil {
			// 	ShowErrorAndExit("Error al leer el directorio: " + err.Error())
			// 	return
			// }

			// // Itera sobre los elementos del directorio
			// for _, entry := range dirEntries {
			// 	// Verifica si es un directorio
			// 	if entry.IsDir() {

			// 		fmt.Println("Directorio:", entry.Name())
			// 	}
			// }

			// for _, folder := range pkgs_names {

			// 	 filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			// 		if info.IsDir() && !contain(path) {

			// 			fmt.Println(path)

			// 		}
			// 		return nil
			// 	})
			// }

			for _, folder := range pkgs_names {
				err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						ShowErrorAndExit(err.Error())
					}
					if info.IsDir() && !contain(path) {
						fmt.Println(path)
					}
					return nil
				})
				if err != nil {
					// Aquí puedes manejar el error como desees.
					// Por ejemplo, puedes imprimirlo o devolverlo como resultado de la función.
					ShowErrorAndExit("Error en filepath.Walk: " + err.Error())
				}
			}

		} else {
			ShowErrorAndExit("No se logro obtener el dueño del repositorio del modulo " + module)
		}
	} else {
		ShowErrorAndExit("Archivo go.mod no encontrado en el directorio")
	}
}

func RegisterApp(app_name, app_version string, file_watcher_start bool, modules ...*model.Module) *ui {

	ui_store.modules = modules

	// registrar carpetas a observar
	if len(modules) == 0 {
		ShowErrorAndExit("módulos no Ingresados")
	}
	for n, m := range modules {
		if m == nil {
			ShowErrorAndExit("módulo No: " + strconv.Itoa(n) + " es nulo")
		}

		if m.Theme != nil && m.Theme.FolderPath() != "" && ui_store.theme_folder == "" {
			ui_store.packages_watch = append(ui_store.packages_watch, m.Theme.FolderPath())
			ui_store.theme_folder = m.Theme.FolderPath()
		}

		// registrar rutas a observar módulos
		if m.Path != nil && m.Path.FolderPath() != "" {
			ui_store.packages_watch = append(ui_store.packages_watch, m.Path.FolderPath())
		}
	}

	_, err := os.Stat("modules")
	if !os.IsNotExist(err) {
		// por defecto si se encuentra la carpeta modules
		ui_store.packages_watch = append(ui_store.packages_watch, "modules")
	}

	page.AppName = app_name
	page.AppVersion = app_version

	ui_store.registerComponentsAndObjects()

	ui_store.checkStaticFileFolders()
	ui_store.copyStaticFilesFromUiTheme()

	ui_store.webAssemblyCheck()

	ui_store.compilerCheck()

	if file_watcher_start {
		ui_store.DevFileWatcherSTART()
	}

	return &ui_store
}
