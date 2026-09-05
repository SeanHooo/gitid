package version

import "testing"

func TestString(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = originalVersion, originalCommit, originalDate })
	Version = "v1.2.3"
	Commit = "abc123"
	Date = "2026-08-24T00:00:00Z"
	want := "v1.2.3 (commit abc123, built 2026-08-24T00:00:00Z)"
	if got := String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
