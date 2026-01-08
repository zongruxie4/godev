module github.com/tinywasm/app

go 1.25.2

require (
	github.com/stretchr/testify v1.11.1
	github.com/tinywasm/client v0.4.3
	github.com/tinywasm/devflow v0.0.31
	github.com/tinywasm/devtui v0.2.26
	github.com/tinywasm/goflare v0.0.42
	github.com/tinywasm/kvdb v0.0.17
)

require (
	al.essio.dev/pkg/shellescape v1.6.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/charmbracelet/bubbles v0.21.0 // indirect
	github.com/charmbracelet/bubbletea v1.3.10 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.11.3 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.14 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/chromedp/cdproto v0.0.0-20250803210736-d308e07a266d // indirect
	github.com/chromedp/chromedp v0.14.2 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/clipperhouse/displaywidth v0.6.2 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mark3labs/mcp-go v0.43.2 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tdewolff/parse/v2 v2.8.5 // indirect
	github.com/tinywasm/depfind v0.0.19 // indirect
	github.com/tinywasm/fmt v0.14.0 // indirect
	github.com/tinywasm/gobuild v0.0.22 // indirect
	github.com/tinywasm/gorun v0.0.15 // indirect
	github.com/tinywasm/time v0.2.11 // indirect
	github.com/tinywasm/unixid v0.2.13 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/zalando/go-keyring v0.2.6 // indirect
	golang.design/x/clipboard v0.7.1 // indirect
	golang.org/x/exp/shiny v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/image v0.34.0 // indirect
	golang.org/x/mobile v0.0.0-20251209145715-2553ed8ce294 // indirect
	golang.org/x/text v0.32.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/tdewolff/minify/v2 v2.24.8 // indirect
	github.com/tinywasm/assetmin v0.0.74
	github.com/tinywasm/devbrowser v0.2.7
	github.com/tinywasm/devwatch v0.0.48
	github.com/tinywasm/mcpserve v0.0.6
	github.com/tinywasm/server v0.1.21
	golang.org/x/sys v0.40.0 // indirect
)

replace github.com/tinywasm/devwatch => ../devwatch

replace github.com/tinywasm/client => ../client

replace github.com/tinywasm/server => ../server

replace github.com/tinywasm/assetmin => ../assetmin

replace github.com/tinywasm/goflare => ../goflare

replace github.com/tinywasm/devbrowser => ../devbrowser

replace github.com/tinywasm/mcpserve => ../mcpserve

replace github.com/tinywasm/devtui => ../devtui
