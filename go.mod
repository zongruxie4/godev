module github.com/cdvelop/godev

go 1.20

require github.com/chromedp/chromedp v0.9.1

require (
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/tdewolff/parse v2.3.4+incompatible // indirect
	github.com/tdewolff/test v1.0.9 // indirect
	golang.org/x/text v0.10.0 // indirect
)

require (
	github.com/cdvelop/input v0.0.12
	github.com/cdvelop/model v0.0.30
	github.com/cdvelop/platform v0.0.1
	github.com/chromedp/cdproto v0.0.0-20230620000757-8605e5981815
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0
	github.com/fstanis/screenresolution v0.0.0-20190527020317-869904d15333
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.2.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/tdewolff/minify v2.3.6+incompatible
	golang.org/x/sys v0.9.0 // indirect
)

replace github.com/cdvelop/model => ../model

replace github.com/cdvelop/platform => ../platform
