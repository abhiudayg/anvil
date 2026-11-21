// Package lockstate tracks which workspaces are locked in ~/.anvil/locks.
// The guard hook reads it; lock and unlock write it.
package lockstate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".anvil"), nil
}

// LocksFile returns the path of the locked-workspace list, ~/.anvil/locks.
func LocksFile() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "locks"), nil
}

// AnswersDir returns the directory agents may write into, ~/.anvil/answers.
func AnswersDir() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "answers"), nil
}

// Locked returns the locked workspace directories, one per line.
// A missing locks file is not an error.
func Locked() ([]string, error) {
	path, err := LocksFile()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			dirs = append(dirs, line)
		}
	}
	return dirs, nil
}

func write(dirs []string) error {
	path, err := LocksFile()
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "locks-*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(strings.Join(dirs, "\n") + "\n"); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Add records dir as locked. It is a no-op when dir is already listed.
func Add(dir string) error {
	state, err := stateDir()
	if err != nil {
		return err
	}
	for _, d := range []string{state, filepath.Join(state, "answers")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	dirs, err := Locked()
	if err != nil {
		return err
	}
	for _, d := range dirs {
		if d == dir {
			return nil
		}
	}
	return write(append(dirs, dir))
}

// Remove drops dir from the locked list. The file is deleted when it
// would be left empty.
func Remove(dir string) error {
	dirs, err := Locked()
	if err != nil {
		return err
	}
	kept := dirs[:0]
	for _, d := range dirs {
		if d != dir {
			kept = append(kept, d)
		}
	}
	if len(kept) == len(dirs) {
		return nil
	}
	return write(kept)
}

// Canonical returns dir as an absolute path with symlinks resolved, the
// form stored in the locks file.
func Canonical(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %s: %w", dir, err)
	}
	return real, nil
}
