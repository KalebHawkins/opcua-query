package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerDoneMsg struct{}

type spinnerModel struct {
	spinner  spinner.Model
	label    string
	styles   styles
	quitting bool
}

func newSpinnerModel(label string) spinnerModel {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	return spinnerModel{
		spinner: sp,
		label:   label,
		styles:  newStyles(),
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case spinnerDoneMsg:
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s\n", m.styles.SpinnerText.Render(m.spinner.View()), m.styles.SpinnerText.Render(m.label))
}

type taskResult[T any] struct {
	value T
	err   error
}

func RunTask[T any](label string, fn func(ctx context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	program := tea.NewProgram(newSpinnerModel(label))
	results := make(chan taskResult[T], 1)

	go func() {
		value, err := fn(ctx)
		results <- taskResult[T]{value: value, err: err}
		program.Send(spinnerDoneMsg{})
	}()

	_, runErr := program.Run()
	result := <-results
	if runErr != nil {
		var zero T
		return zero, runErr
	}
	return result.value, result.err
}
