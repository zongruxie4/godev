package godev

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/fstanis/screenresolution"
	"gopkg.in/yaml.v3"
)

type ConfigHandler struct {
	config          *Config
	configErrors    []error
	configFileFound bool
	appRootDir      string     // app root directory e.g: /home/user/go/src/github.com/user/app
	conFileName     string     // config file name e.g: godev.yml
	observers       []Observer // observers almacena la lista de observadores registrados
}

// ConfigField representa un campo de configuración editable
type ConfigField struct {
	index    int
	label    string
	name     string
	value    string
	editable bool
	selected bool
	cursor   int // Posición del cursor
	validate func(string) error
}

type Config struct {
	// ej: app
	AppName string `yaml:"AppName" label:"App Name" value:"app" editable:"true"`
	// ej: test/app.go
	MainFilePath string `yaml:"MainFilePath" label:"Main File Path" value:"cmd/main.go" editable:"true"`
	// ej: build
	OutputDir string `yaml:"OutputDir" label:"Output Dir" value:"build" editable:"true"`
	// eg : build/app.exe
	OutPathApp string `yaml:"OutPathApp" label:"Out Path App" value:"build/app" editable:"true"`
	// ej: 8080
	ServerPort string `yaml:"ServerPort" label:"Server Port" value:"8080" editable:"true"`
	// ej: "/index.html", "/login", default: "/"
	BrowserStartUrl string `yaml:"BrowserStartUrl" label:"Browser Home Path" value:"/" editable:"true"`
	//ej: "1930,0:800,600" (when you have second monitor) default: "0,0:800,600"
	BrowserPositionAndSize string `yaml:"BrowserPositionAndSize" label:"Browser Position" value:"0,0:800,600" editable:"true"`
}

func (h *handler) NewConfig() {

	h.ch = &ConfigHandler{
		conFileName: "godev.yml",
	}

	h.ch.config = &Config{}

	currentDir, err := os.Getwd()
	if err != nil {
		h.ch.configErrors = append(h.ch.configErrors, err)
	}

	// Check if current directory is a user root directory
	homeDir, _ := os.UserHomeDir()
	if currentDir == homeDir || currentDir == "/" {
		log.Fatal("Cannot run godev in user root directory. Please run in a Go project directory")
	}

	// 1 load default config
	h.ch.config.DefaultConfig()

	// 2 load  default browser config
	if err := h.ch.config.DefaultBrowserConfig(); err != nil {
		h.ch.configErrors = append(h.ch.configErrors, err)
	}

	// 3 load config from file
	if err := h.ch.LoadConfigFromYML(); err != nil {
		h.ch.configErrors = append(h.ch.configErrors, err)
	} else {
		h.ch.configFileFound = true
	}

	// 4 load config from params
	if err := h.ch.config.LoadConfigFromParams(); err != nil {
		h.ch.configErrors = append(h.ch.configErrors, err)
	}

	// 5 Crear el directorio de salida si no existe
	if err := os.MkdirAll(h.ch.config.OutputDir, os.ModePerm); err != nil {
		h.ch.configErrors = append(h.ch.configErrors, errors.New("Could not create output directory: "+err.Error()))
	}

	h.ch.appRootDir = currentDir

}

func (c *Config) LoadConfigFromParams() error {

	// Obtener el archivo principal a compilar
	if len(os.Args) > 1 && os.Args[1] != "" {
		c.MainFilePath = os.Args[1]
	}

	if _, err := os.Stat(c.MainFilePath); errors.Is(err, os.ErrNotExist) {
		return errors.New("Main file not found: " + c.MainFilePath)
	}

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	c.OutPathApp = path.Join(c.OutputDir, c.AppName+exe_ext)

	return nil
}

func (cf *ConfigField) SetCursorAtEnd() {
	cf.cursor = len(cf.value)
}

func (c *Config) DefaultConfig() {
	t := reflect.TypeOf(*c)
	v := reflect.ValueOf(c).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := field.Tag.Get("value")

		if value != "" {
			v.Field(i).SetString(value)
		}
	}
}

func (c *Config) DefaultBrowserConfig() error {

	r := screenresolution.GetPrimary()
	if r == nil {
		return errors.New("error SetScreenSize sistema operativo no soportado")
	}

	c.SetBrowserPosition("0,0", r.Width, r.Height)
	return nil
}

