package buildinfo

import "testing"

func TestCurrentUsesInjectedIdentity(t *testing.T) {
	oldVersion, oldCommit, oldBuiltAt := Version, Commit, BuiltAt
	Version, Commit, BuiltAt = "2.0.0", "abc123", "2026-10-05T17:00:00Z"
	t.Cleanup(func() { Version, Commit, BuiltAt = oldVersion, oldCommit, oldBuiltAt })

	got := Current()
	if got.Service != "dinapay-routing" || got.Commit != "abc123" || got.Version != "2.0.0" {
		t.Fatalf("unexpected build identity: %+v", got)
	}
}
