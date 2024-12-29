package godev

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Terminal mantiene el estado de la aplicación
type Terminal struct {
	messages     []TerminalMessage
	footer       string
	currentTime  string
	width        int
	height       int
	messagesChan chan TerminalMessage
	tea          *tea.Program
	// mu           sync.Mutex
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

		case "t", "T": // Detecta tanto minúscula como mayúscula
			t.MsgWarning("TinyGo compiler activated!")
		case "b", "B":
			t.MsgWarning("Opening browser...")

		case "ctrl+l": // Limpieza con Ctrl+L (común en terminales Unix)
			t.messages = []TerminalMessage{}
		case "esc", "ctrl+c":
			return t, tea.Quit

		}

	case channelMsg:
		// t.mu.Lock()
		t.messages = append(t.messages, TerminalMessage(msg))
		// t.mu.Unlock()
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

const background = "#FF6600" // orange
const foreGround = "#F4F4F4" //white

// Define estilos base
var (
	// Estilo para el borde principal
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(background)).
			Padding(0, 1)

	// Estilo para el header y footer
	headerFooterStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(background)).
				Foreground(lipgloss.Color(foreGround)).
				Bold(true).
				Padding(0, 2)

	// Estilo para los mensajes
	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(foreGround)).
			PaddingLeft(2)
)

// View renderiza la interfaz
func (t *Terminal) View() string {
	if t.width < 40 || t.height < 10 {
		return "Terminal too small. Minimum size: 40x10"
	}

	headerHeight := 3
	footerHeight := 3
	contentHeight := t.height - headerHeight - footerHeight + 2
	contentWidth := t.width - 2
	// contentWidth := t.width - 4

	// Header
	header := headerFooterStyle.
		Width(contentWidth).
		Render(fmt.Sprintf("🚀 GoDEV - %s", t.currentTime))

	// Content
	// t.mu.Lock()
	visibleMessages := contentHeight
	// visibleMessages := contentHeight - 2
	start := 0
	if len(t.messages) > visibleMessages {
		start = len(t.messages) - visibleMessages
	}

	var contentLines []string
	for i := start; i < len(t.messages); i++ {
		formattedMsg := t.formatMessage(t.messages[i])
		contentLines = append(contentLines, messageStyle.Render(formattedMsg))
	}
	// t.mu.Unlock()

	content := strings.Join(contentLines, "\n")
	contentArea := borderStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	// Footer
	footer := headerFooterStyle.
		Width(contentWidth).
		Render(t.footer)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		contentArea,
		footer,
	)
}

// NewTerminal crea una nueva instancia de Terminal
func NewTerminal() *Terminal {
	t := &Terminal{
		messages:     make([]TerminalMessage, 0),
		messagesChan: make(chan TerminalMessage, 100),
		footer:       "Press 'ESC' to exit | 't' for TinyGo | 'b' for Browser",
		currentTime:  time.Now().Format("15:04:05"),
	}

	log.SetOutput(t)

	return t
}

func (t *Terminal) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	t.tea = tea.NewProgram(t, tea.WithAltScreen())
	_, err := t.tea.Run()
	if err != nil {
		log.Println(err)
	}
}
