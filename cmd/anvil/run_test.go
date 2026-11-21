package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"anvil/internal/run"
)

// writeFixtureKeys builds a fake keys repo: keys/redis.json plus a minimal
// verify/ tree with one tiny stage for redis.
func writeFixtureKeys(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("keys/redis.json", `{
  "project": "redis",
  "stages": [
    {
      "number": 1,
      "stage": "RESP reader and TCP server",
      "concept": "bufio-scanner",
      "injection": "x",
      "root_cause": "y",
      "rubric": [
        {"name": "Bulk strings are length prefixed."},
        {"name": "The reader reads exactly the announced number of bytes."},
        {"name": "The trailing CRLF follows the payload."}
      ],
      "reference_answer": "Bulk strings are length prefixed. The reader reads exactly the announced number of bytes. The trailing CRLF follows the payload."
    },
    {
      "number": 2,
      "stage": "Key-value store and commands",
      "concept": "maps",
      "injection": "x",
      "root_cause": "y",
      "rubric": [
        {"name": "The store is a map."},
        {"name": "Commands are dispatched by name."},
        {"name": "Replies use RESP types."}
      ],
      "reference_answer": "The store is a map. Commands are dispatched by name. Replies use RESP types."
    }
  ]
}
`)
	write("verify/go.mod", "module anvil/verify\n\ngo 1.27\n")
	write("verify/go.sum", "")
	write("verify/redis/s01/correct.go", "//go:build !injected\n\npackage stage\n\nfunc Value() int { return 1 }\n")
	write("verify/redis/s01/injected.go", "//go:build injected\n\npackage stage\n\nfunc Value() int { return 2 }\n")
	write("verify/redis/s01/bug_test.go", "package stage\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {\n\tif Value() != 1 {\n\t\tt.Fatalf(\"got %d, want 1\", Value())\n\t}\n}\n")
	write("verify/redis/s02/x_correct.go", "//go:build !injected\n\npackage stage\n\nfunc Pair() int { return 1 }\n")
	write("verify/redis/s02/x_injected.go", "//go:build injected\n\npackage stage\n\nfunc Pair() int { return 2 }\n")
	write("verify/redis/s02/bug_test.go", "package stage\n\nimport \"testing\"\n\nfunc TestPair(t *testing.T) {\n\tif Pair() != 1 {\n\t\tt.Fatalf(\"got %d, want 1\", Pair())\n\t}\n}\n")
	write("verify/redis/s02/cmd/tool/main.go", "package main\n\nfunc main() {}\n")
	write("verify/redis/s02/cmd/tool/y_correct.go", "//go:build !injected\n\npackage main\n\nfunc version() int { return 1 }\n")
	write("verify/redis/s02/cmd/tool/y_injected.go", "//go:build injected\n\npackage main\n\nfunc version() int { return 2 }\n")
	return dir
}

func gitCount(t *testing.T, dir string) int {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n); err != nil {
		t.Fatal(err)
	}
	return n
}

func stageStatus(t *testing.T, ws string) run.Stage {
	t.Helper()
	canonical, err := run.Canonical(ws)
	if err != nil {
		t.Fatal(err)
	}
	state, ok, err := run.Load()
	if err != nil || !ok {
		t.Fatalf("no run state: %v", err)
	}
	return state.Workspaces[canonical]
}

