package ui

import "github.com/charmbracelet/lipgloss"

// Color palette constants.
const (
	Primary lipgloss.Color = "#00D4FF" // Cyan — headers, command names, URLs
	Success lipgloss.Color = "#00FF88" // Green — completion messages, checkmarks
	Warning lipgloss.Color = "#FFD700" // Yellow — prompts, version nudges
	Error   lipgloss.Color = "#FF4444" // Red — failures, destructive confirmations
	Muted   lipgloss.Color = "#888888" // Gray — secondary info, timestamps, hints
	Accent  lipgloss.Color = "#FF44FF" // Magenta — prototype names, emphasis
)

// Styles built from the palette.
var (
	// HeaderStyle is bold cyan for command headers.
	HeaderStyle = lipgloss.NewStyle().Foreground(Primary).Bold(true)

	// SuccessStyle is green for checkmarks and success messages.
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)

	// ErrorStyle is red for errors.
	ErrorStyle = lipgloss.NewStyle().Foreground(Error)

	// WarningStyle is yellow for warnings.
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)

	// MutedStyle is gray for secondary info.
	MutedStyle = lipgloss.NewStyle().Foreground(Muted)

	// AccentStyle is magenta bold for prototype names.
	AccentStyle = lipgloss.NewStyle().Foreground(Accent).Bold(true)

	// URLStyle is cyan underlined for URLs.
	URLStyle = lipgloss.NewStyle().Foreground(Primary).Underline(true)
)

// Status icons rendered in their respective colors.
var (
	SuccessIcon = SuccessStyle.Render("\u2713")
	ErrorIcon   = ErrorStyle.Render("\u2717")
	WarningIcon = WarningStyle.Render("\u26A0")
)
