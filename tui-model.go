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

const BUILD_TAB_INDEX = 1

// Tab representa una pestaña individual incluye un slice de campos de configuración
type TabAction struct {
	message      string
	active       bool
	shortCuts    string
	openHandler  func() error // handler para abrir/iniciar
	closeHandler func() error // handler para cerrar/detener
}

type Tab struct {
	title    string
	content  []TerminalMessage
	selected bool
	footer   string
	actions  []TabAction   // Now it's a slice instead of map
	configs  []ConfigField // Campos de configuración para GODEV
}

// TextUserInterface mantiene el estado de la aplicación
type TextUserInterface struct {
	tabs          []Tab
	activeTab     int
	activeConfig  int  // Índice del campo de configuración seleccionado
	editingConfig bool // Si estamos editando un campo
	messages      []TerminalMessage
	footer        string
	currentTime   string
	width         int
	height        int
	messagesChan  chan TerminalMessage
	tea           *tea.Program
}

// channelMsg es un tipo especial para mensajes del canal
type channelMsg TerminalMessage

// Msg representa un mensaje de actualización
type tickMsg time.Time

// Init inicializa el modelo
func (h *handler) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		h.tui.listenToMessages(),
		h.tui.tickEverySecond(),
	)
}

// listenToMessages crea un comando para escuchar mensajes del canal
func (t *TextUserInterface) listenToMessages() tea.Cmd {
	return func() tea.Msg {
		msg := <-t.messagesChan
		return channelMsg(msg)
	}
}

// tickEverySecond crea un comando para actualizar el tiempo
func (t *TextUserInterface) tickEverySecond() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update maneja las actualizaciones del estado
func (h *handler) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if h.tui.activeTab == 0 && h.tui.editingConfig { // GODEV tab y editando
			switch msg.String() {
			case "enter":
				currentField := &h.tui.tabs[0].configs[h.tui.activeConfig]
				if err := h.UpdateFieldWithNotification(currentField, currentField.value); err != nil {
					h.tui.tabs[BUILD_TAB_INDEX].content = append(
						h.tui.tabs[BUILD_TAB_INDEX].content,
						TerminalMessage{
							Type:    ErrorMsg,
							Content: fmt.Sprintf("Error updating field: %v %v", currentField.name, err),
							Time:    time.Now(),
						},
					)
				}

				// volvemos el cursor a su posición
				currentField.SetCursorAtEnd()

				h.tui.editingConfig = false
				return h, nil
			case "esc":
				// Al presionar ESC, descartamos los cambios
				currentField := &h.tui.tabs[0].configs[h.tui.activeConfig]
				currentField.value = h.GetConfigFields()[h.tui.activeConfig].value // Restaurar valor original
				h.tui.editingConfig = false

				// volvemos el cursor a su posición
				currentField.SetCursorAtEnd()

				return h, nil
			case "left":
				currentField := &h.tui.tabs[0].configs[h.tui.activeConfig]
				if currentField.cursor > 0 {
					currentField.cursor--
				}
			case "right":
				currentField := &h.tui.tabs[0].configs[h.tui.activeConfig]
				if currentField.cursor < len(currentField.value) {
					currentField.cursor++
				}
			default:
				currentField := &h.tui.tabs[0].configs[h.tui.activeConfig]
				if msg.String() == "backspace" && currentField.cursor > 0 {
					currentField.value = currentField.value[:currentField.cursor-1] + currentField.value[currentField.cursor:]
					currentField.cursor--
				} else if len(msg.String()) == 1 {
					currentField.value = currentField.value[:currentField.cursor] + msg.String() + currentField.value[currentField.cursor:]
					currentField.cursor++
				}
			}
		} else {
			switch msg.String() {
			case "up":
				if h.tui.activeTab == 0 && h.tui.activeConfig > 0 {
					h.tui.activeConfig--
				}
			case "down":
				if h.tui.activeTab == 0 && h.tui.activeConfig < len(h.tui.tabs[0].configs)-1 {
					h.tui.activeConfig++
				}
			case "enter":
				if h.tui.activeTab == 0 {
					h.tui.editingConfig = true
				}
			case "tab":
				h.tui.activeTab = (h.tui.activeTab + 1) % len(h.tui.tabs)
			case "shift+tab":
				h.tui.activeTab = (h.tui.activeTab - 1 + len(h.tui.tabs)) % len(h.tui.tabs)
			case "ctrl+l":
				h.tui.tabs[h.tui.activeTab].content = []TerminalMessage{}
			case "ctrl+c":
				close(h.exitChan) // Cerrar el canal para señalizar a todas las goroutines
				return h, tea.Quit
			default:
				// Manejar acciones específicas de la pestaña

				action, exist := h.tui.getAction(h.tui.activeTab, msg.String())

				if exist {
					// Toggle the active state of the action
					action.active = !action.active

					status := "opened"
					if !action.active {
						status = "closed"
					}

					// console action message
					h.tui.tabs[h.tui.activeTab].content = append(
						h.tui.tabs[h.tui.activeTab].content,
						TerminalMessage{
							Type:    OkMsg,
							Content: fmt.Sprintf("%s %s", action.message, status),
							Time:    time.Now(),
						},
					)

					var err error
					if !action.active {
						if action.closeHandler != nil {
							err = action.closeHandler()
						}
					} else {
						if action.openHandler != nil {
							err = action.openHandler()
						}
					}

					if err != nil {
						// execution result message
						h.tui.tabs[h.tui.activeTab].content = append(
							h.tui.tabs[h.tui.activeTab].content,
							TerminalMessage{
								Type:    ErrorMsg,
								Content: err.Error(),
								Time:    time.Now(),
							},
						)
					}

					// Update the action in the tab's actions list by finding the matching message
					// and replacing it with the updated action
					for i, a := range h.tui.tabs[h.tui.activeTab].actions {
						if a.message == action.message {
							h.tui.tabs[h.tui.activeTab].actions[i] = action
							break
						}
					}

				}
			}
		}
	case channelMsg:
		h.tui.tabs[h.tui.activeTab].content = append(h.tui.tabs[h.tui.activeTab].content, TerminalMessage(msg))
		cmds = append(cmds, h.tui.listenToMessages())

	case tea.WindowSizeMsg:
		h.tui.width = msg.Width
		h.tui.height = msg.Height

	case tickMsg:
		h.tui.currentTime = time.Now().Format("15:04:05")
		cmds = append(cmds, h.tui.tickEverySecond())
	}

	return h, tea.Batch(cmds...)
}
func (t *TextUserInterface) getAction(activeTab int, shortcut string) (TabAction, bool) {

	if activeTab >= 0 && activeTab < len(t.tabs) {
		for _, action := range t.tabs[activeTab].actions {
			if action.shortCuts == shortcut {
				return action, true
			}
		}
	}
	return TabAction{}, false
}

