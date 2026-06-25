package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) handleDetailKey(msg tea.KeyMsg) tea.Cmd {
	if m.activeDetail == nil {
		return nil
	}

	switch msg.String() {
	case "esc", "left", "h":
		m.currentView = listView
		// Refresh list when going back
		return m.fetchServicesCmd()

	case "tab":
		if m.detailFocusedSection == focusInfo {
			m.detailFocusedSection = focusLogs
		} else {
			m.detailFocusedSection = focusInfo
		}
		return nil

	case "s": // Start
		return m.runActionCmd("start", m.activeDetail.Name)

	case "t": // Stop
		return m.runActionCmd("stop", m.activeDetail.Name)

	case "r": // Restart
		return m.runActionCmd("restart", m.activeDetail.Name)

	case "e": // Enable
		return m.runActionCmd("enable", m.activeDetail.Name)

	case "d": // Disable
		return m.runActionCmd("disable", m.activeDetail.Name)

	case "R": // Refresh details & logs
		return m.fetchDetailsCmd(m.activeDetail.Name)
	}

	// If Logs panel is focused, send scrolling keys to viewport
	if m.detailFocusedSection == focusLogs {
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		return cmd
	}

	return nil
}

func (m Model) renderDetailView() string {
	if m.activeDetail == nil {
		return DocStyle.Render("\n  Loading details...")
	}

	var s strings.Builder

	// 1. Header
	title := TitleStyle.Render("SERVICE DETAILS")
	subtitle := SubTitleStyle.Render(m.activeDetail.Name)
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle) + "\n\n")

	// 2. Responsive Panels Layout
	mainContentHeight := m.height - 7
	if mainContentHeight < 6 {
		mainContentHeight = 6
	}

	var content string
	if m.width >= 100 {
		// Side-by-side layout
		leftWidth := 45
		rightWidth := m.width - leftWidth - 6
		if rightWidth < 20 {
			rightWidth = 20
		}

		leftBox := m.renderInfoPanel(leftWidth, mainContentHeight)
		rightBox := m.renderLogPanel(rightWidth, mainContentHeight)
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	} else {
		// Vertical layout
		width := m.width - 2
		if width < 20 {
			width = 20
		}
		infoBoxHeight := mainContentHeight / 2
		logBoxHeight := mainContentHeight - infoBoxHeight

		leftBox := m.renderInfoPanel(width, infoBoxHeight)
		rightBox := m.renderLogPanel(width, logBoxHeight)
		content = lipgloss.JoinVertical(lipgloss.Left, leftBox, rightBox)
	}
	s.WriteString(content + "\n")

	// 3. Status Banner - always exactly 2 lines
	if m.statusMsg != "" {
		if m.statusIsErr {
			s.WriteString(ErrorBanner.Render("⚠ "+m.statusMsg) + "\n")
		} else {
			s.WriteString(SuccessBanner.Render("✔ "+m.statusMsg) + "\n")
		}
	} else {
		s.WriteString("\n\n")
	}

	// 4. Footer
	s.WriteString(m.renderDetailFooter())

	return DocStyle.Render(s.String())
}

