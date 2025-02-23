package godev

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Print sends a normal Label or error to the tui
func (h *TextUserInterface) Print(messages ...any) {
	msgType := NormMsg
	newMessages := make([]any, 0, len(messages))

	for _, msg := range messages {
		if str, isString := msg.(string); isString {

			switch strings.ToLower(str) {
			case "error":
				msgType = ErrorMsg
				continue
			case "warning", "debug":
				msgType = WarnMsg
				continue
			case "info":
				msgType = InfoMsg
				continue
			case "ok":
				msgType = OkMsg
				continue
			}
		}
		if _, isError := msg.(error); isError {
			msgType = ErrorMsg
		}

		newMessages = append(newMessages, msg)
	}

	h.SendMessage(joinMessages(newMessages...), msgType)
}

// PrintError sends an error Label to the tui
func (h *TextUserInterface) PrintError(messages ...any) {
	h.SendMessage(joinMessages(messages...), ErrorMsg)
}

// PrintWarning sends a warning Label to the tui
func (h *TextUserInterface) PrintWarning(messages ...any) {
	h.SendMessage(joinMessages(messages...), WarnMsg)
}

// PrintInfo sends an informational Label to the tui
func (h *TextUserInterface) PrintInfo(messages ...any) {
	h.SendMessage(joinMessages(messages...), InfoMsg)
}

// PrintOK sends a success Label to the tui
func (h *TextUserInterface) PrintOK(messages ...any) {
	h.SendMessage(joinMessages(messages...), OkMsg)
}

func joinMessages(messages ...any) (Label string) {
	var space string
	for _, m := range messages {
		Label += space + fmt.Sprint(m)
		space = " "
	}
	return
}

// SendMessage envía un mensaje al tui
func (t *TextUserInterface) SendMessage(content string, msgType MessageType) {

	t.messagesChan <- TerminalPrint{
		Content: content,
		Type:    msgType,
		Time:    time.Now(),
	}
}

// Definir estilos con colores más intensos
var (
	okStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00")) // Verde brillante

	errStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF0000")) // Rojo brillante

	warnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFF00")) // Amarillo brillante

	infoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(background)) //

	normStyle = lipgloss.NoColor{}

	timeStyle = lipgloss.NewStyle().Foreground(
		lipgloss.Color("#666666"),
	)
)

// MessageType define el tipo de mensaje
type MessageType string

const (
	NormMsg  MessageType = "normal"
	InfoMsg  MessageType = "info"
	ErrorMsg MessageType = "error"
	WarnMsg  MessageType = "warn"
	OkMsg    MessageType = "ok"
)

// formatMessage formatea un mensaje según su tipo
func (t *TextUserInterface) formatMessage(msg TerminalPrint) string {
	timeStr := timeStyle.Render(fmt.Sprintf("%s", msg.Time.Format("15:04:05")))
	// content := fmt.Sprintf("[%s] %s", timeStr, msg.Content)

	switch msg.Type {
	case ErrorMsg:
		msg.Content = errStyle.Render(msg.Content)
	case WarnMsg:
		msg.Content = warnStyle.Render(msg.Content)
	case InfoMsg:
		msg.Content = infoStyle.Render(msg.Content)
	case OkMsg:
		msg.Content = okStyle.Render(msg.Content)
		// default:
		// 	msg.Content= msg.Content
	}

	return fmt.Sprintf("%s %s", timeStr, msg.Content)
}

// Función para detectar el tipo de mensaje basado en su contenido
func detectMessageType(content string) MessageType {
	lowerContent := strings.ToLower(content)

	// Detectar errores
	if strings.Contains(lowerContent, "error") ||
		strings.Contains(lowerContent, "failed") ||
		strings.Contains(lowerContent, "exit status 1") ||
		strings.Contains(lowerContent, "undeclared") ||
		strings.Contains(lowerContent, "undefined") ||
		strings.Contains(lowerContent, "fatal") {
		return ErrorMsg
	}

	// Detectar advertencias
	if strings.Contains(lowerContent, "warning") ||
		strings.Contains(lowerContent, "warn") {
		return WarnMsg
	}

	// Detectar mensajes informativos
	if strings.Contains(lowerContent, "info") ||
		strings.Contains(lowerContent, " ...") ||
		strings.Contains(lowerContent, "starting") ||
		strings.Contains(lowerContent, "initializing") ||
		strings.Contains(lowerContent, "success") {
		return InfoMsg
	}

	return NormMsg
}

// Write implementa io.Writer para capturar la salida de otros procesos
func (h *handler) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		// Detectar automáticamente el tipo de mensaje
		msgType := detectMessageType(msg)

		if h.tui != nil {
			h.tui.SendMessage(msg, msgType)
		} else {
			fmt.Println(msg)
		}
		// Si es un error, escribirlo en el archivo de log
		if msgType == ErrorMsg {
			logFile, err := os.OpenFile(path.Join(h.ch.config.WebFilesFolder, h.ch.config.AppName+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer logFile.Close()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				logFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msg))
			}
		}

	}
	return len(p), nil
}
