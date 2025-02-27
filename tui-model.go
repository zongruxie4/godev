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
type channelMsg tabContent

// Print representa un mensaje de actualización
type tickMsg time.Time

// tabContent imprime un mensaje en la tui
type tabContent struct {
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

	activeTab                  int  // current tab index
	editingFieldValueInSection bool // Si estamos editando un campo
	SectionFooter              string
	currentTime                string
	tabContentsChan            chan tabContent
	tea                        *tea.Program
}

// represent the tab section in the tui
type TabSection struct {
	Title         string         // eg: "BUILD", "TEST"
	SectionFields []SectionField // Field actions configured for the section
	SectionFooter string         // eg: "Press 't' to compile", "Press 'r' to run tests"
	// internal use
	tabContents          []tabContent // message contents
	indexActiveEditField int          // Índice del campo de configuración seleccionado
}

// Interface for handling tab field sectionFields
type SectionField struct {
	Name             string                                               // eg: "port", "Server Port", "8080"
	Label            string                                               // eg: "Server Port"
	Value            string                                               // eg: "8080"
	Editable         bool                                                 // if no editable eject the action FieldValueChange directly
	FieldValueChange func(newValue string) (runMessage string, err error) //eg: "8080" -> "9090" runMessage: "Port changed from 8080 to 9090"
	//internal use
	index  int
	cursor int // cursor position in text value
}

type TuiConfig struct {
	TabIndexStart int          // is the index of the tab to start
	ExitChan      chan bool    //  global chan to close app
	TabSections   []TabSection // represent sections in the tui
	Color         *ColorStyle
}

func NewTUI(c *TuiConfig) *TextUserInterface {

	// Create default tab if no tabs provided
	if len(c.TabSections) == 0 {
		defaultTab := TabSection{
			Title:         "BUILD",
			SectionFields: []SectionField{},
			SectionFooter: "build footer example",
			tabContents:   []tabContent{},
		}
		c.TabSections = append(c.TabSections, defaultTab)

		testTab := TabSection{
			Title:         "DEPLOY",
			SectionFields: []SectionField{},
			SectionFooter: "deploy footer example",
			tabContents:   []tabContent{},
		}
		c.TabSections = append(c.TabSections, testTab)

		c.TabIndexStart = 0 // Set the default tab index to 0
	}

	// Recorremos c.TabSections y actualizamos el índice de cada campo.
	for i := range c.TabSections {
		section := &c.TabSections[i]
		for j := range section.SectionFields {
			section.SectionFields[j].index = j
			section.SectionFields[j].cursor = 0
		}
		// Si es necesario asignar otros valores, se hace aquí.
	}

	tui := &TextUserInterface{
		TuiConfig:       c,
		activeTab:       c.TabIndexStart,
		tabContentsChan: make(chan tabContent, 100),
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
	cf.cursor = len(cf.Value)
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
	h.TabSections[h.activeTab].tabContents = append(
		h.TabSections[h.activeTab].tabContents,
		tabContent{
			Type:    msgType,
			Content: content,
			Time:    time.Now(),
		},
	)
}

// Update maneja las actualizaciones del estado
func (h *TextUserInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg: // Al presionar una tecla
		if h.editingFieldValueInSection { // EDITING FIELD IN SECTION

			currentTab := &h.TabSections[h.activeTab]

			currentField := &h.TabSections[h.activeTab].SectionFields[currentTab.indexActiveEditField]

			if currentField.Editable { // Si el campo es editable, permitir la edición

				switch msg.String() {
				case "enter": // Al presionar ENTER, guardamos los cambios o ejecutamos la acción
					if _, err := currentField.FieldValueChange(currentField.Value); err != nil {
						h.addTerminalPrint(ErrorMsg, fmt.Sprintf("Error updating field: %v %v", currentField.Name, err))
					}
					// return the cursor to its position in the field
					currentField.SetCursorAtEnd()
					h.editingFieldValueInSection = false
					return h, nil
				case "esc": // Al presionar ESC, descartamos los cambios
					currentField := &h.TabSections[h.activeTab].SectionFields[currentTab.indexActiveEditField]
					// currentField.Value = GetConfigFields()[currentTab.indexActiveEditField].value // Restaurar valor original

					// volvemos el cursor a su posición
					currentField.SetCursorAtEnd()

					h.editingFieldValueInSection = false
					return h, nil
				case "left": // Mover el cursor a la izquierda
					currentField := &h.TabSections[h.activeTab].SectionFields[currentTab.indexActiveEditField]
					if currentField.cursor > 0 {
						currentField.cursor--
					}
				case "right": // Mover el cursor a la derecha
					currentField := &h.TabSections[h.activeTab].SectionFields[currentTab.indexActiveEditField]
					if currentField.cursor < len(currentField.Value) {
						currentField.cursor++
					}
				default:
					currentField := &h.TabSections[h.activeTab].SectionFields[currentTab.indexActiveEditField]
					if msg.String() == "backspace" && currentField.cursor > 0 {
						currentField.Value = currentField.Value[:currentField.cursor-1] + currentField.Value[currentField.cursor:]
						currentField.cursor--
					} else if len(msg.String()) == 1 {
						currentField.Value = currentField.Value[:currentField.cursor] + msg.String() + currentField.Value[currentField.cursor:]
						currentField.cursor++
					}
				}
			} else { // Si el campo no es editable, solo ejecutar la acción

				switch msg.String() {
				case "enter":

					msgType := OkMsg
					// content eg: "Browser Opened"
					content, err := currentField.FieldValueChange(currentField.Value)
					if err != nil {
						msgType = ErrorMsg
						content = fmt.Sprintf("%s %s %s", currentField.Label, content, err.Error())
					}
					currentField.Value = content
					h.addTerminalPrint(msgType, content)
					h.editingFieldValueInSection = false
				}

			}

		} else {

			switch msg.String() {
			case "up": // Mover hacia arriba el indice del campo activo
				currentTab := &h.TabSections[h.activeTab]

				if currentTab.indexActiveEditField > 0 {
					currentTab.indexActiveEditField--
				}
			case "down": // Mover hacia abajo el indice del campo activo
				currentTab := &h.TabSections[h.activeTab]
				if currentTab.indexActiveEditField < len(h.TabSections[0].SectionFields)-1 {
					currentTab.indexActiveEditField++
				}
			case "enter":
				h.editingFieldValueInSection = true
			case "tab": // change tabSection
				h.activeTab = (h.activeTab + 1) % len(h.TabSections)
				h.updateViewport()
			case "shift+tab": // change tabSection
				h.activeTab = (h.activeTab - 1 + len(h.TabSections)) % len(h.TabSections)
				h.updateViewport()
			case "ctrl+l":
				h.TabSections[h.activeTab].tabContents = []tabContent{}
			case "ctrl+c":
				close(h.ExitChan) // Cerrar el canal para señalizar a todas las goroutines
				return h, tea.Quit
			default:

			}
		}
	case channelMsg:
		h.TabSections[h.activeTab].tabContents = append(h.TabSections[h.activeTab].tabContents, tabContent(msg))
		cmds = append(cmds, h.listenToMessages())

		h.updateViewport()

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
	// Handle keyboard and mouse events in the viewport
	h.viewport, cmd = h.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return h, tea.Batch(cmds...)
}

func (h *TextUserInterface) updateViewport() {
	h.viewport.SetContent(h.ContentView())
	h.viewport.GotoBottom()
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
