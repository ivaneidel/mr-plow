package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ivaneidel/mr-plow/internal/i18n"
	"github.com/ivaneidel/mr-plow/internal/plan"
)

// T is a shortcut for i18n.T.
func T(key string) string { return i18n.T(key) }

func human(n int64) string { return plan.Human(n) }

// num formats an integer with thousands separators.
func num(n int64) string {
	s := strconv.FormatInt(n, 10)
	sep := ","
	if i18n.Current() == i18n.ES {
		sep = "."
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(c)
	}
	return b.String()
}

// trunc cuts a (possibly styled) string to w cells with an ellipsis.
func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}

// truncLeft keeps the end of a plain string (paths), prefixing an ellipsis.
func truncLeft(s string, w int) string {
	r := []rune(s)
	if len(r) <= w || w <= 1 {
		return s
	}
	return "…" + string(r[len(r)-w+1:])
}

// pad right-pads a styled string to w cells (truncating if longer).
func pad(s string, w int) string {
	s = trunc(s, w)
	if d := w - lipgloss.Width(s); d > 0 {
		s += strings.Repeat(" ", d)
	}
	return s
}

// padL left-pads a styled string to w cells.
func padL(s string, w int) string {
	if d := w - lipgloss.Width(s); d > 0 {
		s = strings.Repeat(" ", d) + s
	}
	return s
}

// bar draws a fraction 0..1 as a fixed-width bar with eighth-block precision.
func bar(frac float64, w int, c lipgloss.TerminalColor) string {
	if w <= 0 {
		return ""
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	eighths := int(frac*float64(w*8) + 0.5)
	full := eighths / 8
	rem := eighths % 8
	parts := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	s := strings.Repeat("█", full)
	if rem > 0 && full < w {
		s += parts[rem]
		full++
	}
	empty := strings.Repeat(" ", w-full)
	return lipgloss.NewStyle().Foreground(c).Render(s) + lipgloss.NewStyle().Background(cFaint).Render(empty)
}

// gauge is a solid used/free bar.
func gauge(frac float64, w int, c lipgloss.TerminalColor) string {
	n := int(frac*float64(w) + 0.5)
	if n > w {
		n = w
	}
	return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("█", n)) +
		lipgloss.NewStyle().Foreground(cFaint).Render(strings.Repeat("█", w-n))
}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// scroll keeps the cursor visible in a window of h rows.
func scroll(cursor, offset, h, n int) int {
	if h <= 0 {
		return 0
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+h {
		offset = cursor - h + 1
	}
	if offset > n-h {
		offset = n - h
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
