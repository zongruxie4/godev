package godev

import (
	"fmt"
	"os"
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

// // Init inicializa el terminal
// func (t Terminal) Init() tea.Cmd {
// 	return tea.Batch(
// 		tea.Tick(time.Second, func(t time.Time) tea.Msg {
// 			return tickMsg(t)
// 		}),
// 		tea.EnterAltScreen, // Entrar en modo de pantalla alternativa
// 	)
// }

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
// func (t *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	switch msg := msg.(type) {
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "esc":
// 			// Mostrar todos los mensajes antes de salir
// 			t.messages = append(t.messages, fmt.Sprintf("%s: Exiting... Showing all messages",
// 				time.Now().Format("15:04:05")))
// 			time.Sleep(1 * time.Second) // Dar tiempo para mostrar los mensajes
// 			return t, tea.Quit
// 		case "t":
// 			// Acción especial al presionar 't'
// 			t.messages = append(t.messages, fmt.Sprintf("%s: You have activated a special action!",
// 				time.Now().Format("15:04:05")))
// 		case "b":
// 			// Acción para abrir el navegador
// 			t.messages = append(t.messages, fmt.Sprintf("%s: Opening browser...",
// 				time.Now().Format("15:04:05")))

// 		default:
// 			// Registra cualquier otra tecla presionada
// 			t.messages = append(t.messages, fmt.Sprintf("%s: Key pressed: %s",
// 				time.Now().Format("15:04:05"), msg.String()))
// 		}
// 	case tickMsg:
// 		// Actualiza el tiempo cada segundo
// 		now := time.Now()
// 		t.currentTime = now.Format("15:04:05")
// 		// Actualiza el footer
// 		t.footer = fmt.Sprintf("Press 'ESC' to exit | 't' Tinygo Compiler Activated: %s | 'b' Browser | ",
// 			t.currentTime)
// 	case tea.WindowSizeMsg:
// 		t.width = msg.Width
// 		t.height = msg.Height
// 	}

// 	return t, tea.Tick(time.Second, func(t time.Time) tea.Msg {
// 		return tickMsg(t)
// 	})
// }

// Update maneja las actualizaciones del estado
func (t *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
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

// Define estilos base
var (
	// Estilo para el borde principal
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")). // Morado
			Padding(0, 1)

	// Estilo para el header y footer
	headerFooterStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")). // Morado
				Foreground(lipgloss.Color("15")). // Blanco
				Bold(true).
				Padding(0, 2)

	// Estilo para los mensajes
	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")). // Blanco
			PaddingLeft(2)
)

// func (t *Terminal) forceUpdate() {
// 	// Enviar mensaje de actualización
// 	t.tea.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
// 	time.Sleep(200 * time.Millisecond) // Aumentar el tiempo para asegurar la actualización
// }

// View renderiza la interfaz
// func (t Terminal) View() string {
// 	// Dimensiones mínimas requeridas
// 	minWidth := 40
// 	minHeight := 10

// 	if t.width < minWidth || t.height < minHeight {
// 		return fmt.Sprintf("Terminal too small. Minimum size: %dx%d", minWidth, minHeight)
// 	}

// 	// Dimensiones fijas para header y footer
// 	headerHeight := 3
// 	footerHeight := 3
// 	contentHeight := t.height - headerHeight - footerHeight

// 	// Ancho efectivo del contenido
// 	contentWidth := t.width - 4 // 4 para los bordes

// 	// Header siempre visible
// 	header := headerFooterStyle.
// 		Width(contentWidth).
// 		Render(fmt.Sprintf("🚀 GoDEV - %s", t.currentTime))

// 	// Footer siempre visible
// 	footer := headerFooterStyle.
// 		Width(contentWidth).
// 		Render(t.footer)

// 	// Manejo de mensajes con scroll
// 	visibleMessages := contentHeight - 2
// 	start := 0
// 	if len(t.messages) > visibleMessages {
// 		start = len(t.messages) - visibleMessages
// 	}

// 	// Procesar mensajes manteniendo el estilo
// 	var contentLines []string
// 	for i := start; i < len(t.messages); i++ {
// 		msg := t.messages[i]
// 		if msg != "" {
// 			// Dividir mensajes largos en múltiples líneas
// 			maxLineWidth := contentWidth - 6 // Espacio para bullet y padding
// 			for len(msg) > maxLineWidth {
// 				line := msg[:maxLineWidth]
// 				contentLines = append(contentLines, messageStyle.Render("• "+line))
// 				msg = msg[maxLineWidth:]
// 			}
// 			if len(msg) > 0 {
// 				contentLines = append(contentLines, messageStyle.Render("• "+msg))
// 			}
// 		}
// 	}

// 	// Asegurar que hay suficientes líneas para llenar el espacio
// 	for len(contentLines) < visibleMessages {
// 		contentLines = append(contentLines, "")
// 	}

// 	// Si hay más líneas que espacio visible, truncar
// 	if len(contentLines) > visibleMessages {
// 		contentLines = contentLines[len(contentLines)-visibleMessages:]
// 	}

// 	content := strings.Join(contentLines, "\n")

// 	// Área de contenido con scroll
// 	contentArea := borderStyle.
// 		Width(contentWidth).
// 		Height(contentHeight).
// 		Render(content)

// 	// Unir las secciones manteniendo la alineación
// 	return lipgloss.JoinVertical(
// 		lipgloss.Left,
// 		header,
// 		contentArea,
// 		footer,
// 	)
// }

// View renderiza la interfaz
func (t *Terminal) View() string {
	if t.width < 40 || t.height < 10 {
		return "Terminal too small. Minimum size: 40x10"
	}

	headerHeight := 3
	footerHeight := 3
	contentHeight := t.height - headerHeight - footerHeight
	contentWidth := t.width - 4

	// Header
	header := headerFooterStyle.
		Width(contentWidth).
		Render(fmt.Sprintf("🚀 GoDEV - %s", t.currentTime))

	// Content
	// t.mu.Lock()
	visibleMessages := contentHeight - 2
	start := 0
	if len(t.messages) > visibleMessages {
		start = len(t.messages) - visibleMessages
	}

	var contentLines []string
	for i := start; i < len(t.messages); i++ {
		formattedMsg := t.formatMessage(t.messages[i])
		contentLines = append(contentLines, messageStyle.Render("• "+formattedMsg))
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
		lipgloss.Left,
		header,
		contentArea,
		footer,
	)
}

// inicia una nueva terminal
// func (h *handler) NewTerminal() {
// 	h.terminal = &Terminal{
// 		messages:    make([]string, 0),
// 		footer:      "Starting...",
// 		currentTime: time.Now().Format("15:04:05"),
// 		tickCount:   0,
// 	}

// 	options := []tea.ProgramOption{tea.WithAltScreen()}
// 	h.terminal.tea = tea.NewProgram(h.terminal, options...)
// }

// NewTerminal crea una nueva instancia de Terminal
func NewTerminal() *Terminal {
	t := &Terminal{
		messages:     make([]TerminalMessage, 0),
		messagesChan: make(chan TerminalMessage, 100),
		footer:       "Press 'ESC' to exit | 't' for TinyGo | 'b' for Browser",
		currentTime:  time.Now().Format("15:04:05"),
	}
	return t
}

// inicia la aplicación de terminal
// func (h *handler) RunTerminal(wg *sync.WaitGroup) {

// 	if _, err := h.terminal.tea.Run(); err != nil {
// 		fmt.Printf("Error running the application: %v\n", err)
// 		os.Exit(1)
// 	}
// }

func (h *handler) RunTerminal(wg *sync.WaitGroup) {
	defer wg.Done()
	if err := h.terminal.Start(); err != nil {
		fmt.Printf("Error running terminal: %v\n", err)
		os.Exit(1)
	}
}

// StartProgram inicia el programa de terminal
func (t *Terminal) Start() error {
	p := tea.NewProgram(t, tea.WithAltScreen())
	t.tea = p
	_, err := p.Run()
	return err
}
