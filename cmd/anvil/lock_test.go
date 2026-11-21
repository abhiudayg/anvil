package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// makeWorkspace builds a temp tree: a file, a subdir with a file, a .git
// directory with a file, and a symlink to a file outside the tree.
func makeWorkspace(t *testing.T) (dir, outside string) {
	t.Helper()
	dir = t.TempDir()
	t.Cleanup(func() { setWritable(dir, true) })
	outside = filepath.Join(t.TempDir(), "outside.txt")
	for p, data := range map[string]string{
		outside:                             "outside",
		filepath.Join(dir, "main.go"):       "package main",
		filepath.Join(dir, "sub", "lib.go"): "package sub",
		filepath.Join(dir, ".git", "HEAD"):  "ref: refs/heads/main",
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	return dir, outside
}

func writable(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode()&0o200 != 0
}

func TestLockMakesTreeReadOnly(t *testing.T) {
	dir, outside := makeWorkspace(t)
	code, out, _ := runCLI("lock", dir)
	if code != 0 || !strings.Contains(out, "locked "+dir) {
		t.Fatalf("exit = %d, out = %q", code, out)
	}
	for _, p := range []string{
		dir,
		filepath.Join(dir, "main.go"),
		filepath.Join(dir, "sub"),
		filepath.Join(dir, "sub", "lib.go"),
	} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o222 != 0 {
			t.Errorf("%s still has write bits: %v", p, info.Mode())
		}
	}
	if !writable(t, filepath.Join(dir, ".git", "HEAD")) {
		t.Error(".git contents must stay writable")
	}
	if !writable(t, outside) {
		t.Error("symlink target outside the tree must stay writable")
	}
}

func TestLockUsageErrors(t *testing.T) {
	if code, _, _ := runCLI("lock"); code != 2 {
		t.Error("no args: want exit 2")
	}
	if code, _, _ := runCLI("lock", filepath.Join(t.TempDir(), "absent")); code != 2 {
		t.Error("missing dir: want exit 2")
	}
}

func TestUnlockPerfectAnswerRestoresWrite(t *testing.T) {
	setupModel(t, []bool{true, true, true})
	keys := writeKeys(t)
	ans := writeAnswer(t, "Bulk strings are length prefixed, so the reader reads exactly that many bytes, then the CRLF.")
	dir, _ := makeWorkspace(t)
	if code, _, _ := runCLI("lock", dir); code != 0 {
		t.Fatal("lock failed")
	}
	code, out, _ := runCLI("unlock", "--keys", keys, dir, "redis", "1", ans)
	if code != 0 {
		t.Fatalf("exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "unlocked "+dir) {
		t.Errorf("out = %q", out)
	}
	for _, p := range []string{dir, filepath.Join(dir, "main.go"), filepath.Join(dir, "sub"), filepath.Join(dir, "sub", "lib.go")} {
		if !writable(t, p) {
			t.Errorf("%s not writable after unlock", p)
		}
	}
}

func TestUnlockNotPerfectStaysLocked(t *testing.T) {
	setupModel(t, []bool{true, false, true})
	keys := writeKeys(t)
	ans := writeAnswer(t, "It is length prefixed.")
	dir, _ := makeWorkspace(t)
	if code, _, _ := runCLI("lock", dir); code != 0 {
		t.Fatal("lock failed")
	}
	code, out, _ := runCLI("unlock", "--keys", keys, dir, "redis", "1", ans)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(out, "still locked") {
		t.Errorf("out = %q", out)
	}
	if writable(t, filepath.Join(dir, "main.go")) {
		t.Error("workspace must stay read-only")
	}
}

func TestUnlockUsageErrors(t *testing.T) {
	dir, _ := makeWorkspace(t)
	if code, _, _ := runCLI("unlock", dir, "redis", "1"); code != 2 {
		t.Error("wrong arg count: want exit 2")
	}
	if code, _, _ := runCLI("unlock", filepath.Join(t.TempDir(), "absent"), "redis", "1", "x"); code != 2 {
		t.Error("missing dir: want exit 2")
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestReviewShowsChangesAndUntracked(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "tracked.txt")
	git(t, dir, "commit", "-m", "init")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCLI("review", dir)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "tracked.txt") || !strings.Contains(out, "new.txt") {
		t.Errorf("output missing changed files:\n%s", out)
	}
}

func TestReviewRejectsNonGitDir(t *testing.T) {
	if code, _, stderr := runCLI("review", t.TempDir()); code != 2 || !strings.Contains(stderr, "not a git work tree") {
		t.Errorf("exit = %d, stderr = %q", code, stderr)
	}
}
