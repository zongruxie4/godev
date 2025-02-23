package godev

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TerminalPrint imprime un mensaje en la tui
type TerminalPrint struct {
	Content string
	Type    MessageType
	Time    time.Time
}

// TextUserInterface mantiene el estado de la aplicación
type TextUserInterface struct {
	tabsSection                []TabSection
	activeTabIndex             int
	indexActiveEditField       int  // Índice del campo de configuración seleccionado
	editingFieldValueInSection bool // Si estamos editando un campo
	messages                   []TerminalPrint
	SectionFooter              string
	currentTime                string
	width                      int
	height                     int
	messagesChan               chan TerminalPrint
	tea                        *tea.Program
	exitChan                   chan bool // Canal global para señalizar el cierre
}

type TabSection struct {
	title          string
	SectionFooter  string
	terminalPrints []TerminalPrint
	sectionFields  []SectionField // Field actions configured for the section
}

// Interface for handling tab field sectionFields
type sectionFieldAdapter interface {
	FieldNameLabelAndValue() (name, label, value string)             // eg: "port", "Server Port", "8080"
	Editable() bool                                                  // if no editable eject the action FieldValueChange directly
	FieldValueChange(newValue string) (runMessage string, err error) //eg: "8080" -> "9090" runMessage: "Port changed from 8080 to 9090"
}

// SectionField representa un campo de configuración Editable()
type SectionField struct {
	index          int
	name           string // eg: "port"
	label          string // eg: "Server Port"
	value          string // eg: "8080"
	isOpenedStatus bool
	cursor         int // Posición del cursor en la cadena de texto

	sectionFieldAdapter
}

// represent the tab section in the tui
type tuiTabSectionAdapter interface {
	SectionTitle() string // eg: "BUILD", "TEST"
	SectionFieldsAdapters() []sectionFieldAdapter
	SectionFooter() string // eg: "Press 't' to compile", "Press 'r' to run tests"
}

// tabIndexStart is the index of the tab to start
func NewTUI(tabIndexStart int, exitChan chan bool, tabsSections ...tuiTabSectionAdapter) *TextUserInterface {

	tabsSection := make([]TabSection, len(tabsSections))

	for i, section := range tabsSections {

		var tabActions []SectionField

		for i, action := range section.SectionFieldsAdapters() {

			name, label, value := action.FieldNameLabelAndValue()

			tabActions = append(tabActions, SectionField{
				index:               i,
				name:                name,
				label:               label,
				value:               value,
				isOpenedStatus:      false,
				cursor:              0,
				sectionFieldAdapter: action,
			})
		}

		tabsSection[i] = TabSection{
			title:          section.SectionTitle(),
			SectionFooter:  section.SectionFooter(),
			terminalPrints: []TerminalPrint{},
			sectionFields:  tabActions,
		}
	}

	// Create default tab if no tabs provided
	if len(tabsSection) == 0 {
		defaultTab := TabSection{
			title:          "GODEV",
			terminalPrints: []TerminalPrint{},
			sectionFields:  []SectionField{},
		}
		tabsSection = append(tabsSection, defaultTab)

		tabIndexStart = 0 // Set the default tab index to 0
	}

	tui := &TextUserInterface{
		tabsSection:    tabsSection,
		activeTabIndex: tabIndexStart,
		messagesChan:   make(chan TerminalPrint, 100),
		currentTime:    time.Now().Format("15:04:05"),
		exitChan:       exitChan,
	}

	tui.tea = tea.NewProgram(tui, tea.WithAltScreen())

	return tui
}

func NewTUIOLD() *TextUserInterface {
	// crea una nueva instancia de TextUserInterface

	// tui := &TextUserInterface{
	// 	tabsSection: []TabSection{
	// 		{
	// 			title:          "GODEV",
	// 			terminalPrints: []TerminalPrint{},
	// 			sectionFields:        GetConfigFields(),
	// 			SectionFooter:  "↑↓ to navigate | ENTER to edit | ESC to exit edit",
	// 		},
	// 		{
	// 			title:          "BUILD",
	// 			terminalPrints: []TerminalPrint{},
	// 			sectionFields: []SectionField{
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
	// 			terminalPrints: []TerminalPrint{},
	// 			sectionFields: []SectionField{
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
	// 			terminalPrints: []TerminalPrint{},
	// 			SectionFooter:  "'d' Docker | 'v' VPS Setup",
	// 			sectionFields: []SectionField{
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
	// 			terminalPrints: []TerminalPrint{},
	// 			SectionFooter:  "Press 'h' for commands list | 'ctrl+c' to Exit",
	// 		},
	// 	},
	// 	activeTabIndex:    BUILD_TAB_INDEX, // Iniciamos en BUILD
	// 	messagesChan: make(chan TerminalPrint, 100),
	// 	currentTime:  time.Now().Format("15:04:05"),
	// }

	// tui.tea = tea.NewProgram(tui, tea.WithAltScreen())

	return nil
}

func GetConfigFields() []SectionField {
	fields := make([]SectionField, 0)
	// t := reflect.TypeOf(Config{})
	// v := reflect.ValueOf(Config{}).Elem()

	// for i := 0; i < t.NumField(); i++ {
	// 	field := t.Field(i)
	// 	Label() := field.Tag.Get("Label()")
	// 	Editable() := field.Tag.Get("Editable()") == "true"
	// 	value := v.Field(i).String()
	// 	validatorType := field.Tag.Get("FieldValidate")

	// 	newField := SectionField{
	// 		index:         i,
	// 		Label():       Label(),
	// 		FieldName():   field.Name,
	// 		value:         value,
	// 		Editable():      Editable(),
	// 		cursor:        len(value),
	// 		FieldValidate: getValidationFunc(validatorType),
	// 	}

	// 	setNotifyObserver(&newField)

	// 	fields = append(fields, newField)
	// }
	return fields
}

func (cf *SectionField) SetCursorAtEnd() {
	cf.cursor = len(cf.value)
}

func setNotifyObserver(f *SectionField) {

	// h.tui.Print("setNotifyObserver: " + f.FieldName())
	// log.Println("setNotifyObserver: " + f.FieldName())
	// switch f.FieldName() {
	// case "BrowserPositionAndSize":
	// f.FieldValueChange = h.BrowserPositionAndSizeChanged

	// case "BrowserStartUrl":
	// f.FieldValueChange = h.BrowserStartUrlChanged
	// }

}
