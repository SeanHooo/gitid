package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type Profile struct {
	Name      string `json:"name"`
	GitName   string `json:"gitName"`
	Email     string `json:"email"`
	KeyPath   string `json:"keyPath"`
	HostAlias string `json:"hostAlias"`
	HostName  string `json:"hostName"`
}

func (p Profile) Validate() error {
	if !namePattern.MatchString(p.Name) {
		return fmt.Errorf("profile name must contain letters, numbers, hyphens, or underscores")
	}
	if strings.TrimSpace(p.GitName) == "" {
		return fmt.Errorf("git name is required")
	}
	if !strings.Contains(p.Email, "@") {
		return fmt.Errorf("a valid email is required")
	}
	if strings.TrimSpace(p.KeyPath) == "" {
		return fmt.Errorf("SSH key path is required")
	}
	if strings.TrimSpace(p.HostAlias) == "" || !namePattern.MatchString(p.HostAlias) {
		return fmt.Errorf("SSH host alias must contain letters, numbers, hyphens, or underscores")
	}
	if strings.TrimSpace(p.HostName) == "" {
		return fmt.Errorf("SSH hostname is required")
	}
	info, err := os.Stat(p.KeyPath)
	if err != nil {
		return fmt.Errorf("SSH key %q: %w", p.KeyPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("SSH key %q is a directory", p.KeyPath)
	}
	return nil
}

func (p Profile) ExpandedKeyPath() string {
	if p.KeyPath == "~" || strings.HasPrefix(p.KeyPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(p.KeyPath, "~/"))
		}
	}
	return p.KeyPath
}
