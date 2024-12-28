package godev

import (
	"fmt"
	"os"
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
}

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

// Define estilos para el borde del contenido header y footer
var borderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("10")) // Verde claro

var headerFooterStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("7")). // Gris claro de fondo
	Foreground(lipgloss.Color("0")). // Texto negro
	Padding(0, 1)

func (t *Terminal) updateHeaderFooterStyle() {
	headerFooterStyle = headerFooterStyle.Width(t.width - 4) // Ajustar al ancho del borde
}

// View renderiza la interfaz
func (t Terminal) View() string {
	if t.width == 0 || t.height == 0 {
		return "Terminal too small"
	}

	// Construye el header
	header := borderStyle.
		Width(t.width).
		Render(
			headerFooterStyle.
				Render(fmt.Sprintf("GoDEV: %s", t.currentTime)),
		)

	// Construye el footer
	footer := borderStyle.
		Width(t.width).
		Render(
			headerFooterStyle.
				Render(t.footer),
		)

	// Calcula la altura disponible para los mensajes
	// Restamos 4 líneas: 1 para header, 1 para su borde, 1 para footer, 1 para su borde
	messageHeight := t.height - 4

	// Asegura que messageHeight no sea negativo
	if messageHeight < 0 {
		messageHeight = 0
	}

	// Determina el punto de inicio para mostrar los mensajes
	start := 0
	if len(t.messages) > messageHeight {
		start = len(t.messages) - messageHeight
	}

	// Crea un área de contenido para los mensajes
	content := ""

	// Muestra los últimos mensajes que caben en la pantalla
	for i := start; i < len(t.messages); i++ {
		// Aplica estilo a cada mensaje
		msgStyle := lipgloss.NewStyle().
			Width(t.width - 4).              // Ancho ajustado
			PaddingLeft(2).                  // Margen izquierdo
			Foreground(lipgloss.Color("15")) // Color blanco
		content += msgStyle.Render(t.messages[i]) + "\n"
	}

	// Construye la vista completa
	s := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		borderStyle.
			Width(t.width).
			Height(messageHeight).
			Render(content),
		footer,
	)

	return s
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
	h.tea = tea.NewProgram(h.terminal, options...)

}

// inicia la aplicación de terminal
func (h *handler) RunTerminal() {
	if _, err := h.tea.Run(); err != nil {
		fmt.Printf("Error running the application: %v\n", err)
		os.Exit(1)
	}
}
