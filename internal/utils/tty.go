package utils

import (
	"os"

	"golang.org/x/term"
)

// IsTTY checks if the current process is running in an interactive terminal
func IsTTY() bool {
	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}

	// Also check if we can open /dev/tty (required by Bubble Tea)
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	defer tty.Close()

	return true
}