func (m Model) renderInfoPanel(width, height int) string {
	var sb strings.Builder

	// Header inside the panel
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [ Service Properties ] ") + "\n\n")

	// Helper to render key-value details
	renderDetailRow := func(key, val string) string {
		k := DetailKeyStyle.Render(key)
		v := DetailValStyle.Render(val)
		return fmt.Sprintf("%s : %s", k, v)
	}

	sb.WriteString(renderDetailRow("Description", m.activeDetail.Description) + "\n")
	sb.WriteString(renderDetailRow("Load State", m.activeDetail.LoadState) + "\n")
	sb.WriteString(renderDetailRow("Active State", m.formatActiveState(m.activeDetail.ActiveState)) + "\n")
	sb.WriteString(renderDetailRow("Sub State", m.activeDetail.SubState) + "\n")
	sb.WriteString(renderDetailRow("Enable State", m.formatEnableState(m.activeDetail.UnitFileState)) + "\n")

	// Active Since (Uptime) / Inactive Since (Downtime)
	if m.activeDetail.ActiveState == "active" {
		sb.WriteString(renderDetailRow("Active Since", formatTimestamp(m.activeDetail.ActiveEnterTimestamp)) + "\n")
	} else if m.activeDetail.ActiveState == "failed" || m.activeDetail.ActiveState == "inactive" {
		sb.WriteString(renderDetailRow("Inactive Since", formatTimestamp(m.activeDetail.ActiveExitTimestamp)) + "\n")
	}

	pidStr := "N/A"
	if m.activeDetail.MainPID > 0 {
		pidStr = fmt.Sprintf("%d", m.activeDetail.MainPID)
	}
	sb.WriteString(renderDetailRow("Main PID", pidStr) + "\n")

	// Tasks
	tasksStr := "N/A"
	if m.activeDetail.TasksCurrent > 0 && m.activeDetail.TasksCurrent != 18446744073709551615 {
		maxTasks := "Unlimited"
		if m.activeDetail.TasksMax > 0 && m.activeDetail.TasksMax != 18446744073709551615 {
			maxTasks = fmt.Sprintf("%d", m.activeDetail.TasksMax)
		}
		tasksStr = fmt.Sprintf("%d / %s", m.activeDetail.TasksCurrent, maxTasks)
	}
	sb.WriteString(renderDetailRow("Tasks/Threads", tasksStr) + "\n")

	// Memory
	memStr := formatMemory(m.activeDetail.MemoryCurrent)
	if m.activeDetail.MemoryLimit > 0 && m.activeDetail.MemoryLimit != 18446744073709551615 {
		memStr += " / " + formatMemory(m.activeDetail.MemoryLimit)
	} else if m.activeDetail.MemoryCurrent > 0 && m.activeDetail.MemoryCurrent != 18446744073709551615 {
		memStr += " / Unlimited"
	}
	sb.WriteString(renderDetailRow("Memory Current", memStr) + "\n")
	sb.WriteString(renderDetailRow("CPU Usage", formatCPU(m.activeDetail.CPUUsageNSec)) + "\n")

	// Traffic details
	ipTraffic := "N/A"
	if (m.activeDetail.IPTrafficRxBytes > 0 && m.activeDetail.IPTrafficRxBytes != 18446744073709551615) ||
		(m.activeDetail.IPTrafficTxBytes > 0 && m.activeDetail.IPTrafficTxBytes != 18446744073709551615) {
		ipTraffic = fmt.Sprintf("Rx: %s, Tx: %s", formatMemory(m.activeDetail.IPTrafficRxBytes), formatMemory(m.activeDetail.IPTrafficTxBytes))
	}
	sb.WriteString(renderDetailRow("IP Traffic", ipTraffic) + "\n")

	ioTraffic := "N/A"
	if (m.activeDetail.IOReadBytes > 0 && m.activeDetail.IOReadBytes != 18446744073709551615) ||
		(m.activeDetail.IOWriteBytes > 0 && m.activeDetail.IOWriteBytes != 18446744073709551615) {
		ioTraffic = fmt.Sprintf("Read: %s, Write: %s", formatMemory(m.activeDetail.IOReadBytes), formatMemory(m.activeDetail.IOWriteBytes))
	}
	sb.WriteString(renderDetailRow("I/O Traffic", ioTraffic) + "\n")

	// Exit Code
	if m.activeDetail.ActiveState == "failed" || m.activeDetail.ActiveState == "inactive" {
		if m.activeDetail.ExecMainStatus != 0 {
			sb.WriteString(renderDetailRow("Exit Status", fmt.Sprintf("status=%d code=%d", m.activeDetail.ExecMainStatus, m.activeDetail.ExecMainCode)) + "\n")
		}
	}

	// Apply box styling based on focus
	style := BoxStyle
	if m.detailFocusedSection == focusInfo {
		style = FocusBoxStyle
	}

	boxContent := style.Width(width - 2).Height(height).Render(sb.String())
	return boxContent
}

func (m Model) renderLogPanel(width, height int) string {
	var sb strings.Builder

	// Header inside the panel
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [ Service Logs (journalctl) ] ") + "\n")

	// We render the viewport inside the box
	sb.WriteString(m.logViewport.View())

	// Apply box styling based on focus
	style := BoxStyle
	if m.detailFocusedSection == focusLogs {
		style = FocusBoxStyle
	}

	boxContent := style.Width(width - 2).Height(height).Render(sb.String())
	return boxContent
}

func (m Model) renderDetailFooter() string {
	var f strings.Builder

	keys := []string{
		"Tab", "Toggle Panel Focus",
		"Esc/Left", "Back to List",
		"s", "Start",
		"t", "Stop",
		"r", "Restart",
		"e", "Enable",
		"d", "Disable",
		"R", "Refresh Logs",
	}

	// Add scroll info if logs focused
	if m.detailFocusedSection == focusLogs {
		keys = append(keys, "↑/↓/PgUp/PgDn", "Scroll Logs")
	}

	items := []string{}
	for i := 0; i < len(keys); i += 2 {
		items = append(items, fmt.Sprintf("%s %s", HelpKeyStyle.Render(keys[i]), HelpDescStyle.Render(keys[i+1])))
	}

	f.WriteString(FooterStyle.Render(strings.Join(items, "  •  ")))
	return f.String()
}

func formatMemory(bytes uint64) string {
	if bytes == 18446744073709551615 || bytes == 0 {
		return "N/A"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatCPU(nsec uint64) string {
	if nsec == 18446744073709551615 || nsec == 0 {
		return "N/A"
	}
	dur := time.Duration(nsec) * time.Nanosecond
	return fmt.Sprintf("%.3fs", dur.Seconds())
}

func formatTimestamp(usec uint64) string {
	if usec == 0 {
		return "N/A"
	}
	t := time.UnixMicro(int64(usec))
	dur := time.Since(t)

	var durStr string
	if dur < time.Minute {
		durStr = fmt.Sprintf("%ds ago", int(dur.Seconds()))
	} else if dur < time.Hour {
		durStr = fmt.Sprintf("%dm %ds ago", int(dur.Minutes()), int(dur.Seconds())%60)
	} else if dur < 24*time.Hour {
		durStr = fmt.Sprintf("%dh %dm ago", int(dur.Hours()), int(dur.Minutes())%60)
	} else {
		durStr = fmt.Sprintf("%dd %dh ago", int(dur.Hours()/24), int(dur.Hours())%24)
	}

	return fmt.Sprintf("%s (%s)", t.Format("2006-01-02 15:04:05"), durStr)
}
