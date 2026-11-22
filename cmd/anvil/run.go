package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"anvil/internal/grader"
	"anvil/internal/run"
)

const runUsage = `usage:
  anvil start [--keys DIR]
  anvil stop
  anvil build <workspace> <project> <stage-number>
  anvil unlock <workspace> <answer-file>
  anvil reveal <workspace>
  anvil check <workspace>
  anvil review <workspace>

--keys defaults to $ANVIL_KEYS, the keys repository.
`

// revealAfter is how many graded not-perfect answers unlock anvil reveal.
const revealAfter = 5

// setWritable walks dir and clears (writable=false) or restores (writable=true)
// the owner write bit on every file and directory. The .git directory and
// symlinks are skipped: chmod follows symlinks outside the tree.
func setWritable(dir string, writable bool) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != dir && d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		if writable {
			mode |= 0o200
		} else {
			mode &^= 0o222
		}
		return os.Chmod(path, mode)
	})
}

// projectRoot is the keys repo's project directory.
func projectRoot(keysDir, slug string) string {
	return filepath.Join(keysDir, "projects", slug)
}

// stageOverlay is a stage's correct or injected overlay directory.
func stageOverlay(keysDir, slug, overlay string, number int) string {
	return filepath.Join(projectRoot(keysDir, slug), "stages",
		fmt.Sprintf("%02d", number), overlay)
}

// applyOverlay copies every file under src to the same relative path in
// dst, overwriting existing files. Files get 0644, dirs 0755.
func applyOverlay(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// gitCmd runs a git command in dir, streaming output to stdout/stderr.
func gitCmd(dir string, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func gitCommit(dir string, stdout, stderr io.Writer, message string) error {
	if err := gitCmd(dir, stdout, stderr, "add", "-A"); err != nil {
		return err
	}
	return gitCmd(dir, stdout, stderr,
		"-c", "user.name=Anvil", "-c", "user.email=anvil@localhost",
		"commit", "-q", "-m", message)
}

func gitHead(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// recordAttempt counts one more not-perfect graded answer for the
// locked workspace building this stage. It returns the new count.
func recordAttempt(project string, number int) int {
	state, ok, err := run.Load()
	if err != nil || !ok {
		return 0
	}
	for ws, stage := range state.Workspaces {
		if stage.Project == project && stage.Number == number && stage.Status == "locked" {
			stage.Attempts++
			state.Workspaces[ws] = stage
			if err := run.Save(state); err != nil {
				return 0
			}
			return stage.Attempts
		}
	}
	return 0
}

// attemptNote prints the reveal progress line after a not-perfect grade.
func attemptNote(project string, number int, stdout io.Writer) {
	if k := recordAttempt(project, number); k > 0 {
		if k >= revealAfter {
			fmt.Fprintln(stdout, "you can now run anvil reveal <ws> in your own terminal.")
		} else {
			fmt.Fprintf(stdout, "attempt %d; anvil reveal becomes available after %d.\n", k, revealAfter)
		}
	}
}

// runStart begins a run: it records the keys repo so build, unlock and
// the guard hook know what is protected.
func runStart(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keysDir := fs.String("keys", os.Getenv("ANVIL_KEYS"), "keys repository directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	if *keysDir == "" {
		fmt.Fprintln(stderr, "anvil start: no keys directory (use --keys or set ANVIL_KEYS)")
		return 2
	}
	canonical, err := run.Canonical(*keysDir)
	if err != nil {
		fmt.Fprintf(stderr, "anvil start: %v\n", err)
		return 2
	}
	if _, err := grader.Load(canonical); err != nil {
		fmt.Fprintf(stderr, "anvil start: %v\n", err)
		return 2
	}
	state, _, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil start: %v\n", err)
		return 2
	}
	state.Keys = canonical
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil start: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, "run started: agent tool calls are restricted until you run anvil stop in your own terminal.")
	return 0
}

// runStop ends the run: locked workspaces become writable again.
func runStop(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil stop: %v\n", err)
		return 2
	}
	if ok {
		for ws, stage := range state.Workspaces {
			if stage.Status == "locked" {
				if err := setWritable(ws, true); err != nil {
					fmt.Fprintf(stderr, "anvil stop: %v\n", err)
					return 2
				}
			}
		}
	}
	if err := run.Delete(); err != nil {
		fmt.Fprintf(stderr, "anvil stop: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, "run stopped")
	return 0
}

// runBuild lays the stage's bugged code into the workspace and locks it.
// A fresh workspace gets the project base plus every earlier stage's
// correct overlay; an existing one must have finished the previous stage.
// Exit codes: 0 built, 2 usage or setup error.
func runBuild(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "anvil build: no run is active (run anvil start first)")
		return 2
	}
	number, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Fprintf(stderr, "anvil build: stage number must be a number, got %q\n", args[2])
		return 2
	}
	step, found := findStep(args[1], number)
	if !found {
		fmt.Fprintf(stderr, "anvil build: no stage %d in project %q\n", number, args[1])
		return 2
	}
	slug := step.Project.Slug
	fresh := false
	if _, err := os.Stat(filepath.Join(args[0], "go.mod")); os.IsNotExist(err) {
		fresh = true
	}
	if err := os.MkdirAll(args[0], 0o755); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	ws, err := run.Canonical(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	if fresh {
		base := filepath.Join(projectRoot(state.Keys, slug), "base")
		if _, err := os.Stat(filepath.Join(base, "go.mod")); err != nil {
			fmt.Fprintf(stderr, "anvil build: %s has no base/go.mod\n", slug)
			return 2
		}
		if err := applyOverlay(base, ws); err != nil {
			fmt.Fprintf(stderr, "anvil build: %v\n", err)
			return 2
		}
		if _, err := os.Stat(filepath.Join(ws, ".git")); os.IsNotExist(err) {
			if err := gitCmd(ws, stdout, stderr, "init", "-q"); err != nil {
				fmt.Fprintf(stderr, "anvil build: %v\n", err)
				return 2
			}
		}
		for n := 1; n < number; n++ {
			if err := applyOverlay(stageOverlay(state.Keys, slug, "correct", n), ws); err != nil {
				fmt.Fprintf(stderr, "anvil build: %v\n", err)
				return 2
			}
		}
	} else {
		stage, exists := state.Workspaces[ws]
		if !exists || stage.Status != "fixed" || stage.Number != number-1 {
			fmt.Fprintf(stderr, "anvil build: %s stage %d needs stage %d fixed first\n", slug, number, number-1)
			return 2
		}
	}
	if err := applyOverlay(stageOverlay(state.Keys, slug, "correct", number), ws); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	if err := applyOverlay(stageOverlay(state.Keys, slug, "injected", number), ws); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	message := fmt.Sprintf("%s stage %d: build", slug, number)
	if err := gitCommit(ws, stdout, stderr, message); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	head, err := gitHead(ws)
	if err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	if err := setWritable(ws, false); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	state.Workspaces[ws] = run.Stage{
		Project:     slug,
		Number:      number,
		Status:      "locked",
		BuildCommit: head,
	}
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "built %s stage %d in %s; the workspace is locked.\n", slug, number, ws)
	return 0
}

