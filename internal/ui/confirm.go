package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Confirm asks a yes/no question on stdin and reports the answer.
func Confirm(question string) (bool, error) {
	_, _ = lipgloss.Printf("%s %s ", Normal.Render(question), Muted.Render("[y/N]"))

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	return Affirmative(line), nil
}

// Affirmative reports whether an answer to a yes/no question means yes. Only
// "y" and "yes" do, so a bare enter keeps whatever the safe answer is.
func Affirmative(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// IsTerminal reports whether stdin is a terminal, i.e. whether there is
// somebody present to answer a question. A redirect from /dev/null does not
// count, even though it is a character device like a terminal is.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
