package main

import (
	"commi/commit"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Bold(true)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type MenuAction int

const (
	CommitThis MenuAction = iota
	CopyToClipboard
	Regenerate
	Cancel
)

type item struct {
	title  string
	action MenuAction
}

func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := i.title

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	commit   *commit.Commit
	choice   MenuAction
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.action
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("Exiting...")
	}

	commitMessage := renderCommitMessage(m.commit)
	return fmt.Sprintf("%s\n\n%s", commitMessage, m.list.View())
}

func renderCommitMessage(commit *commit.Commit) string {
	return fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit.Commit) {
	items := []list.Item{
		item{title: "✅ Commit this", action: CommitThis},
		item{title: "📋 Copy to clipboard and exit", action: CopyToClipboard},
		item{title: "🔄 Regenerate", action: Regenerate},
		item{title: "❌ Cancel", action: Cancel},
	}

	const defaultWidth = 30

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l, commit: commit}

	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
		os.Exit(1)
	}

	if finalModel, ok := finalModel.(model); ok {
		switch finalModel.choice {
		case CommitThis:
			if err := executeGitAdd(); err != nil {
				log.Error().Err(err).Msg("Failed to execute git add")
				return
			}
			if err := executeGitCommit(commit.Title, commit.Message); err != nil {
				log.Error().Err(err).Msg("Failed to execute git commit")
				return
			}
			log.Info().Msg("Commit successfully created!")
		case CopyToClipboard:
			content := fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
			log.Debug().Msg(fmt.Sprintf("Attempting to copy to clipboard: %s", content))
			if err := copyToClipboard(content); err != nil {
				log.Error().Err(err).Msg("Failed to copy to clipboard")
			} else {
				log.Info().Msg("Commit message copied to clipboard.")
			}
			log.Debug().Msg("Clipboard operation completed")
		case Regenerate:
			runAICommit(cmd, args)
		case Cancel:
			log.Info().Msg("Commit aborted.")
		}
	}
}

func copyToClipboard(content string) error {
	log.Debug().Msg("Entering copyToClipboard function")
	err := clipboard.WriteAll(content)
	if err != nil {
		log.Error().Err(err).Msg("clipboard.WriteAll failed")
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}
	log.Debug().Msg("Successfully wrote to clipboard")
	return nil
}
