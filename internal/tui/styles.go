package tui

import "github.com/charmbracelet/lipgloss"

// Colors — Driveline orange primary, with neutral support tones.
var (
	Primary = lipgloss.Color("#FFA300") // Driveline Orange
	Success = lipgloss.Color("#00d478") // Green for connected pill
	Error   = lipgloss.Color("#ff4757") // Red
	Muted   = lipgloss.Color("#6c757d") // Gray
	Fg      = lipgloss.Color("#e0e0e0") // Near-white body text
	OnAccent = lipgloss.Color("#000000") // Text color when on accent background
)

// Layout constants
const (
	SidebarWidth = 30
)

var (
	// Header
	logoStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	brandSubStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	headerLabelStyle = lipgloss.NewStyle().
				Foreground(Fg).
				Bold(true)

	headerTextStyle = lipgloss.NewStyle().
			Foreground(Fg)

	headerMutedStyle = lipgloss.NewStyle().
				Foreground(Muted)

	// Pills
	pillConnected = lipgloss.NewStyle().
			Background(Success).
			Foreground(OnAccent).
			Padding(0, 1).
			Bold(true)

	pillDisconnected = lipgloss.NewStyle().
				Background(Error).
				Foreground(OnAccent).
				Padding(0, 1).
				Bold(true)

	pillNeutral = lipgloss.NewStyle().
			Background(Muted).
			Foreground(OnAccent).
			Padding(0, 1).
			Bold(true)

	// Tabs
	tabSelectedStyle = lipgloss.NewStyle().
				Background(Primary).
				Foreground(OnAccent).
				Bold(true).
				Width(SidebarWidth)

	tabRowStyle = lipgloss.NewStyle().
			Width(SidebarWidth)

	tabNameStyle = lipgloss.NewStyle().
			Foreground(Fg)

	tabSubStyle = lipgloss.NewStyle().
			Foreground(Muted)

	tabArrowStyle = lipgloss.NewStyle().
			Foreground(Primary)

	// Content
	contentStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(Fg)

	sectionValueStyle = lipgloss.NewStyle().
				Foreground(Fg)

	// Footer
	footerStyle = lipgloss.NewStyle().
			Foreground(Muted)

	helpStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// Status / progress
	connectingStyle = lipgloss.NewStyle().
			Foreground(Primary)

	errorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)
)
