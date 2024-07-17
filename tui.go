package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type tuiModel struct {
	list     list.Model
	title    string
	message  string
	choice   string
	quitting bool
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	return fmt.Sprintf(
		"\033[1m%s\033[0m\n\n%s%s",
		m.title,
		m.message,
		m.list.View(),
	)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit) {
	items := []list.Item{
		item("Commit this message"),
		item("No"),
		item("Generate another one"),
		item("Edit"),
	}

	const defaultWidth = 30

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Do you want to proceed with this commit message?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := tuiModel{list: l, title: commit.Title, message: commit.Message}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
		os.Exit(1)
	}

	if finalModel, ok := finalModel.(tuiModel); ok {
		switch finalModel.choice {
		case "No":
			log.Info().Msg("Commit aborted.")
		case "Generate another one":
			log.Info().Msg("Regenerating commit message...")
			runAICommit(cmd, args)
		case "Commit this message":
			if err := executeGitAdd(); err != nil {
				log.Error().Err(err).Msg("Failed to execute git add")
				os.Exit(1)
			}
			if err := executeGitCommit(commit.Title, commit.Message); err != nil {
				log.Error().Err(err).Msg("Failed to execute git commit")
				os.Exit(1)
			}
			log.Info().Msg("Commit successfully created!")
		case "Edit":
			editedCommit, err := runEditCommitMessage(commit)
			if err != nil {
				log.Error().Err(err).Msg("Failed to edit commit message")
				return
			}
			if editedCommit != nil {
				if err := executeGitAdd(); err != nil {
					log.Error().Err(err).Msg("Failed to execute git add")
					os.Exit(1)
				}
				if err := executeGitCommit(editedCommit.Title, editedCommit.Message); err != nil {
					log.Error().Err(err).Msg("Failed to execute git commit")
					os.Exit(1)
				}
				log.Info().Msg("Commit successfully created with edited message!")
			} else {
				log.Info().Msg("Edit cancelled.")
			}
		}
	}
}
func runEditCommitMessage(commit *commit) (*commit, error) {
	initialContent := fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
	
	m := textEditModel{
		textArea: textarea.New(),
		choices:  []string{"Commit this", "Cancel"},
	}
	m.textArea.SetValue(initialContent)
	m.textArea.Focus()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running text edit program: %w", err)
	}

	if finalModel, ok := finalModel.(textEditModel); ok {
		if finalModel.choice == "Commit this" {
			lines := strings.Split(finalModel.textArea.Value(), "\n")
			if len(lines) < 2 {
				return nil, fmt.Errorf("invalid commit message format")
			}
			return &commit{
				Title:   lines[0],
				Message: strings.TrimSpace(strings.Join(lines[1:], "\n")),
			}, nil
		}
	}

	return nil, nil
}

type textEditModel struct {
	textArea textarea.Model
	choices  []string
	cursor   int
	choice   string
}

func (m textEditModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m textEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.textArea.Focused() {
				m.textArea.Blur()
			} else {
				m.choice = m.choices[m.cursor]
				return m, tea.Quit
			}
		case "up", "down":
			if !m.textArea.Focused() {
				if msg.String() == "up" {
					m.cursor--
				} else {
					m.cursor++
				}

				if m.cursor < 0 {
					m.cursor = len(m.choices) - 1
				} else if m.cursor >= len(m.choices) {
					m.cursor = 0
				}
			}
		case "tab":
			if m.textArea.Focused() {
				m.textArea.Blur()
			} else {
				m.textArea.Focus()
			}
		}
	}

	m.textArea, cmd = m.textArea.Update(msg)
	return m, cmd
}

func (m textEditModel) View() string {
	var s strings.Builder

	s.WriteString("Edit your commit message:\n\n")
	s.WriteString(m.textArea.View())
	s.WriteString("\n\n")

	for i, choice := range m.choices {
		if m.cursor == i {
			s.WriteString("> ")
		} else {
			s.WriteString("  ")
		}
		s.WriteString(choice)
		s.WriteString("\n")
	}

	return s.String()
}
