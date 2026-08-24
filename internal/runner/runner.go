package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct{}

func (r Runner) Run(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), message)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r Runner) RunQuiet(name string, args ...string) error {
	command := exec.Command(name, args...)
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s command failed", name)
	}
	return nil
}

func (r Runner) RunQuietOutput(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s command failed", name)
	}
	return strings.TrimSpace(string(output)), nil
}
