package sshconfig

import (
	"strings"
	"testing"

	"github.com/seanhooo/gitid/internal/profile"
)

func TestRenderPreservesUnmanagedConfiguration(t *testing.T) {
	existing := "Host unrelated\n  HostName example.com\n"
	profiles := []profile.Profile{{Name: "work", KeyPath: "/tmp/work", HostAlias: "github-work", HostName: "github.com"}}

	result, err := Render(existing, profiles)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Host unrelated", startMarker, "Host github-work", "IdentityFile /tmp/work", endMarker} {
		if !strings.Contains(result, want) {
			t.Errorf("rendered config missing %q:\n%s", want, result)
		}
	}
}

func TestRenderRejectsUnmanagedAlias(t *testing.T) {
	_, err := Render("Host github-work\n  HostName github.com\n", []profile.Profile{{HostAlias: "github-work"}})
	if err == nil {
		t.Fatal("expected unmanaged alias conflict")
	}
}

func TestRenderReplacesManagedBlock(t *testing.T) {
	existing := "Host unrelated\n" + startMarker + "\nHost old\n" + endMarker + "\n"
	result, err := Render(existing, []profile.Profile{{HostAlias: "github-new", HostName: "github.com", KeyPath: "/tmp/new"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result, "Host old") || !strings.Contains(result, "Host github-new") {
		t.Fatalf("managed block was not replaced:\n%s", result)
	}
}
