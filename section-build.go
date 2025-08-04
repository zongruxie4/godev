package godev

import (
	"path"

	. "github.com/cdvelop/assetmin"
	"github.com/cdvelop/tinywasm"
)

func (h *handler) AddSectionBUILD() {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

	//SERVER
	h.serverHandler = NewServerHandler(&ServerConfig{
		RootFolder:                  h.config.GetWebFilesFolder(),
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: nil,
		ArgumentsToRunServer:        nil,
		PublicFolder:                h.config.GetPublicFolder(),
		AppPort:                     h.config.GetServerPort(),
		Writer:                      sectionBuild,
		ExitChan:                    h.exitChan,
	})

	//WASM
	h.wasmHandler = tinywasm.New(&tinywasm.Config{
		WebFilesFolder: func() (string, string) {
			return h.config.GetWebFilesFolder(), h.config.GetPublicFolder()
		},
		Writer: sectionBuild,
	})

	//ASSETS
	h.assetsHandler = NewAssetMin(&AssetConfig{
		ThemeFolder: func() string {
			return path.Join(h.config.GetWebFilesFolder(), "theme")
		},
		WebFilesFolder: h.config.GetOutputStaticsDirectory,
		Writer:         sectionBuild,
		GetRuntimeInitializerJS: func() (string, error) {
			return h.wasmHandler.JavascriptForInitializing()
		},
	})

	// WATCHER
	h.watcher = NewWatchHandler(&WatchConfig{
		AppRootDir:      h.config.GetRootDir(),
		FileEventAssets: h.assetsHandler,
		FileEventGO:     h.serverHandler,
		FileEventWASM:   h.wasmHandler,
		FolderEvents:    h.config, // Architecture detection for directory changes
		FileTypeGO: GoFileType{
			FrontendPrefix: []string{"f."}, FrontendFiles: []string{
				h.wasmHandler.MainOutputFile(),
			},
			BackendPrefix: []string{"b."},
			BackendFiles: []string{
				h.serverHandler.mainFileExternalServer,
			},
		},
		BrowserReload: h.browser.Reload,
		Writer:        sectionBuild,
		ExitChan:      h.exitChan,
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

	// Agregar fields usando la nueva API de DevTUI
	sectionBuild.
		AddEditHandler(serverFieldHandler)

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
