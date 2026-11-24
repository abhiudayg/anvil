package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"anvil/internal/run"
	"anvil/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// runTUI opens the terminal interface: a chat with the lead agent plus
// every anvil subcommand as /commands.
func runTUI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "anvil repository directory; the chat agent runs here so hooks and AGENTS.md load")
	agent := fs.String("agent", "claude", "agent CLI used for chat: claude or gemini")
	model := fs.String("model", "", "model passed to the agent CLI (empty: its default)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: anvil tui [--repo DIR] [--agent BIN] [--model M]")
		return 2
	}
	dir, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintf(stderr, "anvil tui: %v\n", err)
		return 2
	}
	if state, ok, _ := run.Load(); ok && state.Keys != "" && !hookedAgent(*agent) {
		fmt.Fprintf(stderr, "anvil tui: warning: %s has no anvil hook configured; during the run it is bound only by AGENTS.md\n", *agent)
	}
	chat := &tui.Chat{Command: *agent, Dir: dir, Model: *model}
	runner := func(argv []string) (string, int) {
		var out, errBuf bytes.Buffer
		code := dispatch(argv, &out, &errBuf)
		return out.String() + errBuf.String(), code
	}
	p := tea.NewProgram(
		tui.New(chat, runner, tuiSnapshot),
		tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "anvil tui: %v\n", err)
		return 2
	}
	return 0
}

// hookedAgent reports whether the chat binary loads an anvil guard hook
// from a project hook file: .claude/settings.json or .gemini/settings.json.
func hookedAgent(command string) bool {
	switch filepath.Base(command) {
	case "claude", "devin":
		return true
	}
	return strings.Contains(strings.ToLower(filepath.Base(command)), "gemini")
}

// tuiSnapshot loads the run state once and shapes it for the screen: the
// header line plus one row per workspace.
func tuiSnapshot() tui.Snapshot {
	state, ok, err := run.Load()
	if err != nil || !ok {
		return tui.Snapshot{Header: "anvil · no run"}
	}
	var dirs []string
	for ws := range state.Workspaces {
		dirs = append(dirs, ws)
	}
	sort.Strings(dirs)
	var b strings.Builder
	fmt.Fprintf(&b, "anvil · run active · keys %s · %d workspaces", state.Keys, len(dirs))
	snap := tui.Snapshot{Header: b.String(), Workspaces: make([]tui.Workspace, 0, len(dirs))}
	for _, ws := range dirs {
		st := state.Workspaces[ws]
		snap.Workspaces = append(snap.Workspaces, tui.Workspace{
			Dir: ws, Project: st.Project, Stage: st.Number, Status: st.Status,
		})
	}
	return snap
}
