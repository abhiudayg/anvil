package main

import (
	"encoding/json"
	"strings"
	"testing"

	"anvil/internal/lockstate"
)

func feedStdin(t *testing.T, payload string) {
	t.Helper()
	old := stdin
	stdin = strings.NewReader(payload)
	t.Cleanup(func() { stdin = old })
}

func TestGuardNoLocksAllowsGarbage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	feedStdin(t, "not json")
	code, out, _ := runCLI("guard")
	if code != 0 || out != "" {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}

func TestGuardBlocksShellWrites(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := lockstate.Add("/w"); err != nil {
		t.Fatal(err)
	}
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

func TestGuardAllowsGoTest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := lockstate.Add("/w"); err != nil {
		t.Fatal(err)
	}
	feedStdin(t, `{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}`)
	code, out, _ := runCLI("guard")
	if code != 0 || out != "" {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}

func TestGuardBlocksGarbageWhenLocked(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := lockstate.Add("/w"); err != nil {
		t.Fatal(err)
	}
	feedStdin(t, "not json")
	code, out, _ := runCLI("guard")
	if code != 2 || !strings.Contains(out, `"decision":"block"`) {
		t.Errorf("exit = %d, out = %q", code, out)
	}
}
