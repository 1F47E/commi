package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)
	appStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)

const minViewportHeight = 10

type commit struct {
	Title   string
	Message string
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type tuiModel struct {
	viewport    viewport.Model
	list        list.Model
	commit      *commit
	choice      string
	quitting    bool
	windowWidth int
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.title
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height/2-v)
		m.viewport.Width = msg.Width - h
		m.viewport.Height = msg.Height/2 - v
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	return appStyle.Render(fmt.Sprintf("%s\n\n%s", m.viewport.View(), m.list.View()))
}

func renderCommitMessage(commit *commit, width int) (string, error) {
	content := fmt.Sprintf("# %s\n\n%s", commit.Title, commit.Message)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(content)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit) {
	items := []list.Item{
		item{title: "✅ Commit this message", desc: "Proceed with the generated commit message"},
		item{title: "🔄 Generate another one", desc: "Request a new commit message"},
		item{title: "📋 Copy to clipboard & exit", desc: "Copy the message and exit"},
		item{title: "❌ Cancel", desc: "Abort the commit process"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Do you want to proceed with this commit message?"

	m := tuiModel{
		list:   l,
		commit: commit,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		for {
			renderedContent, err := renderCommitMessage(commit, m.windowWidth)
			if err != nil {
				log.Error().Msg(fmt.Sprintf("Error rendering commit message: %v", err))
				return
			}
			p.Send(setContentMsg(renderedContent))
			time.Sleep(time.Millisecond * 100)
		}
	}()

	finalModel, err := p.Run()
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Error running Bubble Tea program: %v", err))
		os.Exit(1)
	}

	if finalModel, ok := finalModel.(tuiModel); ok {
		switch finalModel.choice {
		case "❌ Cancel":
			log.Info().Msg("Commit aborted.")
		case "🔄 Generate another one":
			log.Info().Msg("Regenerating commit message...")
			runAICommit(cmd, args)
		case "✅ Commit this message":
			if err := executeGitAdd(); err != nil {
				log.Error().Msg(fmt.Sprintf("Failed to execute git add: %v", err))
				os.Exit(1)
			}
			if err := executeGitCommit(commit.Title, commit.Message); err != nil {
				log.Error().Msg(fmt.Sprintf("Failed to execute git commit: %v", err))
				os.Exit(1)
			}
			log.Info().Msg("Commit successfully created!")
		case "📋 Copy to clipboard & exit":
			content := fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
			if err := copyToClipboard(content); err != nil {
				log.Error().Msg(fmt.Sprintf("Failed to copy to clipboard: %v", err))
			} else {
				log.Info().Msg("Commit message copied to clipboard.")
			}
		}
	}
}

type setContentMsg string

func (m tuiModel) updateContent(content string) (tuiModel, tea.Cmd) {
	m.viewport.SetContent(content)
	return m, nil
}
func copyToClipboard(content string) error {
	// Implementation of copyToClipboard function
	// This will depend on the clipboard library you choose to use
	// For example, you might use github.com/atotto/clipboard
	return nil // Replace with actual implementation
}
