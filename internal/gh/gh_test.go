package gh

import "testing"

func TestContainsUser(t *testing.T) {
	output := "github.com\n  ✓ Logged in to github.com account seanhooo (keyring)\n"
	if !containsUser(output, "seanhooo") {
		t.Fatal("expected account to be detected")
	}
	if containsUser(output, "other") {
		t.Fatal("unexpected account match")
	}
}
