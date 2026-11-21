package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
  anvil fix <workspace>
  anvil review <workspace>

--keys defaults to $ANVIL_KEYS, the keys repository.
`

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

// stripBuildTag removes the leading //go:build line and the blank line
// after it from an injected or correct source file.
func stripBuildTag(src []byte) []byte {
	text := string(src)
	if !strings.HasPrefix(text, "//go:build") {
		return src
	}
	text = text[strings.IndexByte(text, '\n')+1:]
	return []byte(strings.TrimPrefix(text, "\n"))
}

// stageDirName finds the verify stage dir under keys/verify/<slug> whose
// name ends in the stage number's digits.
func stageDirName(keysDir, slug string, number int) (string, error) {
	entries, err := os.ReadDir(filepath.Join(keysDir, "verify", slug))
	if err != nil {
		return "", err
	}
	trailing := regexp.MustCompile(`\d+$`)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if digits := trailing.FindString(e.Name()); digits != "" {
			if n, err := strconv.Atoi(digits); err == nil && n == number {
				return e.Name(), nil
			}
		}
	}
	return "", fmt.Errorf("no verify stage %d in %s", number, slug)
}

// stageName maps a build-tagged source file to its workspace name:
// injected.go and correct.go become stage.go; other names get their
// first "injected" or "correct" replaced by "stage".
func stageName(name string) string {
	switch name {
	case "injected.go", "correct.go":
		return "stage.go"
	}
	if i := strings.Index(name, "injected"); i >= 0 {
		return name[:i] + "stage" + name[i+len("injected"):]
	}
	if i := strings.Index(name, "correct"); i >= 0 {
		return name[:i] + "stage" + name[i+len("correct"):]
	}
	return name
}

// copyStage copies the verify stage dir into dst. Files tagged
// !injected are skipped; files tagged injected are written without the
// build tag under their stageName. Everything else is copied unchanged.
func copyStage(src, dst string) error {
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
		switch firstLine(data) {
		case "//go:build !injected":
			return nil
		case "//go:build injected":
			name := stageName(filepath.Base(target))
			return os.WriteFile(filepath.Join(filepath.Dir(target), name), stripBuildTag(data), 0o644)
		default:
			return os.WriteFile(target, data, 0o644)
		}
	})
}

func firstLine(data []byte) string {
	if i := strings.IndexByte(string(data), '\n'); i >= 0 {
		return string(data[:i])
	}
	return string(data)
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

// runBuild copies the stage's injected build into the workspace and locks it.
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
	dirName, err := stageDirName(state.Keys, step.Project.Slug, number)
	if err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
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
	if stage, exists := state.Workspaces[ws]; exists && stage.Status != "fixed" {
		fmt.Fprintf(stderr, "anvil build: %s stage %d is not finished\n", step.Project.Slug, number)
		return 2
	}
	gomod := filepath.Join(ws, "go.mod")
	if _, err := os.Stat(gomod); os.IsNotExist(err) {
		data, err := os.ReadFile(filepath.Join(state.Keys, "verify", "go.mod"))
		if err != nil {
			fmt.Fprintf(stderr, "anvil build: %v\n", err)
			return 2
		}
		lines := strings.SplitN(string(data), "\n", 2)
		lines[0] = "module " + step.Project.Slug
		if err := os.WriteFile(gomod, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			fmt.Fprintf(stderr, "anvil build: %v\n", err)
			return 2
		}
		sum, err := os.ReadFile(filepath.Join(state.Keys, "verify", "go.sum"))
		if err == nil {
			if err := os.WriteFile(filepath.Join(ws, "go.sum"), sum, 0o644); err != nil {
				fmt.Fprintf(stderr, "anvil build: %v\n", err)
				return 2
			}
		}
		if _, err := os.Stat(filepath.Join(ws, ".git")); os.IsNotExist(err) {
			if err := gitCmd(ws, stdout, stderr, "init", "-q"); err != nil {
				fmt.Fprintf(stderr, "anvil build: %v\n", err)
				return 2
			}
		}
	}
	src := filepath.Join(state.Keys, "verify", step.Project.Slug, dirName)
	dst := filepath.Join(ws, dirName)
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "anvil build: %s already exists\n", dst)
		return 2
	}
	if err := copyStage(src, dst); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	message := fmt.Sprintf("%s stage %d: build", step.Project.Slug, number)
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
		Project:     step.Project.Slug,
		Number:      number,
		Dir:         dirName,
		Status:      "locked",
		BuildCommit: head,
	}
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil build: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "built %s stage %d in %s; the workspace is locked.\n", step.Project.Slug, number, dst)
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
	fmt.Fprintf(stdout, "unlocked %s: the agent can now run anvil fix.\n", args[0])
	return 0
}

// runFix writes the correct build over stage.go, builds and tests it in
// the workspace, and commits on success.
// Exit codes: 0 fixed, 1 build or test failed, 2 usage or setup error.
func runFix(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	state, ok, err := run.Load()
	if err != nil {
		fmt.Fprintf(stderr, "anvil fix: %v\n", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "anvil fix: no run is active")
		return 2
	}
	ws, err := run.Canonical(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "anvil fix: %v\n", err)
		return 2
	}
	stage, exists := state.Workspaces[ws]
	if !exists || stage.Status != "unlocked" {
		fmt.Fprintf(stderr, "anvil fix: %s is not an unlocked workspace\n", args[0])
		return 2
	}
	src := filepath.Join(state.Keys, "verify", stage.Project, stage.Dir)
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if firstLine(data) != "//go:build !injected" {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(ws, stage.Dir, filepath.Dir(rel), stageName(filepath.Base(rel)))
		return os.WriteFile(target, stripBuildTag(data), 0o644)
	})
	if err != nil {
		fmt.Fprintf(stderr, "anvil fix: %v\n", err)
		return 2
	}
	out, code := buildAndTest(ws, stage.Dir, stdout, stderr)
	if code != 0 {
		return code
	}
	if strings.Contains(out, "--- SKIP") {
		fmt.Fprintln(stdout, "fix not verified: a test skipped (see above)")
		return 1
	}
	message := fmt.Sprintf("%s stage %d: fix", stage.Project, stage.Number)
	if err := gitCommit(ws, stdout, stderr, message); err != nil {
		fmt.Fprintf(stderr, "anvil fix: %v\n", err)
		return 2
	}
	stage.Status = "fixed"
	state.Workspaces[ws] = stage
	if err := run.Save(state); err != nil {
		fmt.Fprintf(stderr, "anvil fix: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "fixed %s stage %d. Run anvil review %s to see the changes.\n", stage.Project, stage.Number, args[0])
	return 0
}

// buildAndTest runs go build and go test in ws, streaming and capturing
// their combined output. Exit 0 means both passed.
func buildAndTest(ws, dir string, stdout, stderr io.Writer) (string, int) {
	var captured strings.Builder
	tee := io.MultiWriter(stdout, &captured)
	build := exec.Command("go", "build", "./...")
	build.Dir = ws
	build.Stdout = tee
	build.Stderr = tee
	if err := build.Run(); err != nil {
		fmt.Fprintf(tee, "anvil fix: go build: %v\n", err)
		return captured.String(), 1
	}
	test := exec.Command("go", "test", "-race", "-count=1", "-v", "./"+dir+"/...")
	test.Dir = ws
	test.Stdout = tee
	test.Stderr = tee
	if err := test.Run(); err != nil {
		fmt.Fprintf(tee, "anvil fix: go test: %v\n", err)
		return captured.String(), 1
	}
	return captured.String(), 0
}

// runReview shows every change since the build commit (during a run) or
// the last commit (no run), including untracked files.
// Exit codes: 0 ok, 2 usage or setup error.
func runReview(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, runUsage)
		return 2
	}
	dir := args[0]
	base := "HEAD"
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
		stage, exists := state.Workspaces[ws]
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
	return 0
}
