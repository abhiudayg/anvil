package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

const lockUsage = `usage:
  anvil lock <dir>
  anvil unlock [--keys DIR] <dir> <project> <stage-number> <answer-file>
  anvil review <dir>

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

// runLock makes a workspace read-only. Exit codes: 0 ok, 2 usage or setup error.
func runLock(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, lockUsage)
		return 2
	}
	dir := args[0]
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "anvil lock: %s is not a directory\n", dir)
		return 2
	}
	if err := setWritable(dir, false); err != nil {
		fmt.Fprintf(stderr, "anvil lock: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "locked %s\n", dir)
	return 0
}

// runUnlock restores write access only when the answer grades PERFECT.
// Exit codes: 0 unlocked, 1 not perfect, 2 usage or setup error.
func runUnlock(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("unlock", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keysDir := fs.String("keys", os.Getenv("ANVIL_KEYS"), "keys repository directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 4 {
		fmt.Fprint(stderr, lockUsage)
		return 2
	}
	dir := rest[0]
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "anvil unlock: %s is not a directory\n", dir)
		return 2
	}
	code := gradeStage(*keysDir, rest[1], rest[2], rest[3], stdout, stderr)
	if code != 0 {
		if code == 1 {
			fmt.Fprintf(stdout, "still locked: %s\n", dir)
		}
		return code
	}
	if err := setWritable(dir, true); err != nil {
		fmt.Fprintf(stderr, "anvil unlock: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "unlocked %s: the agent may now write the fix.\n", dir)
	return 0
}

// runReview shows every change since the last commit, including untracked files.
// Exit codes: 0 ok, 2 usage or setup error.
func runReview(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, lockUsage)
		return 2
	}
	dir := args[0]
	check := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	if out, err := check.Output(); err != nil || string(out) != "true\n" {
		fmt.Fprintf(stderr, "anvil review: %s is not a git work tree\n", dir)
		return 2
	}
	head := exec.Command("git", "-C", dir, "rev-parse", "--verify", "HEAD")
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
	diff := exec.Command("git", "-C", dir, "diff", "HEAD", "--stat", "--patch")
	diff.Stdout = stdout
	diff.Stderr = stderr
	if err := diff.Run(); err != nil {
		fmt.Fprintf(stderr, "anvil review: %v\n", err)
		return 2
	}
	return 0
}
