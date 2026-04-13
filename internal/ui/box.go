package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderSummaryBox renders a bordered box containing a title, a set of
// key-value fields (order-preserving), and optional next-steps lines.
//
// fields is an ordered slice of [key, value] pairs so the caller controls
// display order.
func RenderSummaryBox(title string, fields [][2]string, nextSteps []string) string {
	var lines []string

	// Blank line for top padding.
	lines = append(lines, "")

	// Title in accent style.
	lines = append(lines, "  "+AccentStyle.Render(title))
	lines = append(lines, "")

	// Determine label width for alignment.
	labelWidth := 0
	for _, f := range fields {
		if len(f[0]) > labelWidth {
			labelWidth = len(f[0])
		}
	}

	// Render each field.
	for _, f := range fields {
		label := padRight(f[0], labelWidth)
		value := f[1]
		// Style URLs.
		if isURL(value) {
			value = URLStyle.Render(value)
		}
		lines = append(lines, "  "+MutedStyle.Render(label)+"  "+value)
	}

	// Next steps section.
	if len(nextSteps) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+HeaderStyle.Render("Next steps:"))
		for _, s := range nextSteps {
			lines = append(lines, "    "+s)
		}
	}

	// Blank line for bottom padding.
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(0, 1)

	return boxStyle.Render(content)
}

// isURL returns true if s looks like a URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "git@")
}