func TestRunFlow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys := writeFixtureKeys(t)
	ws := filepath.Join(home, "ws")

	code, out, _ := runCLI("start", "--keys", keys)
	if code != 0 || !strings.Contains(out, "run started") {
		t.Fatalf("start: exit = %d, out = %q", code, out)
	}
	code, out, _ = runCLI("build", ws, "redis", "1")
	if code != 0 {
		t.Fatalf("build: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "built redis stage 1") || !strings.Contains(out, filepath.Join("ws", "s01")) {
		t.Errorf("build out = %q", out)
	}
	stage, err := os.ReadFile(filepath.Join(ws, "s01", "stage.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stage), "go:build") || !strings.Contains(string(stage), "return 2") {
		t.Errorf("stage.go should hold the injected build without a build tag:\n%s", stage)
	}
	if _, err := os.Stat(filepath.Join(ws, "s01", "correct.go")); !os.IsNotExist(err) {
		t.Error("correct.go must not be copied")
	}
	if n := gitCount(t, ws); n != 1 {
		t.Errorf("commits = %d, want 1", n)
	}
	info, _ := os.Stat(filepath.Join(ws, "s01", "stage.go"))
	if info.Mode()&0o222 != 0 {
		t.Error("workspace must be read-only after build")
	}
	if st := stageStatus(t, ws); st.Status != "locked" || st.BuildCommit == "" || st.Dir != "s01" {
		t.Errorf("state = %+v", st)
	}
	if code, _, _ := runCLI("build", ws, "redis", "1"); code != 2 {
		t.Errorf("second build: exit = %d, want 2", code)
	}
	if code, _, _ := runCLI("fix", ws); code != 2 {
		t.Errorf("fix while locked: exit = %d, want 2", code)
	}
	if code, _, _ := runCLI("review", t.TempDir()); code != 2 {
		t.Errorf("review of a dir outside the run: exit = %d, want 2", code)
	}

	setupModel(t, []bool{true, false, true})
	bad := writeAnswer(t, "It is length prefixed.")
	code, out, _ = runCLI("unlock", ws, bad)
	if code != 1 || !strings.Contains(out, "still locked") {
		t.Fatalf("bad unlock: exit = %d, out = %q", code, out)
	}
	if st := stageStatus(t, ws); st.Status != "locked" {
		t.Errorf("state after bad answer = %+v", st)
	}

	setupModel(t, []bool{true, true, true})
	good := writeAnswer(t, "Bulk strings are length prefixed, so the reader reads exactly that many bytes, then the CRLF.")
	code, out, _ = runCLI("unlock", ws, good)
	if code != 0 {
		t.Fatalf("good unlock: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "Grading this answer:") || !strings.Contains(out, "unlocked "+ws) {
		t.Errorf("unlock out = %q", out)
	}
	if st := stageStatus(t, ws); st.Status != "unlocked" {
		t.Errorf("state after unlock = %+v", st)
	}

	code, out, _ = runCLI("fix", ws)
	if code != 0 {
		t.Fatalf("fix: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "fixed redis stage 1") || !strings.Contains(out, "PASS") {
		t.Errorf("fix out = %q", out)
	}
	if st := stageStatus(t, ws); st.Status != "fixed" {
		t.Errorf("state after fix = %+v", st)
	}
	if n := gitCount(t, ws); n != 2 {
		t.Errorf("commits = %d, want 2", n)
	}
	fixed, _ := os.ReadFile(filepath.Join(ws, "s01", "stage.go"))
	if !strings.Contains(string(fixed), "return 1") {
		t.Error("stage.go should now hold the correct build")
	}
	code, out, _ = runCLI("review", ws)
	if code != 0 || !strings.Contains(out, "stage.go") {
		t.Errorf("review: exit = %d, out = %q", code, out)
	}
	code, out, _ = runCLI("stop")
	if code != 0 || !strings.Contains(out, "run stopped") {
		t.Fatalf("stop: exit = %d, out = %q", code, out)
	}
	if path, _ := run.File(); true {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("run.json should be deleted by stop")
		}
	}
}

func TestRunFlowNamedPairs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys := writeFixtureKeys(t)
	ws := filepath.Join(home, "ws2")

	if code, out, _ := runCLI("start", "--keys", keys); code != 0 {
		t.Fatalf("start: %s", out)
	}
	code, out, _ := runCLI("build", ws, "redis", "2")
	if code != 0 {
		t.Fatalf("build: exit = %d, out = %q", code, out)
	}
	stage := filepath.Join(ws, "s02")
	for _, rel := range []string{"x_stage.go", filepath.Join("cmd", "tool", "y_stage.go")} {
		data, err := os.ReadFile(filepath.Join(stage, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if strings.Contains(string(data), "go:build") || !strings.Contains(string(data), "return 2") {
			t.Errorf("%s should hold the injected build without a tag:\n%s", rel, data)
		}
	}
	if err := filepath.WalkDir(stage, func(path string, d fs.DirEntry, err error) error {
		if err == nil && (strings.Contains(d.Name(), "correct") || strings.Contains(d.Name(), "injected")) {
			t.Errorf("build leaked %s", path)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}

	setupModel(t, []bool{true, true, true})
	good := writeAnswer(t, "The store is a map, commands are dispatched by name, and replies use RESP types.")
	code, out, _ = runCLI("unlock", ws, good)
	if code != 0 {
		t.Fatalf("unlock: exit = %d, out = %q", code, out)
	}
	code, out, _ = runCLI("fix", ws)
	if code != 0 {
		t.Fatalf("fix: exit = %d, out = %q", code, out)
	}
	for _, rel := range []string{"x_stage.go", filepath.Join("cmd", "tool", "y_stage.go")} {
		data, err := os.ReadFile(filepath.Join(stage, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "return 1") {
			t.Errorf("%s should now hold the correct build:\n%s", rel, data)
		}
	}
	if code, _, _ := runCLI("stop"); code != 0 {
		t.Error("stop failed")
	}
}

func TestBuildNeedsRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if code, _, _ := runCLI("build", t.TempDir(), "redis", "1"); code != 2 {
		t.Error("build without a run: want exit 2")
	}
	if code, _, _ := runCLI("unlock", t.TempDir(), "x"); code != 2 {
		t.Error("unlock without a run: want exit 2")
	}
}

func TestStartNeedsKeys(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ANVIL_KEYS", "")
	if code, _, _ := runCLI("start"); code != 2 {
		t.Error("start without keys: want exit 2")
	}
}
