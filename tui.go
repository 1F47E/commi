package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	docStyle     = lipgloss.NewStyle().Margin(1, 2)
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6347"))
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4682B4")).Width(80)
)

type commit struct {
	Title   string
	Message string
}

type item string

func (i item) FilterValue() string { return string(i) }

type model struct {
	list     list.Model
	commit   *commit
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i := m.list.SelectedItem()
			return m, m.choose(i.(item))
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-10)
		messageStyle = messageStyle.Width(msg.Width - h - 4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	commitMessage := renderCommitMessage(m.commit)
	return docStyle.Render(fmt.Sprintf("%s\n\n%s", commitMessage, m.list.View()))
}

func (m model) choose(choice item) tea.Cmd {
	return func() tea.Msg {
		switch choice {
		case "Cancel":
			log.Info().Msg("Commit aborted.")
			return tea.Quit()
		case "Regenerate":
			log.Info().Msg("Regenerating commit message...")
			return tea.Quit()
		case "Commit this":
			if err := executeGitAdd(); err != nil {
				log.Error().Err(err).Msg("Failed to execute git add")
				return tea.Quit()
			}
			if err := executeGitCommit(m.commit.Title, m.commit.Message); err != nil {
				log.Error().Err(err).Msg("Failed to execute git commit")
				return tea.Quit()
			}
			log.Info().Msg("Commit successfully created!")
			return tea.Quit()
		case "Copy to clipboard and exit":
			content := fmt.Sprintf("%s\n\n%s", m.commit.Title, m.commit.Message)
			if err := copyToClipboard(content); err != nil {
				log.Error().Err(err).Msg("Failed to copy to clipboard")
			} else {
				log.Info().Msg("Commit message copied to clipboard.")
			}
			return tea.Quit()
		}
		return nil
	}
}

func renderCommitMessage(commit *commit) string {
	return fmt.Sprintf("%s\n\n%s", titleStyle.Render(commit.Title), messageStyle.Render(commit.Message))
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit) {
	items := []list.Item{
		item("Commit this"),
		item("Copy to clipboard and exit"),
		item("Regenerate"),
		item("Cancel"),
	}

	m := model{
		list:   list.New(items, list.NewDefaultDelegate(), 0, 0),
		commit: commit,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
		os.Exit(1)
	}

	if finalModel, ok := finalModel.(model); ok && finalModel.list.SelectedItem() != nil {
		choice := finalModel.list.SelectedItem().(item)
		if string(choice) == "Regenerate" {
			runAICommit(cmd, args)
		}
	}
}

func copyToClipboard(content string) error {
	// Implementation of copyToClipboard function
	// This will depend on the clipboard library you choose to use
	// For example, you might use github.com/atotto/clipboard
	return nil // Replace with actual implementation
}
