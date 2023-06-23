package godev

import (
	"os"
	"strconv"

	"github.com/cdvelop/model"
)

func RegisterApp(a app, file_watcher_start bool, modules ...*model.Module) *ui {

	ui_store.app = a
	ui_store.modules = modules

	// registrar carpetas a observar
	if len(modules) == 0 {
		showErrorAndExit("módulos no Ingresados")
	}
	for n, m := range modules {
		if m == nil {
			showErrorAndExit("módulo No: " + strconv.Itoa(n) + " es nulo")
		}

		if m.Theme != nil && m.Theme.FolderPath() != "" && ui_store.theme_folder == "" {
			ui_store.folders_watch = append(ui_store.folders_watch, m.Theme.FolderPath())
			ui_store.theme_folder = m.Theme.FolderPath()
		}

		// registrar rutas a observar módulos
		if m.Path != nil && m.Path.FolderPath() != "" {
			ui_store.folders_watch = append(ui_store.folders_watch, m.Path.FolderPath())
		}
	}

	_, err := os.Stat("modules")
	if !os.IsNotExist(err) {
		// por defecto si se encuentra la carpeta modules
		ui_store.folders_watch = append(ui_store.folders_watch, "modules")
	}

	page_store.AppName = a.AppName()
	page_store.AppVersion = a.AppVersion()

	ui_store.registerComponents()

	ui_store.checkStaticFileFolders()
	ui_store.copyStaticFilesFromUiTheme()

	ui_store.webAssemblyCheck()

	if file_watcher_start {
		ui_store.DevFileWatcherSTART()
	}

	return &ui_store
}
