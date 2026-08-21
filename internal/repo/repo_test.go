package repo

import "testing"

func TestParseRemote(t *testing.T) {
	cases := []struct {
		input string
		want  Remote
	}{
		{"git@github.com:owner/repo.git", Remote{Host: "github.com", Owner: "owner", Name: "repo"}},
		{"https://github.com/owner/repo.git", Remote{Host: "github.com", Owner: "owner", Name: "repo"}},
		{"http://git.example.com/team/project", Remote{Host: "git.example.com", Owner: "team", Name: "project"}},
	}
	for _, test := range cases {
		got, err := ParseRemote(test.input)
		if err != nil {
			t.Errorf("ParseRemote(%q): %v", test.input, err)
			continue
		}
		if got != test.want {
			t.Errorf("ParseRemote(%q) = %#v, want %#v", test.input, got, test.want)
		}
	}
}

func TestParseRemoteRejectsUnsupportedURL(t *testing.T) {
	if _, err := ParseRemote("git@github.com:owner/group/repo.git"); err == nil {
		t.Fatal("expected unsupported remote error")
	}
}

func TestSSHURL(t *testing.T) {
	remote := Remote{Owner: "owner", Name: "repo"}
	if got := remote.SSHURL("github-work"); got != "git@github-work:owner/repo.git" {
		t.Fatalf("got %q", got)
	}
}
