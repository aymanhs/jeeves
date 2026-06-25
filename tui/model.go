package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aymanhs/sys-tui/systemd"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type viewType int

const (
	listView viewType = iota
	detailView
)

type showMode int

const (
	showAll showMode = iota
	showRunning
)

type focusSection int

const (
	focusInfo focusSection = iota
	focusLogs
)

// Messages
type servicesFetchedMsg struct {
	services []systemd.ServiceInfo
	err      error
}

type detailsFetchedMsg struct {
	details *systemd.ServiceInfo
	logs    string
	err     error
}

type actionCompletedMsg struct {
	action      string
	serviceName string
	err         error
}

type statusTimeoutMsg struct {
	id uint32
}

// Model represents the bubbletea application state.
type Model struct {
	client           *systemd.Client
	services         []systemd.ServiceInfo
	filteredServices []systemd.ServiceInfo
	selectedIndex    int
	scrollOffset     int

	// Search/Filter
	searchInput textinput.Model
	filtering   bool
	filterQuery string

	// Navigation & Views
	currentView           viewType
	showMode              showMode
	detailFocusedSection  focusSection
	activeDetail          *systemd.ServiceInfo
	logs                  string
	logViewport           viewport.Model
	logWrap               bool

	// Status messages
	statusMsg   string
	statusIsErr bool
	statusMsgID uint32

	// Window dimensions
	width  int
	height int

	// Loading state
	loading bool
	err     error
}

// NewModel initializes the Bubble Tea model with the systemd client.
func NewModel(client *systemd.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter services..."
	ti.Prompt = " 🔍 "
	ti.PromptStyle = SearchPromptStyle
	ti.TextStyle = SearchInputStyle
	ti.CharLimit = 50

	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.DefaultKeyMap()

	return Model{
		client:               client,
		searchInput:          ti,
		currentView:          listView,
		showMode:             showAll,
		detailFocusedSection: focusInfo,
		logViewport:          vp,
		logWrap:              true,
	}
}

// Init triggers initial loading commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.fetchServicesCmd(),
	)
}

// Commands
func (m *Model) fetchServicesCmd() tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		services, err := m.client.ListServices(ctx)
		return servicesFetchedMsg{services: services, err: err}
	}
}

func (m *Model) fetchDetailsCmd(name string) tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		details, err := m.client.GetServiceDetails(ctx, name)
		if err != nil {
			return detailsFetchedMsg{err: err}
		}

		// Fetch 150 lines of logs
		logs, err := m.client.GetLogs(ctx, name, 150)
		if err != nil {
			// If logs fails, we still return the details but with empty logs or an error notice
			return detailsFetchedMsg{
				details: details,
				logs:    fmt.Sprintf("Failed to fetch logs: %v", err),
			}
		}

		return detailsFetchedMsg{details: details, logs: logs}
	}
}

func (m *Model) runActionCmd(action, name string) tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		switch action {
		case "start":
			err = m.client.StartService(ctx, name)
		case "stop":
			err = m.client.StopService(ctx, name)
		case "restart":
			err = m.client.RestartService(ctx, name)
		case "enable":
			err = m.client.EnableService(ctx, name)
		case "disable":
			err = m.client.DisableService(ctx, name)
		}

		return actionCompletedMsg{action: action, serviceName: name, err: err}
	}
}

func (m *Model) triggerStatus(msg string, isErr bool) tea.Cmd {
	m.statusMsg = msg
	m.statusIsErr = isErr
	m.statusMsgID++
	id := m.statusMsgID
	return func() tea.Msg {
		time.Sleep(4 * time.Second)
		return statusTimeoutMsg{id: id}
	}
}