// NewTerminal crea una nueva instancia de TextUserInterface
func (h *handler) NewTextUserInterface() {

	h.tui = &TextUserInterface{
		tabs: []Tab{
			{
				title:   "GODEV",
				content: []TerminalMessage{},
				configs: h.GetConfigFields(),
				footer:  "↑↓ to navigate | ENTER to edit | ESC to exit edit",
			},
			{
				title:   "BUILD",
				content: []TerminalMessage{},
				actions: []TabAction{
					{
						message:   "TinyGo compiler",
						active:    false,
						shortCuts: "t",
						openHandler: func() error {
							// TinyGo compilation logic
							return nil
						},
					},
					{
						message:      "Web Browser",
						active:       false,
						shortCuts:    "w",
						openHandler:  h.OpenBrowser,
						closeHandler: h.CloseBrowser,
					},
				},
			},
			{
				title:   "TEST",
				content: []TerminalMessage{},
				actions: []TabAction{
					{
						message:   "Running tests...",
						active:    false,
						shortCuts: "r",
						openHandler: func() error {
							// Implement test running logic
							return nil
						},
					},
				},
			},
			{
				title:   "DEPLOY",
				content: []TerminalMessage{},
				footer:  "'d' Docker | 'v' VPS Setup",
				actions: []TabAction{
					{
						message:   "Generating Dockerfile...",
						active:    false,
						shortCuts: "d",
						openHandler: func() error {
							// Implement Docker generation logic
							return nil
						},
					},
					{
						message:   "Configuring VPS...",
						active:    false,
						shortCuts: "v",
						openHandler: func() error {
							// Implement VPS configuration logic
							return nil
						},
					},
				},
			},
			{
				title:   "HELP",
				content: []TerminalMessage{},
				footer:  "Press 'h' for commands list | 'ctrl+c' to Exit",
			},
		},
		activeTab:    BUILD_TAB_INDEX, // Iniciamos en BUILD
		messagesChan: make(chan TerminalMessage, 100),
		currentTime:  time.Now().Format("15:04:05"),
	}
	return
}

func (h *handler) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	h.tui.tea = tea.NewProgram(h, tea.WithAltScreen())
	if _, err := h.tui.tea.Run(); err != nil {
		fmt.Println("Error running program:", err)
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
