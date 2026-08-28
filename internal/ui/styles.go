package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	primaryColor   = lipgloss.Color("63")  // Royal Blue
	accentColor    = lipgloss.Color("212") // Soft Pink / Magenta
	greenColor     = lipgloss.Color("42")  // Mint / Green
	redColor       = lipgloss.Color("196") // Bright Red
	grayColor      = lipgloss.Color("240") // Dark Gray
	lightGrayColor = lipgloss.Color("250") // Light Gray
	cyanColor      = lipgloss.Color("86")  // Cyan / Teal

	// Title Banner
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	// Subtitle / Section Headers
	subHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor).
			MarginBottom(1)

	// Summary Cards Header Top
	summaryCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(grayColor).
				Padding(0, 1).
				MarginRight(1)

	summaryValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	// Container Boxes
	tableBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	panelBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1).
			Width(44)

	chartBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor).
			Padding(0, 1).
			MarginTop(1)

	// Table Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(grayColor)

	// Table Rows
	rowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("237"))

	// Status Colors
	positiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(greenColor)

	negativeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(redColor)

	neutralStyle = lipgloss.NewStyle().
			Foreground(lightGrayColor)

	// Input Modal
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(50)

	// Status Bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(grayColor).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(redColor).
			Bold(true)
)
