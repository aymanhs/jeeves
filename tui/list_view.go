package tui

import (
	"fmt"
	"strings"

	"github.com/aymanhs/sys-tui/systemd"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) handleListKey(msg tea.KeyMsg) tea.Cmd {
	if len(m.filteredServices) == 0 && msg.String() != "R" && msg.String() != "/" && msg.String() != "a" {
		return nil
	}

	var selected *systemd.ServiceInfo
	if len(m.filteredServices) > 0 {
		selected = &m.filteredServices[m.selectedIndex]
	}

	switch msg.String() {
	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		} else {
			m.selectedIndex = len(m.filteredServices) - 1 // wrap around
		}

	case "down", "j":
		if m.selectedIndex < len(m.filteredServices)-1 {
			m.selectedIndex++
		} else {
			m.selectedIndex = 0 // wrap around
		}

	case "enter", "right", "l":
		if selected != nil {
			m.currentView = detailView
			m.detailFocusedSection = focusInfo
			m.activeDetail = selected // temporary details while fetching
			m.logs = "Loading logs..."
			m.logViewport.SetContent(m.logs)
			m.recalculateViewportSize()
			return m.fetchDetailsCmd(selected.Name)
		}

	case "s": // Start
		if selected != nil {
			return m.runActionCmd("start", selected.Name)
		}

	case "t": // Stop
		if selected != nil {
			return m.runActionCmd("stop", selected.Name)
		}

	case "r": // Restart
		if selected != nil {
			return m.runActionCmd("restart", selected.Name)
		}

	case "e": // Enable
		if selected != nil {
			return m.runActionCmd("enable", selected.Name)
		}

	case "d": // Disable
		if selected != nil {
			return m.runActionCmd("disable", selected.Name)
		}

	case "/": // Filter
		m.filtering = true
		m.searchInput.Focus()
		m.searchInput.SetValue(m.filterQuery)
		return textinput.Blink

	case "a": // Toggle view mode
		m.showMode = (m.showMode + 1) % 2
		m.filterServices()

	case "R": // Refresh
		return m.fetchServicesCmd()
	}

	return nil
}

