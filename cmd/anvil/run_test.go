package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"anvil/internal/run"
)

const fixtureKeysTwo = `{
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
`

// writeFixtureKeys builds a fake keys repo in the projects/ layout:
// keys/redis.json plus base/ and two stage overlays for redis.
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
	write("keys/redis.json", fixtureKeysTwo)
	write("projects/redis/base/go.mod", "module redis\n\ngo 1.27\n")
	write("projects/redis/stages/01/correct/a.go", "package redis\n\nfunc A() int { return 1 }\n")
	write("projects/redis/stages/01/correct/a_test.go", "package redis\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {\n\tif A() != 1 {\n\t\tt.Fatalf(\"got %d, want 1\", A())\n\t}\n}\n")
	write("projects/redis/stages/01/injected/a.go", "package redis\n\nfunc A() int { return 2 }\n")
	write("projects/redis/stages/02/correct/a.go", "package redis\n\nfunc A() int { return 1 }\n")
	write("projects/redis/stages/02/correct/b.go", "package redis\n\nfunc B() int { return 1 }\n")
	write("projects/redis/stages/02/correct/b_test.go", "package redis\n\nimport \"testing\"\n\nfunc TestB(t *testing.T) {\n\tif B() != 1 {\n\t\tt.Fatalf(\"got %d, want 1\", B())\n\t}\n}\n")
	write("projects/redis/stages/02/injected/b.go", "package redis\n\nfunc B() int { return 2 }\n")
	return dir
}

