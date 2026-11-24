package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func enter(m Model, text string) (Model, tea.Cmd) {
	m.input.SetValue(text)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model), cmd
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
	}, func() string { return "header" })
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
