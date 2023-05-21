package godev

// areas ej: map[string]string{"m": "Area Medicina","d": "Area Dental",}
func RegisterApp(a app) *ui {
	ui_store.app = a

	page_store.AppName = a.AppName()
	page_store.AppVersion = a.AppVersion()

	ui_store.registerComponents()

	ui_store.buildAll()

	return &ui_store
}
