package godev

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
		Render(t.tabs[t.activeTab].footer)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		tabs,
		contentArea,
		footer,
	)
}
