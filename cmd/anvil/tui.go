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
		tui.New(chat, runner, tuiStatus),
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

// tuiStatus renders the header line: whether a run is active and where
// each workspace stands.
func tuiStatus() string {
	state, ok, err := run.Load()
	if err != nil || !ok {
		return "anvil · no run"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "anvil · run active · keys %s", state.Keys)
	if len(state.Workspaces) == 0 {
		b.WriteString(" · no workspaces")
	}
	var dirs []string
	for ws := range state.Workspaces {
		dirs = append(dirs, ws)
	}
	sort.Strings(dirs)
	for _, ws := range dirs {
		st := state.Workspaces[ws]
		fmt.Fprintf(&b, " · %s %s/%d %s", filepath.Base(ws), st.Project, st.Number, st.Status)
	}
	return b.String()
}
