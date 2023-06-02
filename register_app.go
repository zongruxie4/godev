package godev

import "net/http"

func RegisterApp(a app, mux *http.ServeMux, run_server bool) *ui {

	ui_store.app = a

	page_store.AppName = a.AppName()
	page_store.AppVersion = a.AppVersion()

	ui_store.registerComponents()

	if mux != nil {
		ui_store.http_server_mux = mux

		if run_server {
			ui_store.StartDevSERVER()
		}

	}

	return &ui_store

}
