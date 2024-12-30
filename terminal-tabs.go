package godev

import (
	"fmt"
	"strings"
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

// Estilos para las pestañas
var (
	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(lipgloss.Color(background)).
		Padding(0, 1)

	activeTab = lipgloss.NewStyle().
			Border(activeTabBorder, true).
			Bold(true).
			Background(lipgloss.Color(background)).
			Foreground(lipgloss.Color(foreGround)).
			Padding(0, 1)

	tabGap = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, 1)
)

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
	contentHeight := t.height - headerHeight - footerHeight
	contentWidth := t.width - 2

	// Header con pestañas
	header := headerFooterStyle.
		Width(contentWidth).
		Render(fmt.Sprintf("🚀 GoDEV - %s", t.currentTime))

	// Pestañas
	tabs := t.renderTabs()

	// Contenido de la pestaña activa
	visibleMessages := contentHeight - 1
	start := 0
	activeContent := t.tabs[t.activeTab].content
	if len(activeContent) > visibleMessages {
		start = len(activeContent) - visibleMessages
	}

	var contentLines []string
	for i := start; i < len(activeContent); i++ {
		formattedMsg := t.formatMessage(activeContent[i])
		contentLines = append(contentLines, messageStyle.Render(formattedMsg))
	}

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
		tabs,
		contentArea,
		footer,
	)
}

// NewTerminal crea una nueva instancia de Terminal
func NewTerminal() *Terminal {
	t := &Terminal{
		tabs: []Tab{
			{title: "BUILD", content: []TerminalMessage{}},
			{title: "DEPLOY", content: []TerminalMessage{}},
			{title: "HELP", content: []TerminalMessage{}},
		},
		activeTab:    0,
		messagesChan: make(chan TerminalMessage, 100),
		footer:       "Press 'ESC' to exit | '1-3' or TAB to switch tabs | 'ctrl+l' to clear",
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
