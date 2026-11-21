package run

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state, ok, err := Load()
	if err != nil || ok {
		t.Fatalf("Load() = %v, %v, %v", state, ok, err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := State{
		Keys: "/keys",
		Workspaces: map[string]Stage{
			"/ws": {Project: "redis", Number: 1, Dir: "s01", Status: "locked", BuildCommit: "abc"},
		},
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	path, err := File()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("run.json mode = %o, want 600", info.Mode().Perm())
	}
	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("Load() = _, %v, %v", ok, err)
	}
	if got.Keys != want.Keys || got.Workspaces["/ws"] != want.Workspaces["/ws"] {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestDelete(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := Delete(); err != nil {
		t.Fatal("deleting a missing run should not error")
	}
	if err := Save(State{Keys: "/keys"}); err != nil {
		t.Fatal(err)
	}
	if err := Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".anvil", "run.json")); !os.IsNotExist(err) {
		t.Error("run.json should be gone")
	}
}