// Update handles UI events and messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateViewportSize()

	case servicesFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			cmds = append(cmds, m.triggerStatus(fmt.Sprintf("Fetch failed: %v", msg.err), true))
		} else {
			m.services = msg.services
			m.filterServices()
		}

	case detailsFetchedMsg:
		m.loading = false
		if msg.err != nil {
			cmds = append(cmds, m.triggerStatus(fmt.Sprintf("Failed to get details: %v", msg.err), true))
		} else {
			m.activeDetail = msg.details
			m.logs = msg.logs
			m.logViewport.SetContent(m.logs)
			m.logViewport.GotoBottom()
		}

	case actionCompletedMsg:
		m.loading = false
		if msg.err != nil {
			// Extract a cleaner error message if it's D-Bus permission denied
			errStr := msg.err.Error()
			if strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "InteractiveAuthenticationRequired") {
				errStr = "permission denied (try running with sudo)"
			}
			cmds = append(cmds, m.triggerStatus(fmt.Sprintf("Error running %s on %s: %s", msg.action, msg.serviceName, errStr), true))
		} else {
			actionPast := msg.action + "ed"
			if strings.HasSuffix(msg.action, "e") {
				actionPast = msg.action + "d"
			}
			cmds = append(cmds, m.triggerStatus(fmt.Sprintf("Successfully %s %s", actionPast, msg.serviceName), false))
			// Refresh current state
			if m.currentView == detailView && m.activeDetail != nil && m.activeDetail.Name == msg.serviceName {
				cmds = append(cmds, m.fetchDetailsCmd(msg.serviceName))
			} else {
				cmds = append(cmds, m.fetchServicesCmd())
			}
		}

	case statusTimeoutMsg:
		if msg.id == m.statusMsgID {
			m.statusMsg = ""
		}

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		if m.filtering {
			// Search Input handling
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
				m.searchInput.Blur()
				m.filterServices()
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filterQuery = m.searchInput.Value()
				m.filterServices()
			}
			return m, tea.Batch(cmds...)
		}

		// View-specific keys
		if m.currentView == listView {
			cmds = append(cmds, m.handleListKey(msg))
		} else {
			cmds = append(cmds, m.handleDetailKey(msg))
		}
	}

	// Adjust scroll offset for ListView
	maxRows := m.height - 12
	if maxRows < 3 {
		maxRows = 3
	}
	m.adjustScrollOffset(maxRows)

	return m, tea.Batch(cmds...)
}

func (m *Model) recalculateViewportSize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// Dynamic size calculation for detail view panels
	mainContentHeight := m.height - 5
	if mainContentHeight < 6 {
		mainContentHeight = 6
	}

	if m.width >= 100 {
		// Side-by-side
		m.logViewport.Width = m.width - 45 - 6
		m.logViewport.Height = mainContentHeight - 4 // 2 borders, 2 headers inside box
	} else {
		// Stacked
		infoBoxHeight := mainContentHeight / 2
		logBoxHeight := mainContentHeight - infoBoxHeight
		m.logViewport.Width = m.width - 6
		m.logViewport.Height = logBoxHeight - 4 // 2 borders, 2 headers inside box
	}
}

func (m *Model) filterServices() {
	if m.filterQuery == "" {
		m.filteredServices = m.services
	} else {
		m.filteredServices = nil
		q := strings.ToLower(m.filterQuery)
		for _, s := range m.services {
			if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.Description), q) {
				m.filteredServices = append(m.filteredServices, s)
			}
		}
	}

	// Apply view filter (all/running)
	var final []systemd.ServiceInfo
	for _, s := range m.filteredServices {
		switch m.showMode {
		case showRunning:
			if s.ActiveState == "active" && s.SubState == "running" {
				final = append(final, s)
			}
		default:
			final = append(final, s)
		}
	}
	m.filteredServices = final

	// Bounds checking for selectedIndex
	if m.selectedIndex >= len(m.filteredServices) {
		m.selectedIndex = len(m.filteredServices) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}

func (m *Model) adjustScrollOffset(maxRows int) {
	if len(m.filteredServices) == 0 {
		m.scrollOffset = 0
		return
	}

	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	// Scroll up if index is above viewport
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}

	// Scroll down if index is below viewport
	if m.selectedIndex >= m.scrollOffset+maxRows {
		m.scrollOffset = m.selectedIndex - maxRows + 1
	}

	// Clamp to maximum possible scroll offset
	if len(m.filteredServices) <= maxRows {
		m.scrollOffset = 0
	} else if m.scrollOffset > len(m.filteredServices)-maxRows {
		m.scrollOffset = len(m.filteredServices) - maxRows
	}
}

// View renders the screen based on the current view state.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Fatal Error: %v\n  Press Ctrl+C to quit.\n", m.err)
	}

	if m.currentView == listView {
		return m.renderListView()
	}
	return m.renderDetailView()
}

