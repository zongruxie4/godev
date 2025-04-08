package godev

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/cdvelop/devtui"
	"gopkg.in/yaml.v3"
)

type ConfigHandler struct {
	config          *Config
	configErrors    []error
	configFileFound bool
	appRootDir      string
	conFileName     string
}

type Config struct {
	AppNameField         *devtui.Field `yaml:"-"` // eg: "MyApp"
	MainFilePathField    *devtui.Field `yaml:"-"` // eg: "web/main.server.go"
	WebFilesFolderField  *devtui.Field `yaml:"-"` // eg: "web"
	ServerPortField      *devtui.Field `yaml:"-"` // eg: "8080"
	BrowserStartUrlField *devtui.Field `yaml:"-"` //eg: "/" , "/login", "/home"
	BrowserPositionField *devtui.Field `yaml:"-"` // eg: "0,0:800,600"

	fieldsByKey map[string]*devtui.Field `yaml:"-"`
}

func (h *handler) NewConfig() {
	h.ch = &ConfigHandler{
		conFileName: "godev.yml",
		config: &Config{
			fieldsByKey: make(map[string]*devtui.Field),
		},
	}

	currentDir, err := os.Getwd()
	if err != nil {
		h.ch.configErrors = append(h.ch.configErrors, err)
	}

	homeDir, _ := os.UserHomeDir()
	if currentDir == homeDir || currentDir == "/" {
		log.Fatal("Cannot run godev in user root directory. Please run in a Go project directory")
	}

	h.ch.appRootDir = currentDir
}

func (ch *ConfigHandler) InitializeFields(h *handler) {
	triggerSave := func() error {
		return ch.SaveConfigToYML()
	}

	// AppName
	ch.config.AppNameField = devtui.NewField(
		"App Name",
		"app",
		true,
		func(newValue string) (string, error) {
			if newValue == "" {
				return "", fmt.Errorf("AppName cannot be empty")
			}
			if err := triggerSave(); err != nil {
				return "", err
			}
			return "AppName updated", nil
		})
	ch.config.registerField(ch.config.AppNameField)

	// ServerPort
	ch.config.ServerPortField = devtui.NewField(
		"Server Port",
		"8080",
		true,
		func(newValue string) (string, error) {
			if _, err := strconv.Atoi(newValue); err != nil {
				return "", fmt.Errorf("port must be a number")
			}
			if err := triggerSave(); err != nil {
				return "", err
			}
			if h.serverHandler != nil {
				return h.serverHandler.RestartServer()
			}
			return "ServerPort updated (server not restarted)", nil
		})
	ch.config.registerField(ch.config.ServerPortField)

	// WebFilesFolder (no editable)
	ch.config.WebFilesFolderField = devtui.NewField(
		"Web Files Folder",
		"web",
		false,
		nil)
	ch.config.registerField(ch.config.WebFilesFolderField)

	// Cargar valores desde YAML
	if err := ch.LoadConfigFromYML(); err != nil && !os.IsNotExist(err) {
		log.Printf("Error loading config: %v", err)
		ch.configErrors = append(ch.configErrors, err)
	}
}

func (c *Config) registerField(f *devtui.Field) {
	c.fieldsByKey[f.Name()] = f
}

func (ch *ConfigHandler) LoadConfigFromYML() error {
	data, err := os.ReadFile(ch.conFileName)
	if err != nil {
		return err
	}

	yamlData := make(map[string]any)
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return err
	}

	for key, value := range yamlData {
		if field, ok := ch.config.fieldsByKey[key]; ok {
			field.SetValue(fmt.Sprintf("%v", value))
		}
	}

	ch.configFileFound = true
	return nil
}

func (ch *ConfigHandler) SaveConfigToYML() error {
	dataToSave := make(map[string]string)
	for key, field := range ch.config.fieldsByKey {
		dataToSave[key] = field.Value()
	}

	yamlData, err := yaml.Marshal(dataToSave)
	if err != nil {
		return err
	}

	return os.WriteFile(ch.conFileName, yamlData, 0644)
}

// Métodos auxiliares (mantenidos para compatibilidad)
func (c Config) OutPutStaticsDirectory() string {
	return path.Join(c.WebFilesFolderField.Value(), c.PublicFolder())
}

func (c Config) PublicFolder() string {
	return "public"
}
