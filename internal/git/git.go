package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

func GetGitInfo() (string, string, error) {
	status, err := GetGitStatus()
	if err != nil {
		if err.Error() == "nothing to commit" {
			return "", "", fmt.Errorf("nothing to commit")
		}
		return "", "", fmt.Errorf("failed to get git status: %w", err)
	}

	files, err := GetChangedFiles(status)
	if err != nil {
		return "", "", fmt.Errorf("failed to get changed files: %w", err)
	}

	diffs := ""
	for _, file := range files {
		diff, err := GetGitDiff(file)
		if err != nil {
			log.Warn().Err(err).Str("file", file).Msg("Failed to get diff for file")
			continue
		}
		diffs += fmt.Sprintf("Diff for %s:\n%s\n\n", file, diff)
	}

	return status, diffs, nil
}

func GetGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if len(output) == 0 {
		return "", fmt.Errorf("nothing to commit")
	}
	return string(output), nil
}

func GetChangedFiles(status string) ([]string, error) {
	lines := strings.Split(status, "\n")
	var files []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected git status output format")
		}
		files = append(files, parts[1])
	}
	return files, nil
}

func GetGitDiff(file string) (string, error) {
	cmd := exec.Command("git", "--no-pager", "diff", file)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func ExecuteGitAdd() error {
	cmd := exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

func ExecuteGitCommit(title, message string) error {
	cmd := exec.Command("git", "commit", "-m", title, "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}
