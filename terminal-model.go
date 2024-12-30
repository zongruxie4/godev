package godev

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab representa una pestaña individual
type Tab struct {
	title    string
	content  []TerminalMessage
	selected bool
	footer   string
	actions  map[string]string // tecla -> descripción
}

// Terminal mantiene el estado de la aplicación
type Terminal struct {
	tabs         []Tab
	activeTab    int
	messages     []TerminalMessage
	footer       string
	currentTime  string
	width        int
	height       int
	messagesChan chan TerminalMessage
	tea          *tea.Program
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
		switch msg.String() {
		case "1", "2", "3":
			index := int(msg.String()[0] - '1')
			if index >= 0 && index < len(t.tabs) {
				t.activeTab = index
			}
		case "tab":
			t.activeTab = (t.activeTab + 1) % len(t.tabs)
		case "shift+tab":
			t.activeTab = (t.activeTab - 1 + len(t.tabs)) % len(t.tabs)
		case "ctrl+l":
			t.tabs[t.activeTab].content = []TerminalMessage{}
		case "esc", "ctrl+c":
			return t, tea.Quit
		default:
			// Manejar acciones específicas de la pestaña
			if action, exists := t.tabs[t.activeTab].actions[msg.String()]; exists {
				t.tabs[t.activeTab].content = append(
					t.tabs[t.activeTab].content,
					TerminalMessage{
						Type:    "action",
						Content: action,
						Time:    time.Now(), // Agregamos la marca de tiempo actual
					},
				)
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

// renderTabs renderiza la barra de pestañas
func (t *Terminal) renderTabs() string {
	var renderedTabs []string

	for i, currentTab := range t.tabs {
		var style lipgloss.Style
		if i == t.activeTab {
			style = activeTab
		} else {
			style = tab
		}
		renderedTabs = append(renderedTabs, style.Render(currentTab.title))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// NewTerminal crea una nueva instancia de Terminal
func NewTerminal() *Terminal {
	t := &Terminal{
		tabs: []Tab{
			{
				title:   "BUILD",
				content: []TerminalMessage{},
				footer:  "ESC to exit | 't' TinyGo | 'w' Web Browser | 'ctrl+l' clear",
				actions: map[string]string{
					"t": "TinyGo compiler activated!",
					"w": "Opening browser...",
				},
			},
			{
				title:   "DEPLOY",
				content: []TerminalMessage{},
				footer:  "ESC to exit | 'd' Docker | 'v' VPS Setup | 'ctrl+l' clear",
				actions: map[string]string{
					"d": "Generating Dockerfile...",
					"v": "Configuring VPS...",
				},
			},
			{
				title:   "HELP",
				content: []TerminalMessage{},
				footer:  "ESC to exit | 'h' Show Commands | 'ctrl+l' clear",
				actions: map[string]string{
					"h": "Showing available commands...",
				},
			},
		},
		activeTab:    0,
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
	}
}
