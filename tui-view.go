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

	activeTabIndex = lipgloss.NewStyle().
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

	// Estilo para el header y SectionFooter
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
func (h *TextUserInterface) View() string {
	if h.width < 40 || h.height < 10 {
		return "TextUserInterface too small. Minimum size: 40x10"
	}

	headerHeight := 3
	footerHeight := 3
	contentHeight := h.height - headerHeight - footerHeight
	contentWidth := h.width - 2

	// Render components
	tabsSection := h.renderTabs()
	hasFields := len(h.tabsSection[h.activeTabIndex].sectionFields) > 0
	contentArea := h.renderContent(contentWidth, contentHeight, hasFields)

	sectionFooter := headerFooterStyle.
		Width(contentWidth).
		Render(h.renderFooter(h.tabsSection[h.activeTabIndex]))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		tabsSection,
		contentArea,
		sectionFooter,
	)
}

// renderContent renderiza el área de contenido según si tiene campos o no
func (h *TextUserInterface) renderContent(contentWidth, contentHeight int, hasFields bool) string {
	if !hasFields {
		content := h.renderContentMessages(contentHeight, h.tabsSection[h.activeTabIndex].terminalPrints)
		return borderStyle.
			Width(contentWidth).
			Height(contentHeight).
			Render(content)
	}

	// Split layout (30/70)
	leftWidth := (contentWidth * 30) / 100
	rightWidth := contentWidth - leftWidth - 1

	// Left form section
	leftContent := h.renderLeftSectionForm()
	leftArea := borderStyle.
		Width(leftWidth).
		Height(contentHeight).
		Render(leftContent)

	// Right content section
	rightContent := ""
	if h.activeTabIndex > 0 {
		rightContent = h.renderContentMessages(contentHeight, h.tabsSection[h.activeTabIndex].terminalPrints)
	}

	rightArea := borderStyle.
		Width(rightWidth).
		Height(contentHeight).
		Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftArea, rightArea)
}

// renderContentMessages renderiza los mensajes para una sección de contenido
func (h *TextUserInterface) renderContentMessages(contentHeight int, messages []TerminalPrint) string {
	visibleMessages := contentHeight - 1
	start := 0
	if len(messages) > visibleMessages {
		start = len(messages) - visibleMessages
	}

	var contentLines []string
	for i := start; i < len(messages); i++ {
		formattedMsg := h.formatMessage(messages[i])
		contentLines = append(contentLines, messageStyle.Render(formattedMsg))
	}
	return strings.Join(contentLines, "\n")
}

// View renderiza la interfaz
func (h *TextUserInterface) ViewOLD() string {
	if h.width < 40 || h.height < 10 {
		return "TextUserInterface too small. Minimum size: 40x10"
	}

	headerHeight := 3
	footerHeight := 3
	// contentHeight := h.height - footerHeight
	contentHeight := h.height - headerHeight - footerHeight
	contentWidth := h.width - 2

	// Pestañas
	tabsSection := h.renderTabs()

	var terminalPrints string
	if h.activeTabIndex == 0 {
		terminalPrints = h.renderLeftSectionForm()
	} else {
		// Contenido de la pestaña activa
		visibleMessages := contentHeight - 1
		start := 0
		activeContent := h.tabsSection[h.activeTabIndex].terminalPrints
		if len(activeContent) > visibleMessages {
			start = len(activeContent) - visibleMessages
		}

		var contentLines []string
		for i := start; i < len(activeContent); i++ {
			formattedMsg := h.formatMessage(activeContent[i])
			contentLines = append(contentLines, messageStyle.Render(formattedMsg))
		}

		terminalPrints = strings.Join(contentLines, "\n")
	}

	contentArea := borderStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(terminalPrints)

	// Footer
	SectionFooter := headerFooterStyle.
		Width(contentWidth).
		// Render(t.tabsSection[t.activeTabIndex].SectionFooter)
		Render(h.renderFooter(h.tabsSection[h.activeTabIndex]))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		// header,
		tabsSection,
		contentArea,
		SectionFooter,
	)
}

func (t *TextUserInterface) renderTabs() string {
	var leftTab, centerTabs, rightTab []string

	// TabSection izquierdo (GODEV)
	leftStyle := tab
	if t.activeTabIndex == 0 {
		leftStyle = activeTabIndex
	}
	leftTab = append(leftTab, leftStyle.Render(t.tabsSection[0].title))

	// Tabs centrales (BUILD, TEST, DEPLOY)
	for i := 1; i < len(t.tabsSection)-1; i++ {
		style := tab
		if i == t.activeTabIndex {
			style = activeTabIndex
		}
		centerTabs = append(centerTabs, style.Render(t.tabsSection[i].title))
	}

	// TabSection derecho (HELP)
	rightStyle := tab
	if t.activeTabIndex == len(t.tabsSection)-1 {
		rightStyle = activeTabIndex
	}
	rightTab = append(rightTab, rightStyle.Render(t.tabsSection[len(t.tabsSection)-1].title))

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

func (t *TextUserInterface) renderLeftSectionForm() string {
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

	for i, field := range t.tabsSection[0].sectionFields {
		line := fmt.Sprintf("%s: %s", field.label, field.value)

		if t.activeTabIndex == 0 {
			if i == t.indexActiveEditField {
				if t.editingFieldValueInSection {
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

func (t *TextUserInterface) renderFooter(tab TabSection) string {

	if tab.SectionFooter != "" {
		return tab.SectionFooter
	}

	var footerParts []string
	// for _, field := range tab.sectionFields {
	// 	status := "○" // inactive
	// 	if field.isOpenedStatus {
	// 		status = "●" // isOpenedStatus
	// 	}
	// 	footerParts = append(footerParts, fmt.Sprintf("'%s' %s %s", field.ShortCut, field.label, status))
	// }
	return strings.Join(footerParts, " | ")
}
