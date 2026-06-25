package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorPrimary   = lipgloss.Color("#6366f1") // Indigo
	ColorSecondary = lipgloss.Color("#06b6d4") // Cyan
	ColorDark      = lipgloss.Color("#1f2937") // Gray 800
	ColorGray      = lipgloss.Color("#9ca3af") // Gray 400
	ColorDim       = lipgloss.Color("#4b5563") // Gray 600
	ColorBg        = lipgloss.Color("#0f172a") // Slate 900
	ColorText      = lipgloss.Color("#f8fafc") // Slate 50

	ColorActive   = lipgloss.Color("#10b981") // Emerald 500
	ColorInactive = lipgloss.Color("#6b7280") // Gray 500
	ColorFailed   = lipgloss.Color("#ef4444") // Red 500
	ColorWarning  = lipgloss.Color("#f59e0b") // Amber 500
	ColorEnabled  = lipgloss.Color("#14b8a6") // Teal 500
	ColorDisabled = lipgloss.Color("#6b7280") // Gray 500
)

// Styles
var (
	DocStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText)

	TitleStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1).
			MarginRight(2)

	SubTitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorDim).
			PaddingBottom(1)

	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	RowStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(ColorPrimary).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				PaddingLeft(1)

	// Badges
	ActiveBadge = lipgloss.NewStyle().
			Foreground(ColorActive).
			Bold(true)

	InactiveBadge = lipgloss.NewStyle().
			Foreground(ColorInactive)

	FailedBadge = lipgloss.NewStyle().
			Foreground(ColorFailed).
			Bold(true)

	WarningBadge = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	EnabledBadge = lipgloss.NewStyle().
			Foreground(ColorEnabled).
			Bold(true)

	DisabledBadge = lipgloss.NewStyle().
			Foreground(ColorDisabled)

	StaticBadge = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// Borders & Containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(0, 1)

	FocusBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// Details & Metadata
	DetailKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			Width(16)

	DetailValStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Footer & Messages
	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(ColorDim).
			PaddingTop(1).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	SuccessBanner = lipgloss.NewStyle().
			Background(ColorActive).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1).
			MarginTop(1)

	ErrorBanner = lipgloss.NewStyle().
			Background(ColorFailed).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1).
			MarginTop(1)

	InfoBanner = lipgloss.NewStyle().
			Background(ColorSecondary).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1).
			MarginTop(1)

	// Searching & Input
	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	SearchInputStyle = lipgloss.NewStyle().
				Foreground(ColorText)
)