// runUnlock grades the learner's answer; PERFECT unlocks the workspace.
// Exit codes: 0 unlocked, 1 not perfect, 2 usage or setup error.
func runUnlock(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil unlock: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "anvil unlock: no run is active")
		return 2
	}
	ws, err := run.Canonical(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "anvil unlock: %v\n", err)
		return 2
	}
	stage, exists := state.Workspaces[ws]
	if !exists || stage.Status != "locked" {
		fmt.Fprintf(stderr, "anvil unlock: %s is not a locked workspace\n", args[0])
		return 2
	}
	answer, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "anvil unlock: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "Grading this answer:\n%s\n", answer)
	code := gradeStage(state.Keys, stage.Project, strconv.Itoa(stage.Number), string(answer), stdout, stderr)
	if code != 0 {
		if code == 1 {
			fmt.Fprintf(stdout, "still locked: %s\n", args[0])
		}
		return code
	}
	if err := setWritable(ws, true); err != nil {
		fmt.Fprintf(stderr, "anvil unlock: %v\n", err)
		return 2
	}
	stage.Status = "unlocked"
	state.Workspaces[ws] = stage
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil unlock: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "unlocked %s: write your fix, then run anvil check %s.\n", args[0], args[0])
	return 0
}

// runReveal shows the reference answer after revealAfter graded tries.
// The stage is recorded as revealed, not passed.
func runReveal(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil reveal: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "anvil reveal: no run is active")
		return 2
	}
	ws, err := run.Canonical(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "anvil reveal: %v\n", err)
		return 2
	}
	stage, exists := state.Workspaces[ws]
	if !exists || stage.Status != "locked" {
		fmt.Fprintf(stderr, "anvil reveal: %s is not a locked workspace\n", args[0])
		return 2
	}
	if stage.Attempts < revealAfter {
		fmt.Fprintf(stdout, "%d more graded answers before reveal\n", revealAfter-stage.Attempts)
		return 1
	}
	keys, err := grader.Load(state.Keys)
	if err != nil {
		fmt.Fprintf(stderr, "anvil reveal: %v\n", err)
		return 2
	}
	step, found := findStep(stage.Project, stage.Number)
	if !found {
		fmt.Fprintf(stderr, "anvil reveal: no stage %d in project %q\n", stage.Number, stage.Project)
		return 2
	}
	key, found := grader.Lookup(keys, stage.Project, step.Stage.Title)
	if !found {
		fmt.Fprintf(stderr, "anvil reveal: no key for %s stage %d\n", stage.Project, stage.Number)
		return 2
	}
	fmt.Fprintln(stdout, key.ReferenceAnswer)
	if err := setWritable(ws, true); err != nil {
		fmt.Fprintf(stderr, "anvil reveal: %v\n", err)
		return 2
	}
	stage.Revealed = true
	stage.Status = "unlocked"
	state.Workspaces[ws] = stage
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil reveal: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "revealed: this stage will be recorded as revealed, not passed. Write your fix, then run anvil check %s.\n", args[0])
	return 0
}