func (c *Config) SetBrowserPosition(position string, width, height int) {
	c.BrowserPositionAndSize = fmt.Sprintf("%v:%d,%d", position, width, height)
}

func (c *Config) GetConfigFields() []ConfigField {
	fields := make([]ConfigField, 0)
	t := reflect.TypeOf(*c)
	v := reflect.ValueOf(c).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		label := field.Tag.Get("label")
		editable := field.Tag.Get("editable") == "true"
		value := v.Field(i).String()
		validatorType := field.Tag.Get("validate")

		fields = append(fields, ConfigField{
			index:    i,
			label:    label,
			name:     field.Name,
			value:    value,
			editable: editable,
			selected: false,
			cursor:   len(value),
			validate: getValidationFunc(validatorType),
		})
	}
	return fields
}

func (c *ConfigHandler) LoadConfigFromYML() error {
	if _, err := os.Stat(c.conFileName); os.IsNotExist(err) {
		return errors.New("config file: " + c.conFileName + " not found")
	}

	data, err := os.ReadFile(c.conFileName)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
}

// SaveConfigToYML guarda la configuración en un archivo YAML
func (c *ConfigHandler) SaveConfigToYML() error {
	// Convierte la estructura Config a formato YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// Escribe los datos en el archivo con permisos 0644 (lectura/escritura para el propietario, solo lectura para otros)
	err = os.WriteFile(c.conFileName, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Observer es una interfaz para los observadores que quieren ser notificados de cambios
type Observer interface {
	OnConfigChanged(fieldName string, oldValue, newValue string)
}

// Subscribe registra un nuevo observador para recibir notificaciones
func (c *ConfigHandler) Subscribe(observer Observer) {
	c.observers = append(c.observers, observer)
}

// Unsubscribe elimina un observador de la lista
func (c *ConfigHandler) Unsubscribe(observer Observer) {
	for i, obs := range c.observers {
		// Compara directamente las referencias de los observadores para encontrar el que queremos eliminar
		if obs == observer {
			// Cuando lo encuentra, usa slice para remover el elemento uniendo la parte antes y después del índice
			c.observers = append(c.observers[:i], c.observers[i+1:]...)
			break
		}
	}
}

// notifyObservers notifica a todos los observadores registrados sobre un cambio
func (c *ConfigHandler) notifyObservers(fieldName, oldValue, newValue string) {
	for _, observer := range c.observers {
		observer.OnConfigChanged(fieldName, oldValue, newValue)
	}
}

// UpdateFieldWithNotification actualiza un campo y notifica a los observadores
func (c *ConfigHandler) UpdateFieldWithNotification(field *ConfigField, newValue string) error {

	if field == nil {
		return errors.New("field cannot be nil")
	}

	if err := field.validate(newValue); err != nil {
		return err
	}

	oldValue := field.value
	field.value = newValue

	err := c.UpdateField(field.index, newValue)
	if err != nil {
		return err
	}

	err = c.SaveConfigToYML()
	if err != nil {
		return err
	}

	// Notificar a los observadores
	c.notifyObservers(field.name, oldValue, newValue)

	return nil
}

func (c *ConfigHandler) UpdateField(index int, value string) error {
	v := reflect.ValueOf(c).Elem()
	if index < 0 || index >= v.NumField() {
		return errors.New("invalid field index")
	}

	field := v.Field(index)
	if !field.CanSet() {
		return errors.New("field cannot be modified")
	}

	field.SetString(value)
	return nil
}

// Validation functions map
func getValidationFunc(fieldName string) func(input string) error {

	fieldName = strings.ToLower(fieldName)

	switch {
	case strings.Contains(fieldName, "port"):
		return func(input string) error {
			port, err := strconv.Atoi(input)
			if err != nil {
				return errors.New("port must be a number")
			}
			if port < 1 || port > 65535 {
				return errors.New("port must be between 1-65535")
			}
			return nil
		}
	case strings.Contains(fieldName, "url"):
		return func(input string) error {
			if !strings.HasPrefix(input, "/") {
				return errors.New("url must start with /")
			}
			return nil
		}
	case fieldName == "BrowserPositionAndSize":
		return verifyBrowserPosition

	default:
		return func(input string) error {
			if input == "" {
				return errors.New("field cannot be empty")
			}
			return nil
		}
	}
}
