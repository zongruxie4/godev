package godev

import (
	"errors"
	"os"
	"reflect"

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
	Port         string `yaml:"Port" label:"Server Port" value:"9090" editable:"true"`
	OutputDir    string `yaml:"OutputDir" label:"Output Dir" value:"build" editable:"true"`
	MainFilePath string `yaml:"MainFilePath" label:"Main File" value:"cmd/main.go" editable:"true"`
}

var (
	config       *Config
	configErrors []error
)

func init() {
	config = &Config{}
	// 1 load default config
	config.DefaultConfig()
	// 2 load config from file
	if err := config.LoadConfigFromYML(); err != nil {
		configErrors = append(configErrors, err)
	}

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
