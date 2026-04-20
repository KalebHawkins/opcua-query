package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	Title       lipgloss.Style
	Muted       lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
	Section     lipgloss.Style
	Card        lipgloss.Style
	Path        lipgloss.Style
	NodeID      lipgloss.Style
	ValueCell   lipgloss.Style
	HeaderCell  lipgloss.Style
	Border      lipgloss.Style
	SpinnerText lipgloss.Style
}

func newStyles() styles {
	borderColor := lipgloss.Color("63")
	return styles{
		Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1),
		Muted:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Label:       lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Bold(true),
		Value:       lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
		Success:     lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		Warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		Section:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).MarginTop(1),
		Card:        lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(borderColor).Padding(0, 1),
		Path:        lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
		NodeID:      lipgloss.NewStyle().Foreground(lipgloss.Color("117")),
		ValueCell:   lipgloss.NewStyle().Foreground(lipgloss.Color("222")),
		HeaderCell:  lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true),
		Border:      lipgloss.NewStyle().Foreground(borderColor),
		SpinnerText: lipgloss.NewStyle().Foreground(lipgloss.Color("80")).Bold(true),
	}
}
