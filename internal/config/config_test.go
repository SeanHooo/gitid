package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanho/gitid/internal/profile"
)

func TestSaveLoadAndPermissions(t *testing.T) {
	home := t.TempDir()
	want := Config{Profiles: []profile.Profile{{Name: "work", HostAlias: "github-work"}}}
	if err := Save(home, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Name != "work" {
		t.Fatalf("loaded %#v", got)
	}
	info, err := os.Stat(filepath.Join(home, ".config", "gitid", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestAddRejectsDuplicateAlias(t *testing.T) {
	config := Config{Profiles: []profile.Profile{{Name: "work", HostAlias: "github-work"}}}
	if _, err := config.Add(profile.Profile{Name: "personal", HostAlias: "github-work"}); err == nil {
		t.Fatal("expected duplicate alias error")
	}
}
