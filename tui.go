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
	docStyle    = lipgloss.NewStyle().Margin(1, 2)
	titleStyle  = lipgloss.NewStyle().Bold(true)
	listStyle   = lipgloss.NewStyle().Margin(1, 0, 0, 2)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

type commit struct {
	Title   string
	Message string
}

type menuItem string

func (i menuItem) FilterValue() string { return string(i) }

type tuiModel struct {
	viewport viewport.Model
	list     list.Model
	commit   *commit
	choice   string
	quitting bool
	width    int
	height   int
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(menuItem)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		h, v := docStyle.GetFrameSize()
		m.viewport.Width = msg.Width - h
		m.viewport.Height = msg.Height - v - 6 // Reserve space for the list
		m.list.SetWidth(msg.Width - h)
		m.list.SetHeight(5) // Fixed height for the list
		
		if m.commit != nil {
			m.viewport.SetContent(renderCommitMessage(m.commit))
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	return docStyle.Render(fmt.Sprintf(
		"%s\n%s",
		m.viewport.View(),
		listStyle.Render(m.list.View()),
	))
}

func renderCommitMessage(commit *commit) string {
	return fmt.Sprintf("%s\n\n%s", titleStyle.Render(commit.Title), commit.Message)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit) {
	items := []list.Item{
		menuItem("✅ Commit this message"),
		menuItem("🔄 Generate another one"),
		menuItem("📋 Copy to clipboard & exit"),
		menuItem("❌ Cancel"),
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedStyle
	delegate.Styles.SelectedDesc = selectedStyle

	l := list.New(items, delegate, 0, 0)
	l.Title = "Options"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	m := tuiModel{
		list:     l,
		commit:   commit,
		viewport: viewport.New(0, 0),
	}

	m.viewport.SetContent(renderCommitMessage(commit))

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
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
		}
	}
}
func copyToClipboard(content string) error {
	// Implementation of copyToClipboard function
	// This will depend on the clipboard library you choose to use
	// For example, you might use github.com/atotto/clipboard
	return nil // Replace with actual implementation
}
