package main

import (
	"encoding/json"
	"strings"
	"testing"

	"anvil/internal/run"
)

func feedStdin(t *testing.T, payload string) {
	t.Helper()
	old := stdin
	stdin = strings.NewReader(payload)
	t.Cleanup(func() { stdin = old })
}

func startRun(t *testing.T) {
	t.Helper()
	if err := run.Save(run.State{Keys: "/nonexistent-keys", Workspaces: map[string]run.Stage{}}); err != nil {
		t.Fatal(err)
	}
}

func TestGuardNoRunAllowsGarbage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	feedStdin(t, "not json")
	code, out, _ := runCLI("guard")
	if code != 0 || out != "" {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}

func TestGuardCursorNoRunPrintsAllow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	feedStdin(t, "not json")
	code, out, _ := runCLI("guard", "--format", "cursor")
	if code != 0 || !strings.Contains(out, `"permission":"allow"`) {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}

func TestGuardBlocksShellWrites(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	startRun(t)
	feedStdin(t, `{"tool_name":"exec","tool_input":{"command":"chmod -R u+w /w"}}`)
	code, out, stderr := runCLI("guard")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	var reply struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(out), &reply); err != nil {
		t.Fatalf("stdout is not JSON: %q", out)
	}
	if reply.Decision != "block" || reply.Reason == "" {
		t.Errorf("reply = %+v", reply)
	}
	if !strings.Contains(stderr, "anvil guard") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestGuardAllowsRead(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	startRun(t)
	feedStdin(t, `{"tool_name":"Read","tool_input":{"file_path":"/w/s01/stage.go"}}`)
	code, out, _ := runCLI("guard")
	if code != 0 || out != "" {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}

func TestGuardCursorDeniesGoTest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	startRun(t)
	feedStdin(t, `{"tool_name":"Shell","tool_input":{"command":"go test ./..."}}`)
	code, out, _ := runCLI("guard", "--format", "cursor")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	var reply map[string]string
	if err := json.Unmarshal([]byte(out), &reply); err != nil {
		t.Fatalf("stdout is not JSON: %q", out)
	}
	if reply["permission"] != "deny" || reply["user_message"] == "" || reply["agent_message"] == "" {
		t.Errorf("reply = %v", reply)
	}
}

func TestGuardCursorAllowsShell(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	startRun(t)
	feedStdin(t, `{"tool_name":"Shell","tool_input":{"command":"anvil fix /w"}}`)
	code, out, _ := runCLI("guard", "--format", "cursor")
	if code != 0 || !strings.Contains(out, `"permission":"allow"`) {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}

func TestGuardBlocksGarbageWhenRunning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	startRun(t)
	feedStdin(t, "not json")
	code, out, _ := runCLI("guard")
	if code != 2 || !strings.Contains(out, `"decision":"block"`) {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}
