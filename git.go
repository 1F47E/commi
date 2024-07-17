package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

func getGitInfo() (string, string, error) {
	status, err := getGitStatus()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git status: %w", err)
	}

	files, err := getChangedFiles(status)
	if err != nil {
		return "", "", fmt.Errorf("failed to get changed files: %w", err)
	}

	diffs := ""
	for _, file := range files {
		diff, err := getGitDiff(file)
		if err != nil {
			log.Warn().Err(err).Str("file", file).Msg("Failed to get diff for file")
			continue
		}
		diffs += fmt.Sprintf("Diff for %s:\n%s\n\n", file, diff)
	}

	return status, diffs, nil
}

func getGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if len(output) == 0 {
		return "", fmt.Errorf("nothing to commit")
	}
	if strings.Contains(string(output), "nothing to commit") {
		return "", fmt.Errorf("nothing to commit")
	}
	return string(output), nil
}

func getChangedFiles(status string) ([]string, error) {
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

func getGitDiff(file string) (string, error) {
	cmd := exec.Command("git", "--no-pager", "diff", file)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func executeGitAdd() error {
	cmd := exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %v\nOutput: %s", err, string(output))
	}
	log.Info().Msg("Git add executed successfully")
	return nil
}

func executeGitCommit(title, message string) error {
	cmd := exec.Command("git", "commit", "-m", title, "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %v\nOutput: %s", err, string(output))
	}
	log.Info().Msg("Git commit executed successfully")
	return nil
}
