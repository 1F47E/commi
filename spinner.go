package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

func initializeSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return s
}

func runSpinner(s spinner.Model) (*tea.Program, chan struct{}) {
	p := tea.NewProgram(initialModel(s))
	spinnerDone := make(chan struct{})
	go func() {
		if _, err := p.Run(); err != nil {
			log.Error().Err(err).Msg("Error running spinner")
		}
		close(spinnerDone)
	}()
	return p, spinnerDone
}

func stopSpinner(p *tea.Program, spinnerDone chan struct{}) {
	p.Quit()
	<-spinnerDone
}

type model struct {
	spinner  spinner.Model
	quitting bool
	err      error
	state    string
}

func initialModel(s spinner.Model) model {
	return model{spinner: s, state: "running"}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			m.state = "quitting"
			return m, tea.Quit
		default:
			return m, nil
		}
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	if m.state == "quitting" {
		return "\n"
	}
	return fmt.Sprintf("\n\n   %s Generating commit message...\n\n", m.spinner.View())
}
