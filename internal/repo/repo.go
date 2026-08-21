package repo

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mac125/gitid/internal/runner"
)

const backupPrefix = "gitid.backup."

type Remote struct {
	Host  string
	Owner string
	Name  string
}

type State struct {
	Root    string
	Profile string
	Name    string
	Email   string
	Origin  string
}

type Repository struct {
	runner runner.Runner
}

func New() Repository {
	return Repository{runner: runner.Runner{}}
}

func (r Repository) State() (State, error) {
	root, err := r.runner.Run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return State{}, fmt.Errorf("current directory is not a Git repository: %w", err)
	}
	profile, _ := r.get("gitid.profile")
	name, _ := r.get("user.name")
	email, _ := r.get("user.email")
	origin, err := r.runner.Run("git", "remote", "get-url", "origin")
	if err != nil {
		return State{}, fmt.Errorf("repository has no origin remote: %w", err)
	}
	return State{Root: root, Profile: profile, Name: name, Email: email, Origin: origin}, nil
}

func ParseRemote(value string) (Remote, error) {
	value = strings.TrimSpace(value)
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(value); matches != nil {
		return Remote{Host: matches[1], Owner: matches[2], Name: strings.TrimSuffix(matches[3], ".git")}, nil
	}
	parsed, err := url.Parse(value)
	if err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return Remote{Host: parsed.Host, Owner: parts[0], Name: strings.TrimSuffix(parts[1], ".git")}, nil
		}
	}
	return Remote{}, fmt.Errorf("unsupported origin URL %q; expected SSH or HTTPS owner/repository URL", value)
}

func (remote Remote) SSHURL(alias string) string {
	return fmt.Sprintf("git@%s:%s/%s.git", alias, remote.Owner, remote.Name)
}

func (r Repository) Backup(state State) error {
	if _, found := r.get(backupPrefix + "origin"); found {
		return nil
	}
	for key, value := range map[string]string{
		backupPrefix + "profile":   state.Profile,
		backupPrefix + "userName":  state.Name,
		backupPrefix + "userEmail": state.Email,
		backupPrefix + "origin":    state.Origin,
	} {
		if _, err := r.runner.Run("git", "config", "--local", key, value); err != nil {
			return err
		}
	}
	return nil
}

func (r Repository) Use(profile, name, email, remoteURL string) error {
	for key, value := range map[string]string{
		"user.name":     name,
		"user.email":    email,
		"gitid.profile": profile,
	} {
		if _, err := r.runner.Run("git", "config", "--local", key, value); err != nil {
			return err
		}
	}
	_, err := r.runner.Run("git", "remote", "set-url", "origin", remoteURL)
	return err
}

func (r Repository) Restore() error {
	keys := map[string]string{
		"gitid.profile": backupPrefix + "profile",
		"user.name":     backupPrefix + "userName",
		"user.email":    backupPrefix + "userEmail",
	}
	origin, found := r.get(backupPrefix + "origin")
	if !found {
		return fmt.Errorf("no gitid backup exists for this repository")
	}
	for target, source := range keys {
		value, exists := r.get(source)
		if exists && value != "" {
			if _, err := r.runner.Run("git", "config", "--local", target, value); err != nil {
				return err
			}
		} else if _, err := r.runner.Run("git", "config", "--local", "--unset-all", target); err != nil {
			return err
		}
	}
	if _, err := r.runner.Run("git", "remote", "set-url", "origin", origin); err != nil {
		return err
	}
	for _, key := range []string{"profile", "userName", "userEmail", "origin"} {
		if _, err := r.runner.Run("git", "config", "--local", "--unset-all", backupPrefix+key); err != nil {
			return err
		}
	}
	return nil
}

func (r Repository) get(key string) (string, bool) {
	value, err := r.runner.Run("git", "config", "--local", "--get", key)
	return value, err == nil
}
