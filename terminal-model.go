package godev

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tab representa una pestaña individual
type Tab struct {
	title    string
	content  []TerminalMessage
	selected bool
}

// Terminal mantiene el estado de la aplicación
type Terminal struct {
	*terminalModel
	*terminalView
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
		terminalModel: newTerminalModel(),
		terminalView:  newTerminalView(),
	}
	return t
}

func (t *Terminal) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	t.tea = t.terminalModel.startTeaProgram(t.terminalView)
	if _, err := t.tea.Run(); err != nil {
		t.MsgError("Error running program:", err)
	}
}
