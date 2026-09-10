package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanhooo/gitid/internal/repo"
)

func TestUseSwitchesRepositoryAndRestoreReturnsOriginalState(t *testing.T) {
	home := t.TempDir()
	repository := t.TempDir()
	key := filepath.Join(home, "id_work")
	if err := os.WriteFile(key, []byte("test key"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repository, "init")
	git(t, repository, "config", "user.name", "Original")
	git(t, repository, "config", "user.email", "original@example.com")
	git(t, repository, "remote", "add", "origin", "https://github.com/owner/repo.git")

	output := &bytes.Buffer{}
	application := App{Home: home, Input: strings.NewReader(""), Output: output, Repo: repo.New()}
	in(repository, func() {
		mustRun(t, application.Run([]string{"account", "add", "work", "--git-name", "Work", "--email", "work@example.com", "--key", key, "--host-alias", "github-work"}))
		mustRun(t, application.Run([]string{"use", "work", "--yes"}))
		if got := git(t, repository, "config", "--local", "user.email"); got != "work@example.com" {
			t.Fatalf("email = %q", got)
		}
		if got := git(t, repository, "config", "--local", "gitid.profile"); got != "work" {
			t.Fatalf("profile = %q", got)
		}
		if got := git(t, repository, "remote", "get-url", "origin"); got != "git@github-work:owner/repo.git" {
			t.Fatalf("origin = %q", got)
		}
		mustRun(t, application.Run([]string{"restore", "--yes"}))
	})
	if got := git(t, repository, "config", "--local", "user.email"); got != "original@example.com" {
		t.Fatalf("restored email = %q", got)
	}
	if got := git(t, repository, "remote", "get-url", "origin"); got != "https://github.com/owner/repo.git" {
		t.Fatalf("restored origin = %q", got)
	}
}

func TestUsageOmitsColorWhenDisabled(t *testing.T) {
	output := &bytes.Buffer{}
	application := App{Output: output, Color: false}
	application.usage()
	text := output.String()
	if strings.Contains(text, "\x1b[") {
		t.Fatalf("usage contains ANSI codes when color is disabled")
	}
	for _, want := range []string{"account add", "--git-name", "--protocol ssh|https", "gitid restore"} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage is missing %q", want)
		}
	}
}

func TestUsageAddsColorWhenEnabled(t *testing.T) {
	output := &bytes.Buffer{}
	application := App{Output: output, Color: true}
	application.usage()
	if text := output.String(); !strings.Contains(text, "\x1b[") {
		t.Fatalf("usage has no ANSI codes when color is enabled")
	}
}

func git(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %s", strings.Join(arguments, " "), output)
	}
	return strings.TrimSpace(string(output))
}

func mustRun(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func in(directory string, function func()) {
	original, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if err := os.Chdir(directory); err != nil {
		panic(err)
	}
	defer os.Chdir(original)
	function()
}
