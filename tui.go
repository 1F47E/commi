package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
	titleStyle = lipgloss.NewStyle().Bold(true)
)

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
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func renderCommitMessage(commit *commit) string {
	return fmt.Sprintf("%s\n\n%s", titleStyle.Render(commit.Title), commit.Message)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit) {
	items := []list.Item{
		item{title: "✅ Commit this message", desc: "Apply the generated commit message"},
		item{title: "🔄 Generate another one", desc: "Create a new commit message"},
		item{title: "📋 Copy to clipboard & exit", desc: "Copy the message and close"},
		item{title: "🔍 Preview Commit Message", desc: "View the generated commit message"},
		item{title: "❌ Cancel", desc: "Abort the commit process"},
	}

	m := model{
		list:   list.New(items, list.NewDefaultDelegate(), 0, 0),
		commit: commit,
	}
	m.list.Title = "Commit Options"

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
		os.Exit(1)
	}

	if finalModel, ok := finalModel.(model); ok {
		choice := finalModel.list.SelectedItem().(item).title
		switch choice {
		case "❌ Cancel":
			log.Info().Msg("Commit aborted.")
		case "🔄 Generate another one":
			log.Info().Msg("Regenerating commit message...")
			runAICommit(cmd, args)
		case "✅ Commit this message":
			if err := executeGitAdd(); err != nil {
				log.Error().Err(err).Msg("Failed to execute git add")
				os.Exit(1)
			}
			if err := executeGitCommit(commit.Title, commit.Message); err != nil {
				log.Error().Err(err).Msg("Failed to execute git commit")
				os.Exit(1)
			}
			log.Info().Msg("Commit successfully created!")
		case "📋 Copy to clipboard & exit":
			content := fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
			if err := copyToClipboard(content); err != nil {
				log.Error().Err(err).Msg("Failed to copy to clipboard")
			} else {
				log.Info().Msg("Commit message copied to clipboard.")
			}
		case "🔍 Preview Commit Message":
			previewCommitMessage(commit)
		}
	}
}

type previewModel struct {
	viewport viewport.Model
}

func (m previewModel) Init() tea.Cmd {
	return nil
}

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m previewModel) View() string {
	return m.viewport.View()
}

func previewCommitMessage(commit *commit) {
	content := renderCommitMessage(commit)
	vp := viewport.New(80, 20)
	vp.SetContent(content)

	m := previewModel{viewport: vp}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Error().Err(err).Msg("Error running preview")
	}
}

func copyToClipboard(content string) error {
	// Implementation of copyToClipboard function
	// This will depend on the clipboard library you choose to use
	// For example, you might use github.com/atotto/clipboard
	return nil // Replace with actual implementation
}
