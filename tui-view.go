package godev

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ColorStyle struct {
	ForeGround string // eg: #F4F4F4
	Background string // eg: #000000
	Highlight  string // eg: #FF6600
	Lowlight   string // eg: #666666
}

type TuiStyle struct {
	*ColorStyle

	activeTabBorder   lipgloss.Border
	tabBorder         lipgloss.Border
	contentBorder     lipgloss.Border
	normalTabStyle    lipgloss.Style
	activeTabStyle    lipgloss.Style
	borderStyle       lipgloss.Style
	headerFooterStyle lipgloss.Style
	headerTitleStyle  lipgloss.Style
	footerInfoStyle   lipgloss.Style
	messageStyle      lipgloss.Style
}

// eg: colorHighlight #FF6600 colorForeGround  colorBackGround
func NewTuiStyle(cs *ColorStyle) *TuiStyle {
	t := &TuiStyle{
		ColorStyle: cs,
	}

	// Estilos para las pestañas
	t.activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	t.tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	// El borde del contenido necesita conectarse con las pestañas
	t.contentBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	t.normalTabStyle = lipgloss.NewStyle().
		Border(t.tabBorder, true).
		BorderForeground(lipgloss.Color(t.Highlight)).
		Padding(0)

	t.activeTabStyle = lipgloss.NewStyle().
		Border(t.activeTabBorder, true).
		BorderForeground(lipgloss.Color(t.Highlight)).
		Bold(true).
		Background(lipgloss.Color(t.Highlight)).
		Foreground(lipgloss.Color(t.ForeGround)).
		Padding(0)

	t.headerTitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	t.footerInfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return t.headerTitleStyle.BorderStyle(b)
	}()

	// Estilo para el borde principal
	t.borderStyle = lipgloss.NewStyle().
		Border(t.contentBorder).
		BorderForeground(lipgloss.Color(t.Highlight)).
		Padding(0)

	// Estilo para el header y SectionFooter
	t.headerFooterStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Highlight)).
		Foreground(lipgloss.Color(t.ForeGround)).
		Bold(true)
		// Padding(0)

	// Estilo para los mensajes
	t.messageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.ForeGround)).
		PaddingLeft(0)

	return t
}

func (h *TextUserInterface) View() string {
	if !h.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", h.headerView(), h.viewport.View(), h.footerView())
	// return fmt.Sprintf("%s\n%s\n%s", h.headerView(), h.ContentView(), h.footerView())
}

// ContentView renderiza los mensajes para una sección de contenido
func (h *TextUserInterface) ContentView() string {
	tabContent := h.tabsSection[h.activeTab].tabContents
	contentHeight := len(tabContent)
	visibleMessages := contentHeight
	start := 0
	if len(tabContent) > visibleMessages {
		start = len(tabContent) - visibleMessages
	}

	var contentLines []string
	for i := start; i < len(tabContent); i++ {
		formattedMsg := h.formatMessage(tabContent[i])
		contentLines = append(contentLines, h.messageStyle.Render(formattedMsg))
	}
	return strings.Join(contentLines, "\n")
}

func (h *TextUserInterface) headerView() string {

	tab := h.tabsSection[h.activeTab]

	title := h.headerTitleStyle.Render(tab.title)
	line := strings.Repeat("─", max(0, h.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (h *TextUserInterface) headerViewOLD() string {
	var leftTab, centerTabs, rightTab []string

	// TabSection izquierdo (GODEV)
	leftStyle := h.normalTabStyle
	if h.activeTab == 0 {
		leftStyle = h.activeTabStyle
	}
	leftTab = append(leftTab, leftStyle.Render(h.tabsSection[0].title))

	// If only one tab exists, return just the left tab with appropriate spacing
	if len(h.tabsSection) == 1 {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftTab[0],
		)
	}

	// Rest of the existing code for multiple tabs...
	// Central tabs processing example: (BUILD, TEST, DEPLOY)
	for i := 1; i < len(h.tabsSection)-1; i++ {
		style := h.normalTabStyle
		if i == h.activeTab {
			style = h.activeTabStyle
		}
		centerTabs = append(centerTabs, style.Render(h.tabsSection[i].title))
	}

	// Right tab processing (example HELP)
	rightStyle := h.normalTabStyle
	if h.activeTab == len(h.tabsSection)-1 {
		rightStyle = h.activeTabStyle
	}
	rightTab = append(rightTab, rightStyle.Render(h.tabsSection[len(h.tabsSection)-1].title))

	// Combine everything with appropriate spacing
	centerSection := lipgloss.JoinHorizontal(lipgloss.Top, centerTabs...)
	// Calcular espaciado para centrar la sección central
	totalWidth := h.viewport.Width - lipgloss.Width(leftTab[0]) - lipgloss.Width(rightTab[0]) - 4
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
		Background(lipgloss.Color(t.Highlight)).
		Foreground(lipgloss.Color(t.ForeGround))

	editingStyle := selectedStyle
	editingStyle = editingStyle.
		Foreground(lipgloss.Color(t.Background))

	for indexSection, tabSection := range t.tabsSection {

		// break different index
		if indexSection != t.activeTab {
			continue
		}

		for i, field := range tabSection.sectionFields {
			line := fmt.Sprintf("%s: %s", field.label, field.value)

			if i == tabSection.indexActiveEditField {
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

			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func (h *TextUserInterface) footerView() string {
	info := h.footerInfoStyle.Render(fmt.Sprintf("%3.f%%", h.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, h.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}
