package tui

import (
	"commi/internal/config"
	"commi/internal/git"
	"commi/internal/llm"
	"commi/internal/llm/anthropic"
	"commi/internal/llm/openai"
	"commi/internal/xmlparser"
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
	commit   *xmlparser.Commit
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

func renderCommitMessage(commit *xmlparser.Commit) string {
	return fmt.Sprintf("%s\n\n%s", commit.Title, commit.Message)
}

func handleUserResponse(cmd *cobra.Command, args []string, commit *xmlparser.Commit) {
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
			Run(cmd, args)
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

func generateCommitMessage(client llm.LLMProvider, status, diffs, subject string) (*xmlparser.Commit, error) {
	spinner := NewSpinner()
	spinner.Start("Generating commit message...")

	sys := llm.SystemPrompt
	if _, exists := os.LookupEnv("DISABLE_EMOJI"); !exists {
		sys += "\n• Please follow the gitmoji standard (https://gitmoji.dev/) and feel free to use emojis in the commit messages where appropriate to enhance readability and convey the nature of the changes."
	}

	// log.Debug().Msgf("System prompt: %s", sys)
	// log.Debug().Msgf("Status: %s", status)
	// log.Debug().Msgf("Subject: %s", subject)
	// log.Debug().Msgf("Diffs: %s", diffs)
	// log.Fatal().Msg("test")

	xmlContent, err := client.GenerateCommitMessage(sys, status, diffs, subject)

	spinner.Stop()

	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("XML content: %s", xmlContent)

	return xmlparser.ParseXMLCommit(xmlContent)
}

func applyCommit(c *xmlparser.Commit) error {
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

func Run(cmd *cobra.Command, args []string) {
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Println(cmd.Version)
		return
	}
	status, diffs, err := git.GetGitInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get git information")
		os.Exit(1)
	}

	client, err := getClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize AI client")
	}

	var subject string
	if len(args) > 0 {
		subject = args[0]
	}

	forceFlag, _ := cmd.Flags().GetBool("force")
	prefix, _ := cmd.Flags().GetString("prefix")
	log.Debug().Msgf("Force: %t", forceFlag)
	log.Debug().Msgf("Prefix: %s", prefix)

	commitMessage, err := generateCommitMessage(client, status, diffs, subject)
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
		handleUserResponse(cmd, args, commitMessage)
	}
}

func handleForcedCommit(commitMessage *xmlparser.Commit) {
	err := applyCommit(commitMessage)
	if err != nil {
		log.Error().Err(err).Msg("Failed to apply commit")
		os.Exit(1)
	}
	fmt.Printf("Commit applied: %s\n", commitMessage.Title)
	os.Exit(0)
}

func getClient() (llm.LLMProvider, error) {
	// Check for explicit provider selection
	preferredProvider := os.Getenv("LLM_PROVIDER")
	if preferredProvider != "" {
		providerType := llm.LLMProviderType(preferredProvider)
		switch providerType {
		case llm.LLMProviderTypeAnthropic:
			if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				log.Debug().Msg("Using Anthropic as LLM provider (from LLM_PROVIDER)")
				return anthropic.NewAnthropicClient(config.LLMConfig{APIKey: key}), nil
			}
			return nil, fmt.Errorf("%s selected as provider but ANTHROPIC_API_KEY is not set", llm.LLMProviderTypeAnthropic)

		case llm.LLMProviderTypeOpenAI:
			if key := os.Getenv("OPENAI_API_KEY"); key != "" {
				log.Debug().Msg("Using OpenAI as LLM provider (from LLM_PROVIDER)")
				return openai.NewOpenAIClient(config.LLMConfig{APIKey: key}), nil
			}
			return nil, fmt.Errorf("%s selected as provider but OPENAI_API_KEY is not set", llm.LLMProviderTypeOpenAI)

		default:
			return nil, fmt.Errorf("invalid LLM_PROVIDER value: %q. Must be either %s or %s",
				preferredProvider, llm.LLMProviderTypeAnthropic, llm.LLMProviderTypeOpenAI)
		}
	}

	// If no explicit provider selected, try both in order
	var availableClients []llm.LLMProvider
	var clientNames []string

	// Try to initialize Anthropic client
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		availableClients = append(availableClients, anthropic.NewAnthropicClient(config.LLMConfig{APIKey: key}))
		clientNames = append(clientNames, string(llm.LLMProviderTypeAnthropic))
	}

	// Try to initialize OpenAI client
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		availableClients = append(availableClients, openai.NewOpenAIClient(config.LLMConfig{APIKey: key}))
		clientNames = append(clientNames, string(llm.LLMProviderTypeOpenAI))
	}

	if len(availableClients) == 0 {
		return nil, fmt.Errorf("no LLM providers available. Please set either ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable")
	}

	// Use the first available client (prioritizing Anthropic)
	log.Debug().Msgf("Using %s as LLM provider (default)", clientNames[0])
	return availableClients[0], nil
}
