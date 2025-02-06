package godev

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const background = "#FF6600" // orange
const foreGround = "#F4F4F4" //white
const black = "#000000"      //black

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
func (h *handler) View() string {
	if h.tui.width < 40 || h.tui.height < 10 {
		return "TextUserInterface too small. Minimum size: 40x10"
	}

	headerHeight := 3
	footerHeight := 3
	// contentHeight := h.tui.height - footerHeight
	contentHeight := h.tui.height - headerHeight - footerHeight
	contentWidth := h.tui.width - 2

	// Pestañas
	tabs := h.tui.renderTabs()

	var content string
	if h.tui.activeTab == 0 {
		content = h.tui.renderConfigFields()
	} else {
		// Contenido de la pestaña activa
		visibleMessages := contentHeight - 1
		start := 0
		activeContent := h.tui.tabs[h.tui.activeTab].content
		if len(activeContent) > visibleMessages {
			start = len(activeContent) - visibleMessages
		}

		var contentLines []string
		for i := start; i < len(activeContent); i++ {
			formattedMsg := h.tui.formatMessage(activeContent[i])
			contentLines = append(contentLines, messageStyle.Render(formattedMsg))
		}

		content = strings.Join(contentLines, "\n")
	}

	contentArea := borderStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	// Footer
	footer := headerFooterStyle.
		Width(contentWidth).
		// Render(t.tabs[t.activeTab].footer)
		Render(h.tui.renderFooter(h.tui.tabs[h.tui.activeTab]))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		// header,
		tabs,
		contentArea,
		footer,
	)
}

func (t *TextUserInterface) renderTabs() string {
	var leftTab, centerTabs, rightTab []string

	// Tab izquierdo (GODEV)
	leftStyle := tab
	if t.activeTab == 0 {
		leftStyle = activeTab
	}
	leftTab = append(leftTab, leftStyle.Render(t.tabs[0].title))

	// Tabs centrales (BUILD, TEST, DEPLOY)
	for i := 1; i < len(t.tabs)-1; i++ {
		style := tab
		if i == t.activeTab {
			style = activeTab
		}
		centerTabs = append(centerTabs, style.Render(t.tabs[i].title))
	}

	// Tab derecho (HELP)
	rightStyle := tab
	if t.activeTab == len(t.tabs)-1 {
		rightStyle = activeTab
	}
	rightTab = append(rightTab, rightStyle.Render(t.tabs[len(t.tabs)-1].title))

	// Combinar todo con espaciado apropiado
	centerSection := lipgloss.JoinHorizontal(lipgloss.Top, centerTabs...)

	// Calcular espaciado para centrar la sección central
	totalWidth := t.width - lipgloss.Width(leftTab[0]) - lipgloss.Width(rightTab[0]) - 4
	centerWidth := lipgloss.Width(centerSection)
	padding := (totalWidth - centerWidth) / 2

	spacer := strings.Repeat(" ", padding)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftTab[0],
		spacer,
		centerSection,
		spacer,
		rightTab[0],
	)
}

func (t *TextUserInterface) renderConfigFields() string {
	var lines []string

	style := lipgloss.NewStyle().
		Padding(0, 2)

	selectedStyle := style
	selectedStyle = selectedStyle.
		Bold(true).
		Background(lipgloss.Color(background)).
		Foreground(lipgloss.Color(foreGround))

	editingStyle := selectedStyle
	editingStyle = editingStyle.
		Foreground(lipgloss.Color(black))

	for i, field := range t.tabs[0].configs {
		line := fmt.Sprintf("%s: %s", field.label, field.value)

		if t.activeTab == 0 {
			if i == t.activeConfig {
				if t.editingConfig {
					cursorPos := field.cursor + len(field.label) + 2
					line = line[:cursorPos] + "▋" + line[cursorPos:]
					line = editingStyle.Render(line)
				} else {
					line = selectedStyle.Render(line)
				}
			} else {
				line = style.Render(line)
			}
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (t *TextUserInterface) renderFooter(tab Tab) string {

	if tab.footer != "" {
		return tab.footer
	}

	var footerParts []string
	for _, action := range tab.actions {
		status := "○" // inactive
		if action.active {
			status = "●" // active
		}
		footerParts = append(footerParts, fmt.Sprintf("'%s' %s %s", action.shortCuts, action.message, status))
	}
	return strings.Join(footerParts, " | ")
}
