package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"anvil/internal/guard"
	"anvil/internal/run"
)

// stdin is the hook payload source; tests swap it.
var stdin io.Reader = os.Stdin

// cursorTool maps Cursor's tool names to the names the guard knows.
var cursorTool = map[string]string{
	"Shell": "Bash", "Read": "Read", "Grep": "Grep",
	"Write": "Write", "Delete": "Write", "Task": "Task",
}

// runGuard is the PreToolUse hook entry point. While a run is active it
// decides whether each agent tool call may run: Claude format blocks with
// exit 2 and a JSON decision, Cursor format with a JSON permission.
func runGuard(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("guard", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "claude", "hook payload format: claude or cursor")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	state, ok, err := run.Load()
	if err == nil && !ok {
		if *format == "cursor" {
			fmt.Fprintln(stdout, `{"permission":"allow"}`)
		}
		return 0
	}
	block := func(reason string) int {
		if *format == "cursor" {
			out, _ := json.Marshal(map[string]string{
				"permission": "deny", "user_message": reason, "agent_message": reason})
			fmt.Fprintln(stdout, string(out))
		} else {
			out, _ := json.Marshal(map[string]string{"decision": "block", "reason": reason})
			fmt.Fprintln(stdout, string(out))
		}
		fmt.Fprintln(stderr, reason)
		return 2
	}
	if err != nil {
		return block("anvil guard: cannot read the run state: " + err.Error())
	}
	payload, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
	var hook struct {
		Tool  string          `json:"tool_name"`
		Input json.RawMessage `json:"tool_input"`
		Cwd   string          `json:"cwd"`
	}
	if err != nil || json.Unmarshal(payload, &hook) != nil {
		return block("anvil guard: the tool call payload could not be parsed. " + "An Anvil run is active; only the learner can unlock or stop it, in their own terminal.")
	}
	var input map[string]any
	if len(hook.Input) > 0 {
		if err := json.Unmarshal(hook.Input, &input); err != nil {
			var text string
			if err := json.Unmarshal(hook.Input, &text); err != nil || json.Unmarshal([]byte(text), &input) != nil {
				return block("anvil guard: the tool input could not be parsed. An Anvil run is active; only the learner can unlock or stop it, in their own terminal.")
			}
		}
	}
	tool := hook.Tool
	if *format == "cursor" {
		if strings.HasPrefix(tool, "MCP:") {
			return block("anvil guard: " + tool + " is not allowed. An Anvil run is active; only the learner can unlock or stop it, in their own terminal.")
		}
		if mapped, ok := cursorTool[tool]; ok {
			tool = mapped
		}
	}
	answers, err := run.AnswersDir()
	if err != nil {
		return block("anvil guard: cannot find the answers directory: " + err.Error())
	}
	if err := os.MkdirAll(answers, 0o700); err != nil {
		return block("anvil guard: cannot create the answers directory: " + err.Error())
	}
	canonical, err := run.Canonical(answers)
	if err != nil {
		return block("anvil guard: cannot resolve the answers directory: " + err.Error())
	}
	if err := guard.Check(guard.Call{Tool: tool, Input: input, Cwd: hook.Cwd}, guard.Policy{
		Keys: state.Keys, Answers: canonical,
	}); err != nil {
		return block(err.Error())
	}
	if *format == "cursor" {
		fmt.Fprintln(stdout, `{"permission":"allow"}`)
	}
	return 0
}
