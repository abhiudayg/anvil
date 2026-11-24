package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// chatStub writes a fake agent binary executing script body. Its argv are
// logged to args.log in the same directory for later inspection.
func chatStub(t *testing.T, script string) *Chat {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-stub")
	body := "#!/bin/sh\n" +
		`{ printf '%s' "$*" | tr '\n' ' '; echo; } >> "` + filepath.Join(dir, "args.log") + `"` + "\n" +
		script
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Chat{Command: path, Dir: dir}
}

func snapStub(header string, ws ...Workspace) SnapshotFunc {
	return func() Snapshot { return Snapshot{Header: header, Workspaces: ws} }
}

func key(m Model, k string) (Model, tea.Cmd) {
	types := map[string]tea.KeyType{
		"tab": tea.KeyTab, "esc": tea.KeyEsc, "enter": tea.KeyEnter,
		"f1": tea.KeyF1, "ctrl+c": tea.KeyCtrlC, "ctrl+d": tea.KeyCtrlD,
	}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	if t, ok := types[k]; ok {
		msg = tea.KeyMsg{Type: t}
	}
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func enter(m Model, text string) (Model, tea.Cmd) {
	m.input.SetValue(text)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model), cmd
}

func sized(m Model) Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	return next.(Model)
}

func TestChatResumesSession(t *testing.T) {
	c := chatStub(t, `cat >/dev/null
printf '%s\n' '{"result":"reply","session_id":"sess-1","is_error":false}'`)
	out, err := c.Send(context.Background(), "first")
	if err != nil || out != "reply" {
		t.Fatalf("Send = %q, %v", out, err)
	}
	if _, err := c.Send(context.Background(), "second"); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(filepath.Join(c.Dir, "args.log"))
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(calls) != 2 {
		t.Fatalf("calls = %v", calls)
	}
	if strings.Contains(calls[0], "--resume") {
		t.Errorf("first call must not resume: %s", calls[0])
	}
	if !strings.Contains(calls[1], "--resume sess-1") {
		t.Errorf("second call must resume sess-1: %s", calls[1])
	}
}

// geminiStub is chatStub named so provider() selects the gemini dialect.
func geminiStub(t *testing.T, script string) *Chat {
	t.Helper()
	c := chatStub(t, script)
	dir := c.Dir
	gemini := filepath.Join(dir, "gemini-stub")
	if err := os.Rename(c.Command, gemini); err != nil {
		t.Fatal(err)
	}
	c.Command = gemini
	return c
}

func TestGeminiSendsPositionalPromptAndResumes(t *testing.T) {
	c := geminiStub(t, `printf '%s\n' '{"response":"gem hi","session_id":"g-9"}'`)
	out, err := c.Send(context.Background(), "first")
	if err != nil || out != "gem hi" {
		t.Fatalf("Send = %q, %v", out, err)
	}
	if _, err := c.Send(context.Background(), "second"); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(filepath.Join(c.Dir, "args.log"))
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(calls) != 2 {
		t.Fatalf("calls = %v", calls)
	}
	if !strings.Contains(calls[0], "--output-format json") || !strings.Contains(calls[0], "--yolo") {
		t.Errorf("gemini call missing headless flags: %s", calls[0])
	}
	if !strings.HasPrefix(calls[0], "You are the lead agent") {
		t.Errorf("first gemini call must carry the lead prompt as positional arg: %s", calls[0])
	}
	if !strings.Contains(calls[1], "--resume g-9") {
		t.Errorf("second gemini call must resume g-9: %s", calls[1])
	}
}

func TestGeminiWithoutSessionIDResumesLatest(t *testing.T) {
	c := geminiStub(t, `printf '%s\n' '{"response":"hi"}'`)
	if _, err := c.Send(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Send(context.Background(), "two"); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(filepath.Join(c.Dir, "args.log"))
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(log)), "\n")
	if !strings.HasSuffix(calls[1], "--resume") {
		t.Errorf("second call should end with bare --resume (latest): %s", calls[1])
	}
}

