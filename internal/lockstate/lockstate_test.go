package lockstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockedMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dirs, err := Locked()
	if err != nil || dirs != nil {
		t.Fatalf("Locked() = %v, %v", dirs, err)
	}
}

func TestAddDedupesAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Add("/a"); err != nil {
		t.Fatal(err)
	}
	if err := Add("/b"); err != nil {
		t.Fatal(err)
	}
	if err := Add("/a"); err != nil {
		t.Fatal(err)
	}
	dirs, err := Locked()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 2 || dirs[0] != "/a" || dirs[1] != "/b" {
		t.Fatalf("Locked() = %v", dirs)
	}
	info, err := os.Stat(filepath.Join(home, ".anvil", "locks"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("locks file mode = %o, want 600", info.Mode().Perm())
	}
	if err := Remove("/a"); err != nil {
		t.Fatal(err)
	}
	dirs, _ = Locked()
	if len(dirs) != 1 || dirs[0] != "/b" {
		t.Fatalf("after remove, Locked() = %v", dirs)
	}
	if err := Remove("/b"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".anvil", "locks")); !os.IsNotExist(err) {
		t.Error("locks file should be deleted when the list is empty")
	}
}

func TestCanonical(t *testing.T) {
	dir := t.TempDir()
	c, err := Canonical(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(c) {
		t.Errorf("Canonical(%q) = %q, not absolute", dir, c)
	}
}
