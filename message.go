package godev

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// TerminalMessage representa un mensaje en la terminal
type TerminalMessage struct {
	Content string
	Type    MessageType
	Time    time.Time
}

// Msg sends a normal message to the terminal
func (h *Terminal) Msg(messages ...any) {
	h.SendMessage(joinMessages(messages...), NormMsg)
}

// MsgError sends an error message to the terminal
func (h *Terminal) MsgError(messages ...any) {
	h.SendMessage(joinMessages(messages...), ErrorMsg)
}

// MsgWarning sends a warning message to the terminal
func (h *Terminal) MsgWarning(messages ...any) {
	h.SendMessage(joinMessages(messages...), WarnMsg)
}

// MsgInfo sends an informational message to the terminal
func (h *Terminal) MsgInfo(messages ...any) {
	h.SendMessage(joinMessages(messages...), InfoMsg)
}

// MsgOk sends a success message to the terminal
func (h *Terminal) MsgOk(messages ...any) {
	h.SendMessage(joinMessages(messages...), OkMsg)
}

func joinMessages(messages ...any) (message string) {
	var space string
	for _, m := range messages {
		message += space + fmt.Sprint(m)
		space = " "
	}
	return
}

// SendMessage envía un mensaje al terminal
func (t *Terminal) SendMessage(content string, msgType MessageType) {

	t.messagesChan <- TerminalMessage{
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
func (t *Terminal) formatMessage(msg TerminalMessage) string {
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
		strings.Contains(lowerContent, "starting") ||
		strings.Contains(lowerContent, "initializing") {
		return InfoMsg
	}

	return OkMsg
}

// Write implementa io.Writer para capturar la salida de otros procesos
func (t *Terminal) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		// Detectar automáticamente el tipo de mensaje
		msgType := detectMessageType(msg)
		t.SendMessage(msg, msgType)

		// Si es un error, escribirlo en el archivo de log
		if msgType == ErrorMsg {
			logFile, err := os.OpenFile(path.Join(outputDir, outputName+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer logFile.Close()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				logFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msg))
			}
		}

	}
	return len(p), nil
}
