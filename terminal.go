package godev

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		case "q", "ctrl+c":
			return t, tea.Quit
		case "t":
			// Acción especial al presionar 't'
			t.messages = append(t.messages, fmt.Sprintf("%s: ¡Has activado una acción especial!",
				time.Now().Format("15:04:05")))
		default:
			// Registra cualquier otra tecla presionada
			t.messages = append(t.messages, fmt.Sprintf("%s: Tecla presionada: %s",
				time.Now().Format("15:04:05"), msg.String()))
		}
		// **Eliminamos la limitación del historial de mensajes**
		// if len(t.messages) > 10 {
		// 	t.messages = t.messages[1:]
		// }
	case tickMsg:
		// Actualiza el tiempo cada segundo
		now := time.Now()
		t.currentTime = now.Format("15:04:05")
		t.tickCount++

		// Cada 5 segundos muestra un mensaje de tiempo transcurrido
		if t.tickCount%5 == 0 {
			t.messages = append(t.messages, fmt.Sprintf("%s: Han pasado 5 segundos", t.currentTime))
			// **Eliminamos la limitación del historial de mensajes aquí también**
			// if len(t.messages) > 10 {
			// 	t.messages = t.messages[1:]
			// }
		}

		// Actualiza el footer
		t.footer = fmt.Sprintf("Presiona 'q' para salir | 't' para acción especial | Tiempo actual: %s",
			t.currentTime)
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	}

	return t, tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// View renderiza la interfaz
// func (t Terminal) View() string {
// 	// Construye la vista principal
// 	s := fmt.Sprintf("Terminal Simple - Tiempo Actual: %s\n\n", t.currentTime)

// 	// Calcula la altura disponible para los mensajes
// 	messageHeight := t.height - 3 // 1 línea para el título, 1 para el espacio después del título, 1 para el footer

// 	// Muestra los mensajes o espacios vacíos para llenar la altura disponible
// 	numMessages := len(t.messages)
// 	for i := 0; i < messageHeight; i++ {
// 		if i < numMessages {
// 			s += t.messages[i] + "\n"
// 		} else {
// 			s += "\n"
// 		}
// 	}

// 	// Agrega el footer
// 	s += t.footer

// 	return s
// }

// View renderiza la interfaz
func (t Terminal) View() string {
	// Construye la vista principal
	s := fmt.Sprintf("Terminal Simple - Tiempo Actual: %s\n\n", t.currentTime)

	// Calcula la altura disponible para los mensajes
	messageHeight := t.height - 3 // 1 línea para el título, 1 para el espacio después del título, 1 para el footer

	// Determina el punto de inicio para mostrar los mensajes
	start := 0
	if len(t.messages) > messageHeight {
		start = len(t.messages) - messageHeight
	}

	// Muestra los últimos mensajes que caben en la pantalla
	for i := start; i < len(t.messages); i++ {
		s += t.messages[i] + "\n"
	}

	// Rellena el espacio restante con líneas vacías si hay menos mensajes que la altura disponible
	for i := len(t.messages); i < messageHeight; i++ {
		s += "\n"
	}

	// Agrega el footer
	s += t.footer

	return s
}

// RunTerminal inicia la aplicación
func RunTerminal() {
	terminal := &Terminal{
		messages:    make([]string, 0),
		footer:      "Iniciando...",
		currentTime: time.Now().Format("15:04:05"),
		tickCount:   0,
	}

	options := []tea.ProgramOption{tea.WithAltScreen()}
	p := tea.NewProgram(terminal, options...)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error al ejecutar la aplicación: %v\n", err)
		os.Exit(1)
	}
}
