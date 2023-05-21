package godev

type page struct {
	StyleSheet string // url ej style.css

	AppName    string
	AppVersion string

	Menu *string // según nivel

	UserName string
	UserArea string
	Message  string

	Modules string // según nivel

	Script string // url ej main.js
}

var page_store = page{
	StyleSheet: "",
	AppName:    "",
	AppVersion: "",
	Menu:       new(string),
	UserName:   "",
	UserArea:   "",
	Message:    "",
	Modules:    "",
	Script:     "",
}
