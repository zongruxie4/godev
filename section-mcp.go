package app

// AddSectionMCP creates the MCP tab in the TUI
func (h *Handler) AddSectionMCP() any {
	section := h.Tui.NewTabSection("MCP", "MCP Daemon Status")
	h.SectionMCP = section
	return section
}
