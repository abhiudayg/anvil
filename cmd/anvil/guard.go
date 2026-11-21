package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"anvil/internal/guard"
	"anvil/internal/lockstate"
)

// stdin is the hook payload source; tests swap it.
var stdin io.Reader = os.Stdin

// runGuard is the PreToolUse hook entry point. It blocks agent tool calls
// while any workspace is locked: exit 2 with a JSON decision means block,
// exit 0 means allow.
func runGuard(stdout, stderr io.Writer) int {
	locked, err := lockstate.Locked()
	if err == nil && len(locked) == 0 {
		return 0
	}
	reason := ""
	if err != nil {
		reason = fmt.Sprintf("cannot read the locks file: %v", err)
	} else {
		payload, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
		var hook struct {
			Tool  string         `json:"tool_name"`
			Input map[string]any `json:"tool_input"`
		}
		if err != nil || json.Unmarshal(payload, &hook) != nil {
			reason = "the tool call payload could not be parsed"
		} else {
			answers, aerr := lockstate.AnswersDir()
			if aerr != nil {
				reason = fmt.Sprintf("cannot find the answers directory: %v", aerr)
			} else if gerr := guard.Check(hook.Tool, hook.Input, answers); gerr != nil {
				out, _ := json.Marshal(map[string]string{"decision": "block", "reason": gerr.Error()})
				fmt.Fprintln(stdout, string(out))
				fmt.Fprintln(stderr, gerr.Error())
				return 2
			}
		}
	}
	if reason == "" {
		return 0
	}
	reason = "anvil guard: a workspace is locked; " + reason
	out, _ := json.Marshal(map[string]string{"decision": "block", "reason": reason})
	fmt.Fprintln(stdout, string(out))
	fmt.Fprintln(stderr, reason)
	return 2
}
