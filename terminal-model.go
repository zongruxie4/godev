package godev

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const BUILD_TAB_INDEX = 1

// Tab representa una pestaña individual incluye un slice de campos de configuración
type Tab struct {
	title    string
	content  []TerminalMessage
	selected bool
	footer   string
	actions  map[string]string
	configs  []ConfigField // Campos de configuración para GODEV
}

// Terminal mantiene el estado de la aplicación
type Terminal struct {
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
func (t *Terminal) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		t.listenToMessages(),
		t.tickEverySecond(),
	)
}

// listenToMessages crea un comando para escuchar mensajes del canal
func (t *Terminal) listenToMessages() tea.Cmd {
	return func() tea.Msg {
		msg := <-t.messagesChan
		return channelMsg(msg)
	}
}

// tickEverySecond crea un comando para actualizar el tiempo
func (t *Terminal) tickEverySecond() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update maneja las actualizaciones del estado
func (t *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.activeTab == 0 && t.editingConfig { // GODEV tab y editando
			switch msg.String() {
			case "enter":
				currentField := &t.tabs[0].configs[t.activeConfig]
				if err := config.UpdateFieldWithNotification(currentField, currentField.value); err != nil {
					t.tabs[BUILD_TAB_INDEX].content = append(
						t.tabs[BUILD_TAB_INDEX].content,
						TerminalMessage{
							Type:    ErrorMsg,
							Content: fmt.Sprintf("Error updating field:%v %v", currentField.name, err),
							Time:    time.Now(),
						},
					)
				}

				// volvemos el cursor a su posición
				currentField.SetCursorAtEnd()

				t.editingConfig = false
				return t, nil
			case "esc":
				// Al presionar ESC, descartamos los cambios
				currentField := &t.tabs[0].configs[t.activeConfig]
				currentField.value = config.GetConfigFields()[t.activeConfig].value // Restaurar valor original
				t.editingConfig = false

				// volvemos el cursor a su posición
				currentField.SetCursorAtEnd()

				return t, nil
			case "left":
				currentField := &t.tabs[0].configs[t.activeConfig]
				if currentField.cursor > 0 {
					currentField.cursor--
				}
			case "right":
				currentField := &t.tabs[0].configs[t.activeConfig]
				if currentField.cursor < len(currentField.value) {
					currentField.cursor++
				}
			default:
				currentField := &t.tabs[0].configs[t.activeConfig]
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
				if t.activeTab == 0 && t.activeConfig > 0 {
					t.activeConfig--
				}
			case "down":
				if t.activeTab == 0 && t.activeConfig < len(t.tabs[0].configs)-1 {
					t.activeConfig++
				}
			case "enter":
				if t.activeTab == 0 {
					t.editingConfig = true
				}
			case "tab":
				t.activeTab = (t.activeTab + 1) % len(t.tabs)
			case "shift+tab":
				t.activeTab = (t.activeTab - 1 + len(t.tabs)) % len(t.tabs)
			case "ctrl+l":
				t.tabs[t.activeTab].content = []TerminalMessage{}
			case "ctrl+c":
				return t, tea.Quit
			default:
				// Manejar acciones específicas de la pestaña
				if action, exists := t.tabs[t.activeTab].actions[msg.String()]; exists {
					t.tabs[t.activeTab].content = append(
						t.tabs[t.activeTab].content,
						TerminalMessage{
							Type:    "action",
							Content: action,
							Time:    time.Now(),
						},
					)
				}
			}
		}
	case channelMsg:
		t.tabs[t.activeTab].content = append(t.tabs[t.activeTab].content, TerminalMessage(msg))
		cmds = append(cmds, t.listenToMessages())

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

	case tickMsg:
		t.currentTime = time.Now().Format("15:04:05")
		cmds = append(cmds, t.tickEverySecond())
	}

	return t, tea.Batch(cmds...)
}

// NewTerminal crea una nueva instancia de Terminal
func NewTerminal() *Terminal {

	t := &Terminal{
		tabs: []Tab{
			{
				title:   "GODEV",
				content: []TerminalMessage{},
				configs: config.GetConfigFields(),
				footer:  "↑↓ to navigate | ENTER to edit | ESC to exit edit",
			},
			{
				title:   "BUILD",
				content: []TerminalMessage{},
				footer:  "'t' TinyGo | 'w' Web Browser",
				actions: map[string]string{
					"t": "TinyGo compiler activated!",
					"w": "Opening browser...",
				},
			},
			{
				title:   "TEST",
				content: []TerminalMessage{},
				actions: map[string]string{
					"r": "Running tests...",
				},
			},
			{
				title:   "DEPLOY",
				content: []TerminalMessage{},
				footer:  "'d' Docker | 'v' VPS Setup",
				actions: map[string]string{
					"d": "Generating Dockerfile...",
					"v": "Configuring VPS...",
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
	return t
}

func (t *Terminal) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	t.tea = tea.NewProgram(t, tea.WithAltScreen())
	if _, err := t.tea.Run(); err != nil {
		fmt.Println("Error running program:", err)
		fmt.Println("\nPress any key to exit...")
		var input string
		fmt.Scanln(&input)
	}
}
