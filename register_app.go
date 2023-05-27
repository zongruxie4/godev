package godev

// areas ej: map[string]string{"m": "Area Medicina","d": "Area Dental",}
func RegisterApp(a app) *ui {
	ui_store.app = a

	page_store.AppName = a.AppName()
	page_store.AppVersion = a.AppVersion()

	ui_store.registerComponents()

	return &ui_store
}
