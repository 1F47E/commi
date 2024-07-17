package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const viewportWidth = 80

type commit struct {
	Title   string
	Message string
}

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Bold(true)
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
	viewport viewport.Model
	list     list.Model
	commit   *commit
	choice   string
	quitting bool
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
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
	case tea.WindowSizeMsg:
		headerHeight := 6
		footerHeight := 3
		verticalMarginHeight := headerHeight + footerHeight

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMarginHeight
		m.list.SetSize(msg.Width, listHeight)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	return fmt.Sprintf("%s\n\n%s", m.viewport.View(), m.list.View())
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *commit) {
	items := []list.Item{
		item("✅ Commit this message"),
		item("🔄 Generate another one"),
		item("✏️ Edit"),
		item("❌ Cancel"),
	}

	const defaultWidth = 30

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Do you want to proceed with this commit message?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	content := fmt.Sprintf("# %s\n\n%s", commit.Title, commit.Message)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(viewportWidth),
	)
	renderedContent, _ := renderer.Render(content)

	vp := viewport.New(viewportWidth, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)
	vp.SetContent(renderedContent)

	m := tuiModel{
		viewport: vp,
		list:     l,
		commit:   commit,
	}

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
		case "✏️ Edit":
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
func runEditCommitMessage(commitMsg *commit) (*commit, error) {
	initialContent := fmt.Sprintf("%s\n\n%s", commitMsg.Title, commitMsg.Message)
	
	m := textEditModel{
		textArea: textarea.New(),
		choices:  []string{"✅ Commit this", "❌ Cancel"},
	}
	m.textArea.SetValue(initialContent)
	m.textArea.Focus()
	m.textArea.ShowLineNumbers = false
	m.textArea.Placeholder = "Enter your commit message here..."

	p := tea.NewProgram(m, tea.WithAltScreen())
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
	width    int
	height   int
}

func (m textEditModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m textEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textArea.SetWidth(m.width - 4)
		m.textArea.SetHeight(m.height - 10)
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

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	s.WriteString(titleStyle.Render("Edit your commit message"))
	s.WriteString("\n\n")

	s.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Render(m.textArea.View()))
	s.WriteString("\n\n")

	for i, choice := range m.choices {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#383838")).
			Padding(0, 1)

		if m.cursor == i {
			style = style.Background(lipgloss.Color("#7D56F4"))
		}

		s.WriteString(style.Render(choice))
		s.WriteString(" ")
	}

	return lipgloss.NewStyle().
		Margin(1).
		Render(s.String())
}
