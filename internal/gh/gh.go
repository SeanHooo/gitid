package gh

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/seanhooo/gitid/internal/runner"
)

type Client struct {
	runner runner.Runner
}

func New() Client { return Client{runner: runner.Runner{}} }

func (c Client) CheckAndSwitch(host, username string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed; install it and run gh auth login first")
	}
	if err := c.authenticated(host, username); err != nil {
		return err
	}
	if err := c.runner.RunQuiet("gh", "auth", "switch", "--hostname", host, "--user", username); err != nil {
		return fmt.Errorf("could not switch GitHub account %q on %s; run gh auth login first", username, host)
	}
	return nil
}

func (c Client) Check(host, username string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed")
	}
	return c.authenticated(host, username)
}

func (c Client) authenticated(host, username string) error {
	output, err := c.runner.RunQuietOutput("gh", "auth", "status", "--hostname", host)
	if err != nil {
		return fmt.Errorf("GitHub account %q is not authenticated on %s; run gh auth login --hostname %s", username, host, host)
	}
	if !containsUser(output, username) {
		return fmt.Errorf("GitHub account %q is not authenticated on %s; run gh auth login --hostname %s", username, host, host)
	}
	return nil
}

func containsUser(output, username string) bool {
	for _, line := range splitLines(output) {
		if contains(line, "Logged in to") && contains(line, username) {
			return true
		}
	}
	return false
}

func splitLines(value string) []string {
	var lines []string
	start := 0
	for index, character := range value {
		if character == '\n' {
			lines = append(lines, value[start:index])
			start = index + 1
		}
	}
	return append(lines, value[start:])
}

func contains(value, part string) bool {
	return strings.Contains(value, part)
}
