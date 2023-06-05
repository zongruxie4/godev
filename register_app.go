package godev

func RegisterApp(a app, observe_file_change bool) *ui {

	ui_store.app = a

	page_store.AppName = a.AppName()
	page_store.AppVersion = a.AppVersion()

	ui_store.registerComponents()

	ui_store.checkStaticFileFolders()
	ui_store.copyStaticFilesFromUiTheme()

	ui_store.folders_watch = append(ui_store.folders_watch, "modules", ui_store.FolderPath())

	if observe_file_change {
		ui_store.DevFileWatcherSTART()
	}

	return &ui_store
}
