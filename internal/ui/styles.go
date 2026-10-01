package ui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("63")
	accentColor    = lipgloss.Color("212")
	greenColor     = lipgloss.Color("42")
	redColor       = lipgloss.Color("196")
	goldColor      = lipgloss.Color("220")
	grayColor      = lipgloss.Color("240")
	lightGrayColor = lipgloss.Color("250")
	cyanColor      = lipgloss.Color("86")
	bgCardColor    = lipgloss.Color("236")
	whiteColor     = lipgloss.Color("255")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(whiteColor).Background(primaryColor).Padding(0, 2)

	cryptoSectionHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(cyanColor)
	stockSectionHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(goldColor)

	widgetCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(grayColor).
			Padding(0, 1).
			MarginRight(2)

	selectedWidgetCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Background(bgCardColor).
				Padding(0, 1).
				MarginRight(2).
				Bold(true)

	widgetSymbolStyle = lipgloss.NewStyle().Bold(true).Foreground(whiteColor)
	widgetNameStyle   = lipgloss.NewStyle().Foreground(lightGrayColor)
	widgetCapStyle    = lipgloss.NewStyle().Foreground(cyanColor).Bold(true)
	widgetPriceStyle  = lipgloss.NewStyle().Bold(true).Foreground(whiteColor)
	mutedStyle        = lipgloss.NewStyle().Foreground(grayColor)

	positiveStyle = lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	negativeStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	neutralStyle  = lipgloss.NewStyle().Foreground(lightGrayColor)
	warnStyle     = lipgloss.NewStyle().Foreground(goldColor)
	errorStyle    = lipgloss.NewStyle().Foreground(redColor).Bold(true)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 2)

	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().Foreground(grayColor)
)
