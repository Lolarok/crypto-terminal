package main

import "github.com/charmbracelet/lipgloss"

// ─── Color Palette ─────────────────────────────────────
var (
	// Core
	bg      = lipgloss.Color("#0a0a12")
	surface = lipgloss.Color("#12121e")
	accent  = lipgloss.Color("#9945FF")
	cyan    = lipgloss.Color("#14F195")
	white   = lipgloss.Color("#e8e8f0")
	dim     = lipgloss.Color("#6a6a8a")
	border  = lipgloss.Color("#2a2a40")

	// Semantic
	green = lipgloss.Color("#22c55e")
	red   = lipgloss.Color("#ef4444")
	yellow = lipgloss.Color("#eab308")
	orange = lipgloss.Color("#f97316")
	pink   = lipgloss.Color("#ec4899")
	blue   = lipgloss.Color("#3b82f6")
)

// ─── Styles ────────────────────────────────────────────
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 2)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 2).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(accent)

	tabInactiveStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 2)

	rowStyle = lipgloss.NewStyle().
		Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
			Foreground(white).
			Padding(0, 1)

	cellDimStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 1)

	priceUpStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	priceDownStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	scoreHighStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	scoreMedStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	scoreLowStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	coinNameStyle = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	coinSymbolStyle = lipgloss.NewStyle().
			Foreground(dim)

	borderBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(1, 2)

	statusStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 2)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(accent)

	sectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			Padding(0, 0, 1, 0)
)

func scoreStyle(score float64) lipgloss.Style {
	if score >= 70 {
		return scoreHighStyle
	}
	if score >= 50 {
		return scoreMedStyle
	}
	return scoreLowStyle
}

func changeStyle(pct float64) lipgloss.Style {
	if pct >= 0 {
		return priceUpStyle
	}
	return priceDownStyle
}
