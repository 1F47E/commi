package tui

import (
	"commi/internal/utils"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

type Spinner struct {
	program   *tea.Program
	model     spinnerModel
	doneChan  chan struct{}
	startTime time.Time
	isTTY     bool
}

type spinnerModel struct {
	spinner  spinner.Model
	quitting bool
	err      error
	state    string
	duration time.Duration
	text     string
}

func NewSpinner() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	model := spinnerModel{
		spinner: s,
		state:   "idle",
		text:    "Initializing...",
	}

	return &Spinner{
		model:    model,
		doneChan: make(chan struct{}),
		isTTY:    utils.IsTTY(),
	}
}

func (s *Spinner) Start(message string) {
	s.model.state = "running"
	s.model.text = message
	s.startTime = time.Now()

	// If not in TTY, just print the message
	if !s.isTTY {
		fmt.Printf("⏺ %s\n", message)
		return
	}

	s.program = tea.NewProgram(s.model)
	go func() {
		if _, err := s.program.Run(); err != nil {
			log.Error().Err(err).Msg("Error running spinner")
		}
		close(s.doneChan)
	}()
}

func (s *Spinner) Stop() {
	if !s.isTTY {
		return
	}
	s.program.Send(doneMsg{duration: time.Since(s.startTime)})
	<-s.doneChan
}

func (s *Spinner) UpdateText(text string) {
	if !s.isTTY {
		fmt.Printf("⏺ %s\n", text)
		return
	}
	s.program.Send(updateTextMsg(text))
}

type doneMsg struct {
	duration time.Duration
}

type updateTextMsg string

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m spinnerModel) View() string {
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
