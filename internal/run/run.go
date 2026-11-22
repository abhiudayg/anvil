// Package run tracks an active learner run in ~/.anvil/run.json: the keys
// repository in use and each workspace's stage, status and build commit.
// The guard hook reads it; the CLI writes it.
package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Stage is one workspace's place in the run.
type Stage struct {
	Project     string `json:"project"`
	Number      int    `json:"number"`
	Status      string `json:"status"` // "locked", "unlocked", "fixed"
	BuildCommit string `json:"build_commit"`
	Attempts    int    `json:"attempts"` // graded answers that were not perfect
	Revealed    bool   `json:"revealed"` // the reference answer was shown
}

// State is the whole run: which keys repo, and a Stage per workspace.
type State struct {
	Keys       string           `json:"keys"`       // canonical keys repo dir
	Workspaces map[string]Stage `json:"workspaces"` // key: canonical workspace dir
}

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".anvil"), nil
}

// File returns the run state path, ~/.anvil/run.json.
func File() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "run.json"), nil
}

// AnswersDir returns the directory agents may write into, ~/.anvil/answers.
func AnswersDir() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "answers"), nil
}

// Load reads the run state. ok is false (with no error) when no run exists.
func Load() (State, bool, error) {
	path, err := File()
	if err != nil {
		return State{}, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, false, err
	}
	if s.Workspaces == nil {
		s.Workspaces = map[string]Stage{}
	}
	return s, true, nil
}

// Save writes the run state atomically at mode 0600.
func Save(s State) error {
	path, err := File()
	if err != nil {
		return err
	}
	for _, d := range []string{filepath.Dir(path), filepath.Join(filepath.Dir(path), "answers")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "run-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
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

// Delete removes the run state; a missing file is fine.
func Delete() error {
	path, err := File()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Canonical returns dir as an absolute path with symlinks resolved, the
// form stored in the run state.
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
