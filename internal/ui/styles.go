package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	primaryColor   = lipgloss.Color("63")  // Royal Blue
	accentColor    = lipgloss.Color("212") // Soft Pink / Magenta
	greenColor     = lipgloss.Color("42")  // Mint / Green
	redColor       = lipgloss.Color("196") // Bright Red
	goldColor      = lipgloss.Color("220") // Gold / Amber
	grayColor      = lipgloss.Color("240") // Dark Gray
	lightGrayColor = lipgloss.Color("250") // Light Gray
	cyanColor      = lipgloss.Color("86")  // Cyan / Teal
	bgCardColor    = lipgloss.Color("236") // Dark card background

	// Title Banner
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	// Section Headers
	cryptoSectionHeaderStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(cyanColor).
					MarginTop(1).
					MarginBottom(1)

	stockSectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(goldColor).
				MarginTop(1).
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

	// macOS Widget Card Styles
	widgetCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(grayColor).
			Padding(0, 1).
			MarginRight(2).
			MarginBottom(1)

	selectedWidgetCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Background(bgCardColor).
				Padding(0, 1).
				MarginRight(2).
				MarginBottom(1).
				Bold(true)

	// Container & Row Styles
	chartBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor).
			Padding(0, 1).
			MarginTop(1)

	rowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Widget Typography
	widgetSymbolStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	widgetNameStyle = lipgloss.NewStyle().
			Foreground(lightGrayColor)

	widgetCapStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Bold(true)

	widgetPriceStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

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
