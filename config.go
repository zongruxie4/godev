package godev

import (
	"errors"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"

	"gopkg.in/yaml.v3"
)

// ConfigField representa un campo de configuración editable
type ConfigField struct {
	index    int
	label    string
	name     string
	value    string
	editable bool
	selected bool
	cursor   int // Posición del cursor
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
}

var (
	config          *Config
	configErrors    []error
	configFileFound bool
	APP_ROOT_DIR    string // app root directory e.g: /home/user/go/src/github.com/user/app
)

func init() {
	config = &Config{}

	currentDir, err := os.Getwd()
	if err != nil {
		configErrors = append(configErrors, err)
	}

	// Check if current directory is a user root directory
	homeDir, _ := os.UserHomeDir()
	if currentDir == homeDir || currentDir == "/" {
		log.Fatal("Cannot run godev in user root directory. Please run in a Go project directory")
	}

	// 1 load default config
	config.DefaultConfig()

	// 2 load config from file
	if err := config.LoadConfigFromYML(); err != nil {
		configErrors = append(configErrors, err)
	} else {
		configFileFound = true
	}

	// 3 load config from params
	if err := config.LoadConfigFromParams(); err != nil {
		configErrors = append(configErrors, err)
	}

	// 4 Crear el directorio de salida si no existe
	if err := os.MkdirAll(config.OutputDir, os.ModePerm); err != nil {
		configErrors = append(configErrors, errors.New("Could not create output directory: "+err.Error()))
	}

	APP_ROOT_DIR = currentDir

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

func (c *Config) GetConfigFields() []ConfigField {
	fields := make([]ConfigField, 0)
	t := reflect.TypeOf(*c)
	v := reflect.ValueOf(c).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		label := field.Tag.Get("label")
		editable := field.Tag.Get("editable") == "true"
		value := v.Field(i).String()

		fields = append(fields, ConfigField{
			index:    i,
			label:    label,
			name:     field.Name,
			value:    value,
			editable: editable,
			selected: false,
			cursor:   len(value),
		})
	}
	return fields
}

const conFileName = "godev.yml"

func (c *Config) LoadConfigFromYML() error {
	if _, err := os.Stat(conFileName); os.IsNotExist(err) {
		return errors.New("config file: " + conFileName + " not found")
	}

	data, err := os.ReadFile(conFileName)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
}

// SaveConfigToYML guarda la configuración en un archivo YAML
func (c *Config) SaveConfigToYML() error {
	// Convierte la estructura Config a formato YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// Escribe los datos en el archivo con permisos 0644 (lectura/escritura para el propietario, solo lectura para otros)
	err = os.WriteFile(conFileName, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Observer es una interfaz para los observadores que quieren ser notificados de cambios
type Observer interface {
	OnConfigChanged(fieldName string, oldValue, newValue string)
}

// observers almacena la lista de observadores registrados
var observers []Observer

// Subscribe registra un nuevo observador para recibir notificaciones
func (c *Config) Subscribe(observer Observer) {
	observers = append(observers, observer)
}

// Unsubscribe elimina un observador de la lista
func (c *Config) Unsubscribe(observer Observer) {
	for i, obs := range observers {
		// Compara directamente las referencias de los observadores para encontrar el que queremos eliminar
		if obs == observer {
			// Cuando lo encuentra, usa slice para remover el elemento uniendo la parte antes y después del índice
			observers = append(observers[:i], observers[i+1:]...)
			break
		}
	}
}

// notifyObservers notifica a todos los observadores registrados sobre un cambio
func (c *Config) notifyObservers(fieldName, oldValue, newValue string) {
	for _, observer := range observers {
		observer.OnConfigChanged(fieldName, oldValue, newValue)
	}
}

// UpdateFieldWithNotification actualiza un campo y notifica a los observadores
func (c *Config) UpdateFieldWithNotification(field *ConfigField, newValue string) error {

	if field == nil {
		return errors.New("field cannot be nil")
	}

	oldValue := field.value
	field.value = newValue

	c.UpdateField(field.index, newValue)

	err := c.SaveConfigToYML()
	if err != nil {
		return err
	}

	// Notificar a los observadores
	c.notifyObservers(field.name, oldValue, newValue)

	return nil
}

func (c *Config) UpdateField(index int, value string) error {
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
