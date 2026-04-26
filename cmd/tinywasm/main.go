package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devtui"
	twfmt "github.com/tinywasm/fmt"
	_ "github.com/tinywasm/fmt/dictionary"
	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/server"
)

var Version = "dev"

func main() {
	debugFlag := flag.Bool("debug", false, "Enable debug mode for unfiltered logs")
	mcpFlag := flag.Bool("mcp", false, "Run as MCP Daemon")
	flag.Parse()

	if err := run(*debugFlag, *mcpFlag); err != nil {
		os.Exit(1)
	}
}

func run(debug, mcpMode bool) error {
	// Initialize start directory
	startDir, err := os.Getwd()
	if err != nil {
		log.Println("Error getting current working directory:", err)
		return err
	}

	// Create a Logger instance
	logger := app.NewLogger()

	// Initialize GoMod Handler
	goModHandler := devflow.NewGoModHandler()
	// Initialize Git Handler
	gitHandler, err := devflow.NewGit()
	if err != nil {
		log.Println("Error initializing Git handler:", err)
		logger.InternalError("Error initializing Git handler:", err)
	}

	projectRoot, err := devflow.FindProjectRoot(startDir)
	if err != nil {
		// Allow empty directories through — the wizard will initialize them.
		// Non-empty dirs without go.mod are an error: tinywasm cannot work there.
		entries, _ := os.ReadDir(startDir)
		hasFiles := false
		for _, e := range entries {
			n := e.Name()
			if n != ".git" && n != ".DS_Store" {
				hasFiles = true
				break
			}
		}
		if hasFiles {
			twfmt.Println(twfmt.Translate("Directory", "Not", "Initialized"))
			return twfmt.Errf("not initialized")
		}
		// Empty dir: set projectRoot = startDir so kvdb path is consistent if wizard creates go.mod
		projectRoot = startDir
	}

	gitHandler.SetRootDir(projectRoot)
	goModHandler.SetRootDir(projectRoot)
	logger.SetRootDir(projectRoot)
	logger.SetDebug(debug)

	goModHandler.SetLog(logger.Logger)

	// Initialize DB
	db, err := kvdb.New(filepath.Join(projectRoot, ".env"), logger.Logger, &app.FileStore{})
	if err != nil {
		log.Println("Failed to initialize database:", err)
		logger.InternalError("Failed to initialize database:", err)
		return err
	}

	// Configure Bootstrap
	cfg := app.BootstrapConfig{
		StartDir:     startDir,
		McpMode:      mcpMode,
		Debug:        debug,
		Version:      Version,
		Logger:       logger,
		DB:           db,
		GitHandler:   gitHandler,
		GoModHandler: goModHandler,
		GitHubAuth:   devflow.NewGitHubAuth(),

		TuiFactory: func(clientMode bool, clientURL, apiKey string) app.TuiInterface {
			return devtui.NewTUI(&devtui.TuiConfig{
				AppName:    "TINYWASM",
				AppVersion: Version,
				Color:      devtui.DefaultPalette(),
				Logger:     func(messages ...any) { logger.Logger(messages...) },
				Debug:      debug,
				ClientMode: clientMode,
				ClientURL:  clientURL,
				APIKey:     apiKey,
			})
		},

		BrowserFactory: func(ui app.TuiInterface, exitChan chan bool) app.BrowserInterface {
			browser := devbrowser.New(ui, db, exitChan)
			browser.SetLog(func(messages ...any) { logger.Logger(messages...) })
			return browser
		},

		ServerFactory: func(exitChan chan bool, ui app.TuiInterface, browser app.BrowserInterface) app.ServerInterface {
			// We use DB to decide which server implementation to use
			serverType, err := db.Get("TINYWASM_SERVER")
			if err != nil {
				serverType = "server" // Default fallback
			}

			switch serverType {
			case "wasi":
				// Not implemented yet or just return nil/log
				return nil
			default:
				// Default Server implementation
				return server.New().
					SetLogger(logger.Logger).
					SetExitChan(exitChan).
					SetStore(db).
					SetUI(ui).
					SetOpenBrowser(browser.OpenBrowser).
					SetGitIgnoreAdd(gitHandler.GitIgnoreAdd)
			}
		},
	}

	app.Bootstrap(cfg)
	return nil
}
