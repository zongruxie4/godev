package godev

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"errors"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// channelMsg es un tipo especial para mensajes del canal
type channelMsg TabContent

// Print representa un mensaje de actualización
type tickMsg time.Time

// TabContent imprime un mensaje en la tui
type TabContent struct {
	Content string
	Type    MessageType
	Time    time.Time
}

// TextUserInterface mantiene el estado de la aplicación
type TextUserInterface struct {
	*TuiConfig
	*TuiStyle

	ready    bool
	viewport viewport.Model

	tabsSection                []TabSection
	activeTab                  int
	editingFieldValueInSection bool // Si estamos editando un campo
	SectionFooter              string
	currentTime                string
	tabContentsChan            chan TabContent
	tea                        *tea.Program
}

type TabSection struct {
	title                string
	SectionFooter        string
	tabContents          []TabContent   // message contents
	sectionFields        []SectionField // Field actions configured for the section
	indexActiveEditField int            // Índice del campo de configuración seleccionado
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
type tuiSectionAdapter interface {
	SectionTitle() string // eg: "BUILD", "TEST"
	SectionFieldsAdapters() []sectionFieldAdapter
	SectionFooter() string // eg: "Press 't' to compile", "Press 'r' to run tests"
}

type TuiConfig struct {
	TabIndexStart int                 // tabIndexStart is the index of the tab to start
	ExitChan      chan bool           //  global chan to close app
	Sections      []tuiSectionAdapter // represent sections in the tui
	Color         *ColorStyle
}

func NewTUI(c *TuiConfig) *TextUserInterface {

	tabsSection := make([]TabSection, len(c.Sections))

	for i, section := range c.Sections {

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
			title:         section.SectionTitle(),
			SectionFooter: section.SectionFooter(),
			tabContents:   []TabContent{},
			sectionFields: tabActions,
		}
	}

	// Create default tab if no tabs provided
	if len(tabsSection) == 0 {
		defaultTab := TabSection{
			title:         "GODEV",
			tabContents:   []TabContent{},
			sectionFields: []SectionField{},
		}
		tabsSection = append(tabsSection, defaultTab)

		testTab := TabSection{
			title:         "TEST",
			tabContents:   []TabContent{},
			sectionFields: []SectionField{},
		}
		tabsSection = append(tabsSection, testTab)

		c.TabIndexStart = 0 // Set the default tab index to 0
	}

	tui := &TextUserInterface{
		TuiConfig:       c,
		tabsSection:     tabsSection,
		activeTab:       c.TabIndexStart,
		tabContentsChan: make(chan TabContent, 100),
		currentTime:     time.Now().Format("15:04:05"),
		TuiStyle:        NewTuiStyle(c.Color),
	}

	tui.tea = tea.NewProgram(tui,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	return tui
}

func (cf *SectionField) SetCursorAtEnd() {
	cf.cursor = len(cf.value)
}

// Init inicializa el modelo
func (h *TextUserInterface) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		h.listenToMessages(),
		h.tickEverySecond(),
	)
}

// listenToMessages crea un comando para escuchar mensajes del canal
func (h *TextUserInterface) listenToMessages() tea.Cmd {
	return func() tea.Msg {
		msg := <-h.tabContentsChan
		return channelMsg(msg)
	}
}

// tickEverySecond crea un comando para actualizar el tiempo
func (h *TextUserInterface) tickEverySecond() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Add this helper function
func (h *TextUserInterface) addTerminalPrint(msgType MessageType, content string) {
	h.tabsSection[h.activeTab].tabContents = append(
		h.tabsSection[h.activeTab].tabContents,
		TabContent{
			Type:    msgType,
			Content: content,
			Time:    time.Now(),
		},
	)
}

