package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Color palette — inspired by lazygit/k9s
const (
	colorPrimary  = lipgloss.Color("#50d050") // green
	colorWarn     = lipgloss.Color("#e5c07b") // soft yellow
	colorDanger   = lipgloss.Color("#e06c75") // muted red
	colorMuted    = lipgloss.Color("#5c6370") // comment gray
	colorSubtle   = lipgloss.Color("#3e4452") // border
	colorText     = lipgloss.Color("#e8e8e8") // near white
	colorSelFg    = lipgloss.Color("#0a1e1c") // dark teal
	colorSelBg    = lipgloss.Color("#80CBC4") // light cyan/teal bar
	colorHeaderFg = lipgloss.Color("#828997") // dim header
	colorBorder   = lipgloss.Color("#3e4452") // dark border
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			PaddingLeft(1)

	statusUp   = "● up"
	statusNone = "○ none"
	statusDown = "✗ down"
	statusErr  = "✗ error"

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			PaddingLeft(1).
			PaddingTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			PaddingLeft(1)

	confirmStyle = lipgloss.NewStyle().
			Foreground(colorWarn).
			Bold(true).
			PaddingLeft(1)

	attachedStyle  = "attached"
	detachedStyle  = "detached"

	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			Width(50)
)

func columnsWidth(cols []table.Column) int {
	w := 0
	for _, c := range cols {
		w += c.Width + 2
	}
	return w
}

func tableStyles(totalWidth int) table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorSubtle).
		BorderBottom(true).
		Foreground(colorHeaderFg).
		Bold(true)
	// No foreground on Cell — we colorize rows in postProcessTable
	s.Cell = lipgloss.NewStyle().Padding(0, 1)
	// Selected: highlight text color only, no background
	s.Selected = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Width(totalWidth)
	return s
}

var (
	normalRowStyle = lipgloss.NewStyle().Foreground(colorText)
	greenStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#50d050"))
	redStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#e06c75"))
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#5c6370"))
)

func postProcessTable(view string, cursor, headerLines int) string {
	lines := strings.Split(view, "\n")
	for i := range lines {
		if i < headerLines {
			continue
		}
		row := i - headerLines
		if row == cursor {
			continue
		}
		line := lines[i]
		// Colorize status text in non-selected rows
		line = strings.Replace(line, "● up", greenStyle.Render("● up"), 1)
		line = strings.Replace(line, "○ none", dimStyle.Render("○ none"), 1)
		line = strings.Replace(line, "✗ down", redStyle.Render("✗ down"), 1)
		line = strings.Replace(line, "✗ error", redStyle.Render("✗ error"), 1)
		lines[i] = normalRowStyle.Render(line)
	}
	return strings.Join(lines, "\n")
}
