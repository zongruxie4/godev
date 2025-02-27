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

	contentBorder     lipgloss.Border
	headerTitleStyle  lipgloss.Style
	footerInfoStyle   lipgloss.Style
	textContentStyle  lipgloss.Style
	lineHeadFootStyle lipgloss.Style // header right and footer left line
}

func NewTuiStyle(cs *ColorStyle) *TuiStyle {
	t := &TuiStyle{
		ColorStyle: cs,
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

	t.headerTitleStyle = lipgloss.NewStyle().
		Padding(0, 1).
		BorderForeground(lipgloss.Color(t.Highlight)).
		Background(lipgloss.Color(t.Highlight)).
		Foreground(lipgloss.Color(t.ForeGround))

	t.footerInfoStyle = t.headerTitleStyle

	// Estilo para los mensajes
	t.textContentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.ForeGround)).
		PaddingLeft(0)

	t.lineHeadFootStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Highlight))

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
	tabContent := h.TabSections[h.activeTab].tabContents
	var contentLines []string
	for _, content := range tabContent {
		formattedMsg := h.formatMessage(content)
		contentLines = append(contentLines, h.textContentStyle.Render(formattedMsg))
	}
	return strings.Join(contentLines, "\n")
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

	for indexSection, tabSection := range t.TabSections {

		// break different index
		if indexSection != t.activeTab {
			continue
		}

		for i, field := range tabSection.SectionFields {
			line := fmt.Sprintf("%s: %s", field.Label, field.Value)

			if i == tabSection.indexActiveEditField {
				if t.editingFieldValueInSection {
					cursorPos := field.cursor + len(field.Label) + 2
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

func (h *TextUserInterface) headerView() string {
	tab := h.TabSections[h.activeTab]
	title := h.headerTitleStyle.Render(tab.Title)
	line := h.lineHeadFootStyle.Render(strings.Repeat("─", max(0, h.viewport.Width-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (h *TextUserInterface) footerView() string {
	info := h.footerInfoStyle.Render(fmt.Sprintf("%3.f%%", h.viewport.ScrollPercent()*100))
	line := h.lineHeadFootStyle.Render(strings.Repeat("─", max(0, h.viewport.Width-lipgloss.Width(info))))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}
