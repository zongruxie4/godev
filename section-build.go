package godev

import (
	"path"
	"time"

	. "github.com/cdvelop/assetmin"
	"github.com/cdvelop/devwatch"
	"github.com/cdvelop/goserver"
	"github.com/cdvelop/tinywasm"
)

func (h *handler) AddSectionBUILD() {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

	// WRITERS
	wasmWriter := sectionBuild.NewWriter("WASM", false)
	serverWriter := sectionBuild.NewWriter("SERVER", false)
	assetsWriter := sectionBuild.NewWriter("ASSETS", false)
	watchWriter := sectionBuild.NewWriter("WATCH", false)
	configWriter := sectionBuild.NewWriter("CONFIG", false)

	// CONFIG
	h.config = NewAutoConfig(h.rootDir, configWriter) // Use the provided logger
	// Scan initial architecture - this must happen before AddSectionBUILD
	h.config.ScanDirectoryStructure()

	//SERVER
	h.serverHandler = goserver.New(&goserver.Config{
		RootFolder:                  h.config.GetWebFilesFolder(),
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: nil,
		ArgumentsToRunServer:        nil,
		PublicFolder:                h.config.GetPublicFolder(),
		AppPort:                     h.config.GetServerPort(),
		Writer:                      serverWriter,
		ExitChan:                    h.exitChan,
	})

	//WASM
	h.wasmHandler = tinywasm.New(&tinywasm.Config{
		WebFilesFolder: func() (string, string) {
			return h.config.GetWebFilesFolder(), h.config.GetPublicFolder()
		},
		Writer: wasmWriter,
	})

	//ASSETS
	h.assetsHandler = NewAssetMin(&AssetConfig{
		ThemeFolder: func() string {
			return path.Join(h.rootDir, h.config.GetWebFilesFolder(), "theme")
		},
		WebFilesFolder: func() string {
			return path.Join(h.rootDir, h.config.GetOutputStaticsDirectory())
		},
		Writer: assetsWriter,
		GetRuntimeInitializerJS: func() (string, error) {
			return h.wasmHandler.JavascriptForInitializing()
		},
	})

	// WATCHER
	h.watcher = devwatch.New(&devwatch.WatchConfig{
		AppRootDir:      h.config.GetRootDir(),
		FileEventAssets: h.assetsHandler,
		FilesEventGO:    []devwatch.GoFileHandler{h.serverHandler, h.wasmHandler},
		FolderEvents:    h.config, // Architecture detection for directory changes
		BrowserReload:   h.browser.Reload,
		Writer:          watchWriter,
		ExitChan:        h.exitChan,
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

	// Agregar manejadores que requieren interacción del desarrollador
	// BROWSER
	sectionBuild.AddExecutionHandler(h.browser, time.Millisecond*500)

}