// Update maneja las actualizaciones del estado
func (h *TextUserInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg: // Al presionar una tecla
		if h.editingFieldValueInSection { // EDITING FIELD IN SECTION

			currentTab := &h.tabsSection[h.activeTab]

			currentField := &h.tabsSection[h.activeTab].sectionFields[currentTab.indexActiveEditField]

			if currentField.Editable() { // Si el campo es editable, permitir la edición

				switch msg.String() {
				case "enter": // Al presionar ENTER, guardamos los cambios o ejecutamos la acción
					if _, err := currentField.FieldValueChange(currentField.value); err != nil {
						h.addTerminalPrint(ErrorMsg, fmt.Sprintf("Error updating field: %v %v", currentField.name, err))
					}
					// return the cursor to its position in the field
					currentField.SetCursorAtEnd()
					h.editingFieldValueInSection = false
					return h, nil
				case "esc": // Al presionar ESC, descartamos los cambios
					currentField := &h.tabsSection[h.activeTab].sectionFields[currentTab.indexActiveEditField]
					// currentField.value = GetConfigFields()[currentTab.indexActiveEditField].value // Restaurar valor original

					// volvemos el cursor a su posición
					currentField.SetCursorAtEnd()

					h.editingFieldValueInSection = false
					return h, nil
				case "left": // Mover el cursor a la izquierda
					currentField := &h.tabsSection[h.activeTab].sectionFields[currentTab.indexActiveEditField]
					if currentField.cursor > 0 {
						currentField.cursor--
					}
				case "right": // Mover el cursor a la derecha
					currentField := &h.tabsSection[h.activeTab].sectionFields[currentTab.indexActiveEditField]
					if currentField.cursor < len(currentField.value) {
						currentField.cursor++
					}
				default:
					currentField := &h.tabsSection[h.activeTab].sectionFields[currentTab.indexActiveEditField]
					if msg.String() == "backspace" && currentField.cursor > 0 {
						currentField.value = currentField.value[:currentField.cursor-1] + currentField.value[currentField.cursor:]
						currentField.cursor--
					} else if len(msg.String()) == 1 {
						currentField.value = currentField.value[:currentField.cursor] + msg.String() + currentField.value[currentField.cursor:]
						currentField.cursor++
					}
				}
			} else { // Si el campo no es editable, solo ejecutar la acción

				switch msg.String() {
				case "enter":

					msgType := OkMsg
					// content eg: "Browser Opened"
					content, err := currentField.FieldValueChange(currentField.value)
					if err != nil {
						msgType = ErrorMsg
						content = fmt.Sprintf("%s %s %s", currentField.label, content, err.Error())
					}
					currentField.value = content
					h.addTerminalPrint(msgType, content)
					h.editingFieldValueInSection = false
				}

			}

		} else {

			switch msg.String() {
			case "up": // Mover hacia arriba el indice del campo activo
				currentTab := &h.tabsSection[h.activeTab]

				if currentTab.indexActiveEditField > 0 {
					currentTab.indexActiveEditField--
				}
			case "down": // Mover hacia abajo el indice del campo activo
				currentTab := &h.tabsSection[h.activeTab]
				if currentTab.indexActiveEditField < len(h.tabsSection[0].sectionFields)-1 {
					currentTab.indexActiveEditField++
				}
			case "enter":
				h.editingFieldValueInSection = true
			case "tab":
				h.activeTab = (h.activeTab + 1) % len(h.tabsSection)
			case "shift+tab":
				h.activeTab = (h.activeTab - 1 + len(h.tabsSection)) % len(h.tabsSection)
			case "ctrl+l":
				h.tabsSection[h.activeTab].tabContents = []TabContent{}
			case "ctrl+c":
				close(h.ExitChan) // Cerrar el canal para señalizar a todas las goroutines
				return h, tea.Quit
			default:

			}
		}
	case channelMsg:
		h.tabsSection[h.activeTab].tabContents = append(h.tabsSection[h.activeTab].tabContents, TabContent(msg))
		cmds = append(cmds, h.listenToMessages())

	case tea.WindowSizeMsg:

		headerHeight := lipgloss.Height(h.headerView())
		footerHeight := lipgloss.Height(h.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !h.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			h.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			h.viewport.YPosition = headerHeight
			h.viewport.SetContent(h.ContentView())
			h.ready = true
		} else {
			h.viewport.Width = msg.Width
			h.viewport.Height = msg.Height - verticalMarginHeight
		}

	case tickMsg:
		h.currentTime = time.Now().Format("15:04:05")
		cmds = append(cmds, h.tickEverySecond())
	}

	return h, tea.Batch(cmds...)
}

func (h *TextUserInterface) StartTUI(wg *sync.WaitGroup) {
	defer wg.Done()

	if _, err := h.tea.Run(); err != nil {
		fmt.Println("Error running goCompiler:", err)
		fmt.Println("\nPress any key to exit...")
		var input string
		fmt.Scanln(&input)
	}
}

func (t *TextUserInterface) ReturnFocus() error {

	time.Sleep(100 * time.Millisecond)

	pid := os.Getpid()

	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("xdotool", "search", "--pid", fmt.Sprint(pid), "windowactivate")
		return cmd.Run()

	case "darwin":
		cmd := exec.Command("osascript", "-e", fmt.Sprintf(`
            tell application "System Events"
                set frontmost of the first process whose unix id is %d to true
            end tell
        `, pid))
		return cmd.Run()

	case "windows":
		// Usando taskkill para verificar si el proceso existe y obtener su ventana
		cmd := exec.Command("cmd", "/C", fmt.Sprintf("tasklist /FI \"PID eq %d\" /FO CSV /NH", pid))
		return cmd.Run()

	default:
		return errors.New("Focus Unsupported platform")
	}

}
