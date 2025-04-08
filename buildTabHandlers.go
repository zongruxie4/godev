package godev

import (
	"path"

	. "github.com/cdvelop/devtui"
)

func (h *handler) BuildTabHandlers() {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	sectionBuild := TabSection{
		Title: "BUILD",
		FieldHandlers: []FieldHandler{
			{},
		},
	}

	//SERVER
	h.serverHandler = NewServerHandler(&ServerConfig{
		RootFolder:                  h.ch.config.WebFilesFolder,
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: nil,
		ArgumentsToRunServer:        nil,
		PublicFolder:                h.ch.config.PublicFolder(),
		AppPort:                     h.ch.config.ServerPort,
		Writer:                      &sectionBuild,
		ExitChan:                    h.exitChan,
	})

	// SERVER PORT AFFECTED : SERVER, BROWSER CONFIG HANDLERS
	PortFieldHandler := FieldHandler{
		Name:     "Server Port",
		Value:    h.ch.config.ServerPort,
		Editable: true,
		FieldValueChange: func(newValue string) (outMessage string, err error) {
			// STORAGE CONFIG
			h.ch.config.ServerPort = newValue

			// RESTART BROWSER
			err = h.browser.Reload()
			if err != nil {
				return "", err
			}

			return h.serverHandler.RestartServer()
		},
	}
	sectionBuild.FieldHandlers = append(sectionBuild.FieldHandlers, PortFieldHandler)

	//WASM
	h.wasmHandler = NewWasmCompiler(&WasmConfig{
		WebFilesFolder: func() (string, string) {
			return h.ch.config.WebFilesFolder, h.ch.config.PublicFolder()
		},
		Writer: &sectionBuild,
	})

	//ASSETS
	h.assetsHandler = NewAssetsCompiler(&AssetsConfig{
		ThemeFolder: func() string {
			return path.Join(h.ch.config.WebFilesFolder, "theme")
		},
		WebFilesFolder: h.ch.config.OutPutStaticsDirectory,
		Print:          h.tui.Print,
		JavascriptForInitializing: func() (string, error) {
			return h.wasmHandler.JavascriptForInitializing()
		},
	})

	// WATCHER
	h.watcher = NewWatchHandler(&WatchConfig{
		AppRootDir:      h.ch.appRootDir,
		FileEventAssets: h.assetsHandler,
		FileEventGO:     h.serverHandler,
		FileEventWASM:   h.wasmHandler,
		FileTypeGO: GoFileType{
			FrontendPrefix: []string{"f."},
			FrontendFiles: []string{
				h.wasmHandler.mainOutputFile,
			},
			BackendPrefix: []string{"b."},
			BackendFiles: []string{
				h.serverHandler.mainFileExternalServer,
			},
		},
		BrowserReload: h.browser.Reload,

		Writer:   &sectionBuild,
		ExitChan: h.exitChan,
		UnobservedFiles: func() []string {

			uf := []string{
				".git",
				".gitignore",
				".vscode",
				".exe",
				".log",
				"_test.go",
			}

			uf = append(uf, h.assetsHandler.UnobservedFiles()...)
			uf = append(uf, h.wasmHandler.UnobservedFiles()...)
			uf = append(uf, h.serverHandler.UnobservedFiles()...)
			return uf
		},
	})

	h.tui.AddTabSections()

}

// crea una nueva instancia de DevTUI

// tui := &DevTUI{
// 	tabsSection: []TabSection{
// 		{
// 			title:          "GODEV",
// 			tabContents: []TabContent{},
// 			sectionFields:        GetConfigFields(),
// 			SectionFooter:  "↑↓ to navigate | ENTER to edit | ESC to exit edit",
// 		},
// 		{
// 			title:          "BUILD",
// 			tabContents: []TabContent{},
// 			sectionFields: []FieldHandler{
// 				{
// 					Label:     "TinyGo compiler",
// 					isOpenedStatus:  false,
// 					ShortCut: "t",
// 					FieldValueChange: func() error {
// 						// TinyGo compilation logic
// 						return nil
// 					},
// 				},
// 				{
// 					Label:        "Web Browser",
// 					isOpenedStatus:     false,
// 					ShortCut:    "w",
// 					FieldValueChange:  h.OpenBrowser,
// 					closeHandler: h.CloseBrowser,
// 				},
// 			},
// 		},
// 		{
// 			title:          "TEST",
// 			tabContents: []TabContent{},
// 			sectionFields: []FieldHandler{
// 				{
// 					Label:     "Running tests...",
// 					isOpenedStatus:  false,
// 					ShortCut: "r",
// 					FieldValueChange: func() error {
// 						// Implement test running logic
// 						return nil
// 					},
// 				},
// 			},
// 		},
// 		{
// 			title:          "DEPLOY",
// 			tabContents: []TabContent{},
// 			SectionFooter:  "'d' Docker | 'v' VPS Setup",
// 			sectionFields: []FieldHandler{
// 				{
// 					Label:     "Generating Dockerfile...",
// 					isOpenedStatus:  false,
// 					ShortCut: "d",
// 					FieldValueChange: func() error {
// 						// Implement Docker generation logic
// 						return nil
// 					},
// 				},
// 				{
// 					Label:     "Configuring VPS...",
// 					isOpenedStatus:  false,
// 					ShortCut: "v",
// 					FieldValueChange: func() error {
// 						// Implement VPS configuration logic
// 						return nil
// 					},
// 				},
// 			},
// 		},
// 		{
// 			title:          "HELP",
// 			tabContents: []TabContent{},
// 			SectionFooter:  "Press 'h' for commands list | 'ctrl+c' to Exit",
// 		},
// 	},
// 	activeTab:    BUILD_TAB_INDEX, // Iniciamos en BUILD

// }
