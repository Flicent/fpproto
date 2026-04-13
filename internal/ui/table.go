package ui

import (
	"strings"
)

// RenderTable renders a styled text table with cyan/bold headers and
// columns padded to align. It calculates column widths from the data.
func RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	cols := len(headers)

	// Determine the maximum width for each column.
	widths := make([]int, cols)
	for i, h := range headers {
		if len(h) > widths[i] {
			widths[i] = len(h)
		}
	}
	for _, row := range rows {
		for i := 0; i < cols && i < len(row); i++ {
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}

	// Add gutter padding between columns.
	const gutter = 3

	var b strings.Builder

	// Render header row.
	for i, h := range headers {
		cell := padRight(h, widths[i])
		styled := HeaderStyle.Render(cell)
		b.WriteString(styled)
		if i < cols-1 {
			b.WriteString(strings.Repeat(" ", gutter))
		}
	}
	b.WriteString("\n")

	// Render separator.
	for i, w := range widths {
		b.WriteString(MutedStyle.Render(strings.Repeat("─", w)))
		if i < cols-1 {
			b.WriteString(strings.Repeat(" ", gutter))
		}
	}
	b.WriteString("\n")

	// Render data rows.
	for _, row := range rows {
		for i := 0; i < cols; i++ {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			cell := padRight(val, widths[i])
			b.WriteString(cell)
			if i < cols-1 {
				b.WriteString(strings.Repeat(" ", gutter))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// padRight pads s with spaces on the right to reach the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
