package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/ivaneidel/mr-plow/internal/catalog"
)

// Colours adapt to light and dark terminals.
var (
	cAccent   = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	cDim      = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	cFaint    = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
	cText     = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F3F4F6"}
	cSafe     = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	cRegen    = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"}
	cReview   = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	cPersonal = lipgloss.AdaptiveColor{Light: "#BE185D", Dark: "#F472B6"}
	cDanger   = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	cCursorBg = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2E2A5A"}

	segColors = []lipgloss.AdaptiveColor{
		{Light: "#7C3AED", Dark: "#A78BFA"},
		{Light: "#0891B2", Dark: "#22D3EE"},
		{Light: "#D97706", Dark: "#FBBF24"},
		{Light: "#DB2777", Dark: "#F472B6"},
		{Light: "#16A34A", Dark: "#4ADE80"},
		{Light: "#2563EB", Dark: "#60A5FA"},
	}
)

var (
	sTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1E1B4B"}).Background(cAccent).Padding(0, 1)
	sBold     = lipgloss.NewStyle().Bold(true).Foreground(cText)
	sDim      = lipgloss.NewStyle().Foreground(cDim)
	sAccent   = lipgloss.NewStyle().Foreground(cAccent)
	sDanger   = lipgloss.NewStyle().Foreground(cDanger)
	sWarn     = lipgloss.NewStyle().Foreground(cReview)
	sOK       = lipgloss.NewStyle().Foreground(cSafe)
	sCursor   = lipgloss.NewStyle().Background(cCursorBg)
	sTabOn    = lipgloss.NewStyle().Bold(true).Foreground(cAccent).Underline(true)
	sTabOff   = lipgloss.NewStyle().Foreground(cDim)
	sBadgeRed = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(cDanger).Padding(0, 1)
	sBadgeAmb = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(cReview).Padding(0, 1)
	sCmd      = lipgloss.NewStyle().Foreground(cRegen)
)

func safetyColor(s catalog.Safety) lipgloss.AdaptiveColor {
	switch s {
	case catalog.Safe:
		return cSafe
	case catalog.Regenerable:
		return cRegen
	case catalog.Review:
		return cReview
	}
	return cPersonal
}

func safetyLabel(s catalog.Safety) string {
	switch s {
	case catalog.Safe:
		return T("safety.safe")
	case catalog.Regenerable:
		return T("safety.regen")
	case catalog.Review:
		return T("safety.review")
	}
	return T("safety.personal")
}