func TestChatReportsAgentError(t *testing.T) {
	c := chatStub(t, `cat >/dev/null
printf '%s\n' '{"result":"Not logged in","is_error":true}'
exit 1`)
	_, err := c.Send(context.Background(), "hi")
	if err == nil || !strings.Contains(err.Error(), "Not logged in") {
		t.Fatalf("error = %v, want one containing the CLI's error text", err)
	}
}

func TestModelRunsSlashCommand(t *testing.T) {
	m := New(nil, func(args []string) (string, int) {
		return "out:" + strings.Join(args, " "), 0
	}, snapStub("header"))
	m, cmd := enter(m, "/check /tmp/ws")
	if !m.busy || cmd == nil {
		t.Fatal("slash command should run asynchronously")
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	if m.busy {
		t.Error("still busy after command finished")
	}
	if !strings.Contains(m.log.String(), "out:check /tmp/ws") {
		t.Errorf("transcript missing command output:\n%s", m.log.String())
	}
}

func TestModelChatsWithAgent(t *testing.T) {
	c := chatStub(t, `cat >/dev/null
printf '%s\n' '{"result":"hi learner","session_id":"s","is_error":false}'`)
	m := New(c, func([]string) (string, int) { return "", 0 }, nil)
	m, cmd := enter(m, "explain slices")
	if !m.busy || cmd == nil {
		t.Fatal("chat should run asynchronously")
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	if !strings.Contains(m.log.String(), "hi learner") {
		t.Errorf("transcript missing agent reply:\n%s", m.log.String())
	}
}

func TestModelHelpIsLocal(t *testing.T) {
	m := New(nil, func([]string) (string, int) { return "", 0 }, nil)
	m, cmd := enter(m, "/help")
	if cmd != nil {
		t.Error("/help must not spawn work")
	}
	if !strings.Contains(m.log.String(), "/check") {
		t.Errorf("transcript missing help text:\n%s", m.log.String())
	}
}

func TestTabSwitchesToWorkspacesAndEscReturns(t *testing.T) {
	m := sized(New(nil, func([]string) (string, int) { return "", 0 },
		snapStub("hdr", Workspace{Dir: "/tmp/redis-ws", Project: "redis", Stage: 1, Status: "locked"})))
	next, _ := m.Update(snapshotMsg(m.snapFn()))
	m = next.(Model)
	m, _ = key(m, "tab")
	if m.pane != paneWorkspaces {
		t.Fatalf("pane = %v, want workspaces", m.pane)
	}
	if v := m.View(); !strings.Contains(v, "PROJECT") || !strings.Contains(v, "redis") {
		t.Errorf("workspaces view missing table data:\n%s", v)
	}
	m, _ = key(m, "esc")
	if m.pane != paneChat {
		t.Fatalf("pane = %v, want chat", m.pane)
	}
}

func TestSnapshotFeedsHeaderAndTable(t *testing.T) {
	m := sized(New(nil, func([]string) (string, int) { return "", 0 }, nil))
	next, _ := m.Update(snapshotMsg(Snapshot{
		Header:     "anvil · run active",
		Workspaces: []Workspace{{Dir: "/x", Project: "jobq", Stage: 3, Status: "fixed"}},
	}))
	m = next.(Model)
	if m.header != "anvil · run active" {
		t.Errorf("header = %q", m.header)
	}
	m, _ = key(m, "tab")
	if !strings.Contains(m.View(), "jobq") {
		t.Errorf("table missing workspace row:\n%s", m.View())
	}
}

func TestTickRearmsAndRefreshes(t *testing.T) {
	orig := refreshInterval
	refreshInterval = time.Millisecond
	defer func() { refreshInterval = orig }()
	calls := 0
	m := New(nil, func([]string) (string, int) { return "", 0 },
		func() Snapshot { calls++; return Snapshot{Header: "h"} })
	_, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Fatal("tick must produce refresh + re-arm commands")
	}
	var kinds []string
	for _, c := range cmd().(tea.BatchMsg) {
		switch c().(type) {
		case snapshotMsg:
			kinds = append(kinds, "snapshot")
		case tickMsg:
			kinds = append(kinds, "tick")
		}
	}
	if strings.Join(kinds, ",") != "snapshot,tick" {
		t.Fatalf("tick must emit a refresh then re-arm; got %v", kinds)
	}
	if calls != 1 {
		t.Errorf("SnapshotFunc calls = %d, want 1", calls)
	}
}

func TestQuitWorksFromEveryState(t *testing.T) {
	m := sized(New(nil, func([]string) (string, int) { return "", 0 }, nil))
	for _, setup := range []func(Model) Model{
		func(m Model) Model { return m },
		func(m Model) Model { m.pane = paneWorkspaces; return m },
		func(m Model) Model { m.help = true; return m },
	} {
		state := setup(m)
		_, cmd := key(state, "ctrl+c")
		if cmd == nil {
			t.Fatal("ctrl+c produced no command")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("ctrl+c in pane=%d help=%v did not quit", state.pane, state.help)
		}
	}
}

func TestSendWithoutChatReportsError(t *testing.T) {
	m := New(nil, func([]string) (string, int) { return "", 0 }, nil)
	m, _ = enter(m, "hello")
	if m.busy {
		t.Error("must not stay busy without an agent")
	}
	if !strings.Contains(m.log.String(), "no agent configured") {
		t.Errorf("transcript missing error:\n%s", m.log.String())
	}
}

func TestHelpOverlaySwallowsKeys(t *testing.T) {
	m := sized(New(nil, func([]string) (string, int) { return "", 0 }, nil))
	m, _ = key(m, "f1")
	if !m.help {
		t.Fatal("f1 must open the help overlay")
	}
	if !strings.Contains(m.View(), "commands:") {
		t.Errorf("help overlay missing command list:\n%s", m.View())
	}
	m, _ = key(m, "x")
	if m.input.Value() != "" {
		t.Error("keys must not reach the input while help is open")
	}
	m, _ = key(m, "esc")
	if m.help {
		t.Fatal("esc must close the help overlay")
	}
}

func TestHintsReflectActivePane(t *testing.T) {
	m := sized(New(nil, func([]string) (string, int) { return "", 0 }, nil))
	if v := m.View(); !strings.Contains(v, "tab: workspaces") {
		t.Errorf("chat footer missing pane hint:\n%s", v)
	}
	m, _ = key(m, "tab")
	if v := m.View(); !strings.Contains(v, "esc: back") {
		t.Errorf("workspaces footer missing back hint:\n%s", v)
	}
}

func TestFunctionalTuiPanes(t *testing.T) {
	m := New(nil, func(args []string) (string, int) {
		return "ran:" + strings.Join(args, " "), 0
	}, snapStub("anvil · run active",
		Workspace{Dir: "/tmp/w1", Project: "redis", Stage: 2, Status: "unlocked"}))
	m = sized(m)

	next, _ := m.Update(snapshotMsg(m.snapFn()))
	m = next.(Model)
	if !strings.Contains(m.View(), "anvil · run active") {
		t.Fatalf("header missing snapshot text")
	}

	m, _ = key(m, "tab")
	if v := m.View(); !strings.Contains(v, "redis") || !strings.Contains(v, "unlocked") {
		t.Fatalf("workspaces pane missing row:\n%s", v)
	}
	m, _ = key(m, "tab")
	if m.pane != paneChat {
		t.Fatal("tab must cycle back to chat")
	}

	m, cmd := enter(m, "/projects")
	if cmd != nil {
		next, _ = m.Update(cmd())
		m = next.(Model)
	}
	if !strings.Contains(m.View(), "ran:projects") {
		t.Fatalf("transcript missing dispatch output:\n%s", m.View())
	}

	m, _ = key(m, "f1")
	if !strings.Contains(m.View(), "/unlock") {
		t.Fatalf("help overlay missing commands:\n%s", m.View())
	}
	m, _ = key(m, "f1")
	if m.help {
		t.Fatal("f1 must toggle the overlay closed")
	}
}
