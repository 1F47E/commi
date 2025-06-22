package tui

import (
	"commi/internal/core"
	"commi/internal/git"
	"commi/internal/utils"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	commit   *Commit
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

func renderCommitMessage(commit *Commit) string {
	return fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *Commit, c *core.Core) {
	// Check if we're in a TTY environment
	if !utils.IsTTY() {
		// In non-TTY environment with force flag, apply commit directly
		forceFlag, _ := cmd.Flags().GetBool("force")
		if forceFlag {
			handleForcedCommit(commit)
			return
		}
		// Otherwise, just print the commit message and exit
		fmt.Printf("Generated commit message:\n%s\n\n%s\n", commit.Title, commit.Message)
		fmt.Println("\nRun with -f flag to apply this commit automatically in non-interactive environments.")
		return
	}

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
			if err := git.ExecuteGitAdd(); err != nil {
				log.Error().Err(err).Msg("Failed to execute git add")
				return
			}
			if err := git.ExecuteGitCommit(commit.Title, commit.Message); err != nil {
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
			Run(cmd, args, c)
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

func generateCommitMessage(c *core.Core, status, diffs, subject string) (*Commit, error) {
	spinner := NewSpinner()
	spinner.Start("Generating commit message...")

	sys := core.SystemPrompt
	if _, exists := os.LookupEnv("DISABLE_EMOJI"); !exists {
		sys += "\n• Please follow the gitmoji standard (https://gitmoji.dev/) and feel free to use emojis in the commit messages where appropriate to enhance readability and convey the nature of the changes."
	}

	opts := core.GenerateOptions{
		SystemPrompt: sys,
		Status:       status,
		Diffs:        diffs,
		Subject:      subject,
	}

	if utils.IsDebug() {
		log.Debug().Msgf("Generating commit message with options:")
		log.Debug().Msgf("System prompt: %s", sys)
		log.Debug().Msgf("Status: %d bytes", len(status))
		log.Debug().Msgf("Status: %s", status)
		log.Debug().Msgf("Diffs: %d bytes", len(diffs))
		log.Debug().Msgf("Subject: %s", subject)
	}

	commit, err := c.GenerateCommit(context.Background(), opts)
	spinner.Stop()

	if err != nil {
		if utils.IsDebug() {
			log.Debug().Err(err).Msg("Failed to generate commit message")
		}
		return nil, err
	}

	if utils.IsDebug() {
		log.Debug().Interface("commit", commit).Msg("Generated commit message")
	}

	return &Commit{
		Title:   commit.Title,
		Message: commit.Message,
	}, nil
}

func applyCommit(c *Commit) error {
	// Stage all changes
	stageCmd := exec.Command("git", "add", "-A")
	stageOutput, stageErr := stageCmd.CombinedOutput()
	if stageErr != nil {
		return fmt.Errorf("failed to stage changes: %v\nOutput: %s", stageErr, string(stageOutput))
	}

	// Commit the staged changes
	commitCmd := exec.Command("git", "commit", "-m", c.Title, "-m", c.Message)
	commitOutput, commitErr := commitCmd.CombinedOutput()
	if commitErr != nil {
		return fmt.Errorf("failed to apply commit: %v\nOutput: %s", commitErr, string(commitOutput))
	}
	return nil
}

// ===== AI COMMIT GENERATION

func Run(cmd *cobra.Command, args []string, c *core.Core) {
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Println(cmd.Version)
		return
	}
	status, diffs, err := git.GetGitInfo()
	if err != nil {
		if err.Error() == "nothing to commit" {
			fmt.Println("No changes to commit. Make some changes and try again.")
			return
		}
		log.Error().Err(err).Msg("Failed to get git information")
		os.Exit(1)
	}

	var subject string
	if len(args) > 0 {
		subject = args[0]
	}

	forceFlag, _ := cmd.Flags().GetBool("force")
	prefix, _ := cmd.Flags().GetString("prefix")
	log.Debug().Msgf("Force: %t", forceFlag)
	log.Debug().Msgf("Prefix: %s", prefix)

	commitMessage, err := generateCommitMessage(c, status, diffs, subject)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate commit message")
		os.Exit(1)
	}

	if prefix != "" {
		commitMessage.Title = prefix + " " + commitMessage.Title
	}

	if forceFlag {
		handleForcedCommit(commitMessage)
	} else {
		handleUserResponse(cmd, args, commitMessage, c)
	}
}

func handleForcedCommit(commitMessage *Commit) {
	err := applyCommit(commitMessage)
	if err != nil {
		log.Error().Err(err).Msg("Failed to apply commit")
		os.Exit(1)
	}
	fmt.Printf("Commit applied: %s\n", commitMessage.Title)
	os.Exit(0)
}
