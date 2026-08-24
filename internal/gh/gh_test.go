package gh

import "testing"

func TestContainsUser(t *testing.T) {
	output := "github.com\n  ✓ Logged in to github.com account seanho (keyring)\n"
	if !containsUser(output, "seanho") {
		t.Fatal("expected account to be detected")
	}
	if containsUser(output, "other") {
		t.Fatal("unexpected account match")
	}
}
