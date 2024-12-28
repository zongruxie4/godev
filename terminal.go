package godev

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Terminal mantiene el estado de la aplicación
type Terminal struct {
	messages    []string
	footer      string
	currentTime string
	tickCount   int
	width       int
	height      int
	tea         *tea.Program
}

// Estilos para los mensajes de colores
var (
	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("32")) // Verde
	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("31")) // Rojo
	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")) // Amarillo
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("36")) // Cian
)

// Msg representa un mensaje de actualización
type tickMsg time.Time

// Init inicializa el terminal
func (t Terminal) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
		tea.EnterAltScreen, // Entrar en modo de pantalla alternativa
	)
}

// Update maneja las actualizaciones del estado
func (t *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Mostrar todos los mensajes antes de salir
			t.messages = append(t.messages, fmt.Sprintf("%s: Exiting... Showing all messages",
				time.Now().Format("15:04:05")))
			time.Sleep(1 * time.Second) // Dar tiempo para mostrar los mensajes
			return t, tea.Quit
		case "t":
			// Acción especial al presionar 't'
			t.messages = append(t.messages, fmt.Sprintf("%s: You have activated a special action!",
				time.Now().Format("15:04:05")))
		case "b":
			// Acción para abrir el navegador
			t.messages = append(t.messages, fmt.Sprintf("%s: Opening browser...",
				time.Now().Format("15:04:05")))
		default:
			// Registra cualquier otra tecla presionada
			t.messages = append(t.messages, fmt.Sprintf("%s: Key pressed: %s",
				time.Now().Format("15:04:05"), msg.String()))
		}
	case tickMsg:
		// Actualiza el tiempo cada segundo
		now := time.Now()
		t.currentTime = now.Format("15:04:05")
		// Actualiza el footer
		t.footer = fmt.Sprintf("Press 'ESC' to exit | 't' Tinygo Compiler Activated: %s | 'b' Browser | ",
			t.currentTime)
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	}

	return t, tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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

// Métodos de impresión con colores
func (t *Terminal) PrintOK(messages ...string) {
	msg := t.joinMessages(messages...)
	t.messages = append(t.messages, okStyle.Render(msg))
	t.forceUpdate()
}

func (t *Terminal) PrintWarning(messages ...string) {
	msg := t.joinMessages(messages...)
	t.messages = append(t.messages, warnStyle.Render(msg))
	t.forceUpdate()
}

func (t *Terminal) PrintError(messages ...string) {
	msg := t.joinMessages(messages...)
	t.messages = append(t.messages, errStyle.Render(msg))
	t.forceUpdate()
}

func (t *Terminal) PrintInfo(messages ...string) {
	msg := t.joinMessages(messages...)
	t.messages = append(t.messages, infoStyle.Render(msg))
	t.forceUpdate()
}

func (t *Terminal) joinMessages(messages ...string) string {
	var message, space string
	for _, m := range messages {
		message += space + m
		space = " "
	}
	return message
}

func (t *Terminal) forceUpdate() {
	if t.tea != nil {
		t.tea.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		time.Sleep(100 * time.Millisecond)
	}
}

// View renderiza la interfaz
func (t Terminal) View() string {
	// Dimensiones mínimas requeridas
	minWidth := 40
	minHeight := 10

	if t.width < minWidth || t.height < minHeight {
		return fmt.Sprintf("Terminal too small. Minimum size: %dx%d", minWidth, minHeight)
	}

	// Dimensiones fijas para header y footer
	headerHeight := 3
	footerHeight := 3
	contentHeight := t.height - headerHeight - footerHeight

	// Ancho efectivo del contenido
	contentWidth := t.width - 4 // 4 para los bordes

	// Header siempre visible
	header := headerFooterStyle.
		Width(contentWidth).
		Render(fmt.Sprintf("🚀 GoDEV - %s", t.currentTime))

	// Footer siempre visible
	footer := headerFooterStyle.
		Width(contentWidth).
		Render(t.footer)

	// Manejo de mensajes con scroll
	visibleMessages := contentHeight - 2
	start := 0
	if len(t.messages) > visibleMessages {
		start = len(t.messages) - visibleMessages
	}

	// Procesar mensajes manteniendo el estilo
	var contentLines []string
	for i := start; i < len(t.messages); i++ {
		msg := t.messages[i]
		if msg != "" {
			// Dividir mensajes largos en múltiples líneas
			maxLineWidth := contentWidth - 6 // Espacio para bullet y padding
			for len(msg) > maxLineWidth {
				line := msg[:maxLineWidth]
				contentLines = append(contentLines, messageStyle.Render("• "+line))
				msg = msg[maxLineWidth:]
			}
			if len(msg) > 0 {
				contentLines = append(contentLines, messageStyle.Render("• "+msg))
			}
		}
	}

	// Asegurar que hay suficientes líneas para llenar el espacio
	for len(contentLines) < visibleMessages {
		contentLines = append(contentLines, "")
	}

	// Si hay más líneas que espacio visible, truncar
	if len(contentLines) > visibleMessages {
		contentLines = contentLines[len(contentLines)-visibleMessages:]
	}

	content := strings.Join(contentLines, "\n")

	// Área de contenido con scroll
	contentArea := borderStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	// Unir las secciones manteniendo la alineación
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentArea,
		footer,
	)
}

// inicia una nueva terminal
func (h *handler) NewTerminal() {
	h.terminal = &Terminal{
		messages:    make([]string, 0),
		footer:      "Starting...",
		currentTime: time.Now().Format("15:04:05"),
		tickCount:   0,
	}

	options := []tea.ProgramOption{tea.WithAltScreen()}
	h.terminal.tea = tea.NewProgram(h.terminal, options...)
}

// inicia la aplicación de terminal
func (h *handler) RunTerminal() {
	if _, err := h.terminal.tea.Run(); err != nil {
		fmt.Printf("Error running the application: %v\n", err)
		os.Exit(1)
	}
}
