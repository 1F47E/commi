package main

import (
	"fmt"
	"time"

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

func runSpinner(s spinner.Model) (*tea.Program, chan struct{}, time.Time) {
	startTime := time.Now()
	p := tea.NewProgram(initialModel(s))
	spinnerDone := make(chan struct{})
	go func() {
		if _, err := p.Run(); err != nil {
			log.Error().Err(err).Msg("Error running spinner")
		}
		close(spinnerDone)
	}()
	return p, spinnerDone, startTime
}

func stopSpinner(p *tea.Program, spinnerDone chan struct{}, startTime time.Time) {
	p.Send(doneMsg{duration: time.Since(startTime)})
	<-spinnerDone
}

type doneMsg struct {
	duration time.Duration
}

type model struct {
	spinner  spinner.Model
	quitting bool
	err      error
	state    string
	duration time.Duration
	text     string
}

func initialModel(s spinner.Model) model {
	return model{spinner: s, state: "running", text: "Generating commit message..."}
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
	case doneMsg:
		m.state = "done"
		m.duration = msg.duration
		return m, tea.Quit
	case updateTextMsg:
		m.text = string(msg)
		return m, nil
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
	switch m.state {
	case "quitting":
		return "\n"
	case "done":
		return fmt.Sprintf("\n\n   Done! Took %.2f seconds\n\n", m.duration.Seconds())
	default:
		return fmt.Sprintf("\n\n   %s %s\n\n", m.spinner.View(), m.text)
	}
}

func updateSpinnerText(p *tea.Program, text string) {
	p.Send(updateTextMsg(text))
}

type updateTextMsg string