func (m Model) renderListView() string {
	var s strings.Builder

	// 1. Header
	title := TitleStyle.Render("SYSTEMD SERVICES")
	modeStr := fmt.Sprintf("Bus: %s Mode", m.client.Mode().String())
	if m.loading {
		modeStr += " [LOADING...]"
	}
	subtitle := SubTitleStyle.Render(modeStr)
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle) + "\n\n")

	// 2. Search / Filter bar
	if m.filtering {
		s.WriteString(m.searchInput.View() + "\n\n")
	} else if m.filterQuery != "" {
		filterText := SearchPromptStyle.Render("Filter: ") + SearchInputStyle.Render(m.filterQuery)
		infoText := HelpDescStyle.Render(" (press / to edit, esc to clear)")
		s.WriteString(filterText + infoText + "\n\n")
	} else {
		s.WriteString(HelpDescStyle.Render("Press / to search/filter services, [a] to toggle view...") + "\n\n")
	}

	// 3. Mode display indicator
	var modeLabel string
	switch m.showMode {
	case showRunning:
		modeLabel = ActiveBadge.Render("Showing: Running Services Only")
	default:
		modeLabel = HelpDescStyle.Render("Showing: All Services")
	}
	s.WriteString(modeLabel + "\n\n")

	// 4. Services Table
	if len(m.filteredServices) == 0 {
		s.WriteString("\n  No services found matching filters.\n")
	} else {
		// Table Columns configuration
		colStatusW := 12
		colNameW := 35
		colSubW := 12
		colEnableW := 12
		colDescW := m.width - colStatusW - colNameW - colSubW - colEnableW - 6
		if colDescW < 15 {
			colDescW = 15 // min fallback
		}

		// Table Header
		headerRow := fmt.Sprintf("  %s %s %s %s %s",
			padRight("STATUS", colStatusW),
			padRight("SERVICE NAME", colNameW),
			padRight("SUB STATE", colSubW),
			padRight("ENABLE STATE", colEnableW),
			padRight("DESCRIPTION", colDescW),
		)
		s.WriteString(TableHeaderStyle.Render(headerRow) + "\n")
		s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(strings.Repeat("-", m.width)) + "\n")

		// Calculate scrolling viewport for list
		maxRows := m.height - 12
		if maxRows < 3 {
			maxRows = 3 // sane minimum
		}

		start := m.scrollOffset
		end := start + maxRows
		if end > len(m.filteredServices) {
			end = len(m.filteredServices)
		}

		// Render rows
		renderedRows := 0
		for i := start; i < end; i++ {
			svc := m.filteredServices[i]
			renderedRows++

			// Format columns
			statusIndicator := m.formatActiveState(svc.ActiveState)
			nameStr := svc.Name
			subStateStr := svc.SubState
			enableStateStr := m.formatEnableState(svc.UnitFileState)
			descStr := svc.Description

			rowText := fmt.Sprintf("%s %s %s %s %s",
				padRight(statusIndicator, colStatusW),
				padRight(nameStr, colNameW),
				padRight(subStateStr, colSubW),
				padRight(enableStateStr, colEnableW),
				padRight(descStr, colDescW),
			)

			if i == m.selectedIndex {
				// Selected Row
				s.WriteString(SelectedRowStyle.Render("➜ "+rowText) + "\n")
			} else {
				// Regular Row
				s.WriteString(RowStyle.Render("  "+rowText) + "\n")
			}
		}

		// Pad with empty rows to keep status and help pinned to bottom
		for i := renderedRows; i < maxRows; i++ {
			s.WriteString("\n")
		}
	}

	// 5. Status banner (errors or success) - always exactly 2 lines
	if m.statusMsg != "" {
		if m.statusIsErr {
			s.WriteString("\n" + ErrorBanner.Render("⚠ "+m.statusMsg))
		} else {
			s.WriteString("\n" + SuccessBanner.Render("✔ "+m.statusMsg))
		}
	} else {
		s.WriteString("\n\n")
	}

	// 6. Help Footer
	s.WriteString(m.renderListFooter())

	return DocStyle.Render(s.String())
}

func (m Model) formatActiveState(state string) string {
	switch state {
	case "active":
		return ActiveBadge.Render("● active")
	case "failed":
		return FailedBadge.Render("● failed")
	case "activating", "deactivating", "reloading":
		return WarningBadge.Render("● " + state)
	case "inactive":
		return InactiveBadge.Render("● inactive")
	default:
		return InactiveBadge.Render("● " + state)
	}
}

func (m Model) formatEnableState(state string) string {
	switch state {
	case "enabled":
		return EnabledBadge.Render("enabled")
	case "disabled":
		return DisabledBadge.Render("disabled")
	case "static":
		return StaticBadge.Render("static")
	case "masked":
		return WarningBadge.Render("masked")
	case "alias":
		return StaticBadge.Render("alias")
	case "generated":
		return StaticBadge.Render("generated")
	case "":
		return InactiveBadge.Render("-")
	default:
		return InactiveBadge.Render(state)
	}
}

func (m Model) renderListFooter() string {
	var f strings.Builder
	f.WriteString("\n")

	keys := []string{
		"↑/↓/k/j", "Navigate",
		"Enter/→", "Details/Logs",
		"s", "Start",
		"t", "Stop",
		"r", "Restart",
		"e", "Enable",
		"d", "Disable",
		"/", "Filter",
		"a", "Toggle Filter",
		"R", "Refresh",
		"q/Ctrl+C", "Quit",
	}

	items := []string{}
	for i := 0; i < len(keys); i += 2 {
		items = append(items, fmt.Sprintf("%s %s", HelpKeyStyle.Render(keys[i]), HelpDescStyle.Render(keys[i+1])))
	}

	f.WriteString(FooterStyle.Render(strings.Join(items, "  •  ")))
	return f.String()
}

func padRight(s string, width int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth > width {
		if strings.Contains(s, "\x1b") {
			return s // Don't truncate styled strings to avoid breaking ANSI codes
		}
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s + strings.Repeat(" ", width-visualWidth)
}