// runCheck builds and tests the learner's fix in the workspace and
// commits it when it passes.
// Exit codes: 0 fixed, 1 build or test failed, 2 usage or setup error.
func runCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil check: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "anvil check: no run is active")
		return 2
	}
	ws, err := run.Canonical(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "anvil check: %v\n", err)
		return 2
	}
	stage, exists := state.Workspaces[ws]
	if !exists || stage.Status != "unlocked" {
		fmt.Fprintf(stderr, "anvil check: %s is not an unlocked workspace\n", args[0])
		return 2
	}
	out, code := buildAndTest(ws, stdout)
	if code != 0 {
		return code
	}
	if strings.Contains(out, "--- SKIP") {
		fmt.Fprintln(stdout, "fix not verified: a test skipped (see above)")
		return 1
	}
	message := fmt.Sprintf("%s stage %d: fix", stage.Project, stage.Number)
	if stage.Revealed {
		message += " (revealed)"
	}
	if err := gitCommit(ws, stdout, stderr, message); err != nil {
		fmt.Fprintf(stderr, "anvil check: %v\n", err)
		return 2
	}
	stage.Status = "fixed"
	state.Workspaces[ws] = stage
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil check: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "fixed %s stage %d. Run anvil review %s to compare with the reference fix.\n", stage.Project, stage.Number, args[0])
	return 0
}

// buildAndTest runs go build and go test in ws, streaming and capturing
// their combined output. Exit 0 means both passed.
func buildAndTest(ws string, stdout io.Writer) (string, int) {
	var captured strings.Builder
	tee := io.MultiWriter(stdout, &captured)
	build := exec.Command("go", "build", "./...")
	build.Dir = ws
	build.Stdout = tee
	build.Stderr = tee
	if err := build.Run(); err != nil {
		fmt.Fprintf(tee, "anvil check: go build: %v\n", err)
		return captured.String(), 1
	}
	test := exec.Command("go", "test", "-race", "-count=1", "-v", "./...")
	test.Dir = ws
	test.Stdout = tee
	test.Stderr = tee
	if err := test.Run(); err != nil {
		fmt.Fprintf(tee, "anvil check: go test: %v\n", err)
		return captured.String(), 1
	}
	return captured.String(), 0
}

// runReview shows the learner's diff since the build commit, then — once
// the stage is fixed — the reference fix for each bugged file.
// Exit codes: 0 ok, 2 usage or setup error.
func runReview(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	dir := args[0]
	base := "HEAD"
	var stage run.Stage
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil review: %v\n", err)
		return 2
	}
	if ok {
		ws, err := run.Canonical(dir)
		if err != nil {
			fmt.Fprintf(stderr, "anvil review: %v\n", err)
			return 2
		}
		var exists bool
		stage, exists = state.Workspaces[ws]
		if !exists {
			fmt.Fprintf(stderr, "anvil review: %s is not part of this run\n", dir)
			return 2
		}
		base = stage.BuildCommit
	}
	check := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	if out, err := check.Output(); err != nil || string(out) != "true\n" {
		fmt.Fprintf(stderr, "anvil review: %s is not a git work tree\n", dir)
		return 2
	}
	head := exec.Command("git", "-C", dir, "rev-parse", "--verify", base)
	if err := head.Run(); err != nil {
		fmt.Fprintf(stderr, "anvil review: %s has no commits yet\n", dir)
		return 2
	}
	add := exec.Command("git", "-C", dir, "add", "--intent-to-add", "--all")
	add.Stdout = stdout
	add.Stderr = stderr
	if err := add.Run(); err != nil {
		fmt.Fprintf(stderr, "anvil review: %v\n", err)
		return 2
	}
	diff := exec.Command("git", "-C", dir, "diff", base, "--stat", "--patch")
	diff.Stdout = stdout
	diff.Stderr = stderr
	if err := diff.Run(); err != nil {
		fmt.Fprintf(stderr, "anvil review: %v\n", err)
		return 2
	}
	if !ok || stage.Status != "fixed" {
		return 0
	}
	injected := stageOverlay(state.Keys, stage.Project, "injected", stage.Number)
	var files []string
	if err := filepath.WalkDir(injected, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		fmt.Fprintf(stderr, "anvil review: %v\n", err)
		return 2
	}
	sort.Strings(files)
	fmt.Fprintln(stdout, "=== Reference fix ===")
	for _, inj := range files {
		rel, err := filepath.Rel(injected, inj)
		if err != nil {
			fmt.Fprintf(stderr, "anvil review: %v\n", err)
			return 2
		}
		correct := filepath.Join(stageOverlay(state.Keys, stage.Project, "correct", stage.Number), rel)
		cmd := exec.Command("git", "diff", "--no-index", "--", inj, correct)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				fmt.Fprintf(stderr, "anvil review: %v\n", err)
				return 2
			}
		}
	}
	return 0
}