func gitLog(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "--format=%s").Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
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
	if !strings.Contains(out, "built redis stage 1") {
		t.Errorf("build out = %q", out)
	}
	if got := readFile(t, filepath.Join(ws, "a.go")); !strings.Contains(got, "return 2") {
		t.Errorf("a.go should hold the injected body:\n%s", got)
	}
	info, _ := os.Stat(filepath.Join(ws, "a.go"))
	if info.Mode()&0o222 != 0 {
		t.Error("workspace must be read-only after build")
	}
	if st := stageStatus(t, ws); st.Status != "locked" || st.BuildCommit == "" || st.Attempts != 0 {
		t.Errorf("state = %+v", st)
	}
	if code, _, _ := runCLI("review", ws); code != 0 {
		t.Fatal("review failed")
	}
	if _, out, _ := runCLI("review", ws); strings.Contains(out, "Reference fix") {
		t.Error("review must not show the reference before fixed")
	}

	setupModel(t, []bool{true, false, true})
	bad := writeAnswer(t, "It is length prefixed.")
	code, out, _ = runCLI("unlock", ws, bad)
	if code != 1 || !strings.Contains(out, "still locked") {
		t.Fatalf("bad unlock: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "attempt 1") {
		t.Errorf("expected the attempt counter in %q", out)
	}
	if st := stageStatus(t, ws); st.Status != "locked" || st.Attempts != 1 {
		t.Errorf("state after bad answer = %+v", st)
	}

	setupModel(t, []bool{true, true, true})
	good := writeAnswer(t, "Bulk strings are length prefixed, so the reader reads exactly that many bytes, then the CRLF.")
	code, out, _ = runCLI("unlock", ws, good)
	if code != 0 {
		t.Fatalf("good unlock: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "Grading this answer:") || !strings.Contains(out, "write your fix, then run anvil check") {
		t.Errorf("unlock out = %q", out)
	}
	if st := stageStatus(t, ws); st.Status != "unlocked" {
		t.Errorf("state after unlock = %+v", st)
	}

	code, out, _ = runCLI("check", ws)
	if code != 1 {
		t.Fatalf("check with the bug: exit = %d, out = %q", code, out)
	}
	if st := stageStatus(t, ws); st.Status != "unlocked" {
		t.Errorf("state after failed check = %+v", st)
	}

	// The learner writes the fix.
	correct := filepath.Join(keys, "projects", "redis", "stages", "01", "correct", "a.go")
	data, err := os.ReadFile(correct)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "a.go"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runCLI("check", ws)
	if code != 0 {
		t.Fatalf("check: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "fixed redis stage 1") || !strings.Contains(out, "PASS") {
		t.Errorf("check out = %q", out)
	}
	if st := stageStatus(t, ws); st.Status != "fixed" || st.Revealed {
		t.Errorf("state after check = %+v", st)
	}
	if log := gitLog(t, ws); !strings.Contains(log, "redis stage 1: fix") {
		t.Errorf("git log = %q", log)
	}
	code, out, _ = runCLI("review", ws)
	if code != 0 {
		t.Fatalf("review: exit = %d", code)
	}
	if !strings.Contains(out, "a.go") || !strings.Contains(out, "=== Reference fix ===") {
		t.Errorf("review out = %q", out)
	}

	code, out, _ = runCLI("build", ws, "redis", "2")
	if code != 0 {
		t.Fatalf("build 2: exit = %d, out = %q", code, out)
	}
	if st := stageStatus(t, ws); st.Status != "locked" || st.Number != 2 || st.Attempts != 0 {
		t.Errorf("state after build 2 = %+v", st)
	}
	if got := readFile(t, filepath.Join(ws, "b.go")); !strings.Contains(got, "return 2") {
		t.Errorf("b.go should hold the injected body:\n%s", got)
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

func TestBuildStageTwoFreshAppliesStageOne(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys := writeFixtureKeys(t)
	ws := filepath.Join(home, "ws")
	if code, out, _ := runCLI("start", "--keys", keys); code != 0 {
		t.Fatalf("start: %s", out)
	}
	code, out, _ := runCLI("build", ws, "redis", "2")
	if code != 0 {
		t.Fatalf("build: exit = %d, out = %q", code, out)
	}
	if got := readFile(t, filepath.Join(ws, "a.go")); !strings.Contains(got, "return 1") {
		t.Errorf("a.go should hold stage 1+2's correct body:\n%s", got)
	}
	if got := readFile(t, filepath.Join(ws, "b.go")); !strings.Contains(got, "return 2") {
		t.Errorf("b.go should hold the injected body:\n%s", got)
	}
	runCLI("stop")
}

func TestBuildNextStageNeedsFixed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys := writeFixtureKeys(t)
	ws := filepath.Join(home, "ws")
	runCLI("start", "--keys", keys)
	if code, out, _ := runCLI("build", ws, "redis", "1"); code != 0 {
		t.Fatalf("build 1: %s", out)
	}
	if code, _, _ := runCLI("build", ws, "redis", "2"); code != 2 {
		t.Error("build 2 while stage 1 is locked: want exit 2")
	}
	runCLI("stop")
}

func TestRevealAfterFiveAttempts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys := writeFixtureKeys(t)
	ws := filepath.Join(home, "ws")
	runCLI("start", "--keys", keys)
	if code, out, _ := runCLI("build", ws, "redis", "1"); code != 0 {
		t.Fatalf("build: %s", out)
	}
	code, out, _ := runCLI("reveal", ws)
	if code != 1 || !strings.Contains(out, "5 more graded answers") {
		t.Fatalf("reveal too early: exit = %d, out = %q", code, out)
	}
	setupModel(t, []bool{true, false, true})
	bad := writeAnswer(t, "It is length prefixed.")
	for i := 1; i <= 5; i++ {
		if code, _, _ := runCLI("unlock", ws, bad); code != 1 {
			t.Fatalf("unlock %d failed", i)
		}
	}
	if st := stageStatus(t, ws); st.Attempts != 5 {
		t.Fatalf("attempts = %d, want 5", st.Attempts)
	}
	code, out, _ = runCLI("reveal", ws)
	if code != 0 {
		t.Fatalf("reveal: exit = %d, out = %q", code, out)
	}
	if !strings.Contains(out, "Bulk strings are length prefixed") || !strings.Contains(out, "recorded as revealed") {
		t.Errorf("reveal out = %q", out)
	}
	if st := stageStatus(t, ws); !st.Revealed || st.Status != "unlocked" {
		t.Errorf("state after reveal = %+v", st)
	}
	correct := filepath.Join(keys, "projects", "redis", "stages", "01", "correct", "a.go")
	data, err := os.ReadFile(correct)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "a.go"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runCLI("check", ws)
	if code != 0 {
		t.Fatalf("check: exit = %d, out = %q", code, out)
	}
	if log := gitLog(t, ws); !strings.Contains(log, "redis stage 1: fix (revealed)") {
		t.Errorf("git log = %q", log)
	}
	runCLI("stop")
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
