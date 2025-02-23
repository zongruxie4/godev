package godev

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

// channelMsg es un tipo especial para mensajes del canal
type channelMsg TerminalPrint

// Print representa un mensaje de actualización
type tickMsg time.Time

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
		msg := <-h.messagesChan
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
	h.tabsSection[h.activeTabIndex].terminalPrints = append(
		h.tabsSection[h.activeTabIndex].terminalPrints,
		TerminalPrint{
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

			currentField := &h.tabsSection[h.activeTabIndex].sectionFields[h.indexActiveEditField]

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
					currentField := &h.tabsSection[h.activeTabIndex].sectionFields[h.indexActiveEditField]
					currentField.value = GetConfigFields()[h.indexActiveEditField].value // Restaurar valor original

					// volvemos el cursor a su posición
					currentField.SetCursorAtEnd()

					h.editingFieldValueInSection = false
					return h, nil
				case "left": // Mover el cursor a la izquierda
					currentField := &h.tabsSection[h.activeTabIndex].sectionFields[h.indexActiveEditField]
					if currentField.cursor > 0 {
						currentField.cursor--
					}
				case "right": // Mover el cursor a la derecha
					currentField := &h.tabsSection[h.activeTabIndex].sectionFields[h.indexActiveEditField]
					if currentField.cursor < len(currentField.value) {
						currentField.cursor++
					}
				default:
					currentField := &h.tabsSection[h.activeTabIndex].sectionFields[h.indexActiveEditField]
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
				if h.indexActiveEditField > 0 {
					h.indexActiveEditField--
				}
			case "down": // Mover hacia abajo el indice del campo activo
				if h.indexActiveEditField < len(h.tabsSection[0].sectionFields)-1 {
					h.indexActiveEditField++
				}
			case "enter":
				h.editingFieldValueInSection = true
			case "tab":
				h.activeTabIndex = (h.activeTabIndex + 1) % len(h.tabsSection)
			case "shift+tab":
				h.activeTabIndex = (h.activeTabIndex - 1 + len(h.tabsSection)) % len(h.tabsSection)
			case "ctrl+l":
				h.tabsSection[h.activeTabIndex].terminalPrints = []TerminalPrint{}
			case "ctrl+c":
				close(h.exitChan) // Cerrar el canal para señalizar a todas las goroutines
				return h, tea.Quit
			default:

			}
		}
	case channelMsg:
		h.tabsSection[h.activeTabIndex].terminalPrints = append(h.tabsSection[h.activeTabIndex].terminalPrints, TerminalPrint(msg))
		cmds = append(cmds, h.listenToMessages())

	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height

	case tickMsg:
		h.currentTime = time.Now().Format("15:04:05")
		cmds = append(cmds, h.tickEverySecond())
	}

	return h, tea.Batch(cmds...)
}

func (t *TextUserInterface) getFielActionByShortCut(activeTabIndex int, shortcut string) (SectionField, bool) {

	// if activeTabIndex >= 0 && activeTabIndex < len(t.tabsSection) {
	// 	for _, action := range t.tabsSection[activeTabIndex].sectionFields {
	// 		if action.ShortCut() == shortcut {
	// 			return action, true
	// 		}
	// 	}
	// }
	return SectionField{}, false
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
