// Package tui is the terminal interface to an Anvil run: a chat with the
// lead agent plus every anvil subcommand, in one screen.
package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// leadPrompt points the chat agent at the rulebook it must follow.
const leadPrompt = `You are the lead agent for an Anvil learner run, talking to the learner. Read AGENTS.md in this repository and follow it exactly: teach each stage's concepts before running anvil build, never reveal or hint at the root cause beyond the stage's hints, give hints one at a time in order and only when asked, ask the bug question as written, grade answers only with anvil grade and report grades as point numbers, never write workspace code or the fix. The learner speaks to you in this chat; answer briefly and directly.`

// allowedTools pre-approves what the lead agent needs so headless Claude
// never pauses on a permission prompt. During a run the anvil guard hook
// still filters every call against the stricter run policy.
const allowedTools = "Bash Read Grep Glob LS Write Edit MultiEdit NotebookRead NotebookEdit WebFetch WebSearch TodoWrite Task TaskOutput TaskStop Skill AskUserQuestion BashOutput KillShell ExitPlanMode"

// Chat is a multi-turn conversation with an agent CLI in headless mode.
// Supported dialects: claude (Claude Code -p mode) and gemini (Gemini CLI
// headless). It runs in Dir, the anvil repository, so project hook files
// and AGENTS.md load; the guard applies to hooked agents while a run is
// active.
type Chat struct {
	Command string        // agent binary; defaults to "claude"
	Dir     string        // anvil repository directory
	Model   string        // empty uses the CLI's default
	Timeout time.Duration // per message; defaults to 10 minutes

	session string // provider session id, once known
	sent    bool   // a message was already sent (Gemini --resume latest)
}

// provider selects the CLI dialect from the binary name.
func provider(command string) string {
	if strings.Contains(strings.ToLower(filepath.Base(command)), "gemini") {
		return "gemini"
	}
	return "claude"
}

// Send delivers one learner message and returns the agent's reply. The
// first call starts a session; later calls resume it so context persists.
func (c *Chat) Send(ctx context.Context, text string) (string, error) {
	command := c.Command
	if command == "" {
		command = "claude"
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch provider(command) {
	case "gemini":
		args := []string{"--output-format", "json", "--yolo"}
		if c.Model != "" {
			args = append(args, "--model", c.Model)
		}
		if c.session != "" {
			args = append(args, "--resume", c.session)
		} else if c.sent {
			// The session id never reached us (older CLI); the just-saved
			// session is the latest one for this directory. --resume goes
			// last so the parser can't take the prompt as its value.
			args = append(args, "--resume")
		}
		prompt := text
		if !c.sent {
			prompt = leadPrompt + "\n\n---\n\n" + text
		}
		args = append([]string{prompt}, args...)
		cmd = exec.CommandContext(ctx, command, args...)
	default:
		args := []string{"-p", "--output-format", "json",
			"--strict-mcp-config", "--disable-slash-commands",
			"--allowedTools", allowedTools,
			"--append-system-prompt", leadPrompt}
		if c.Model != "" {
			args = append(args, "--model", c.Model)
		}
		if c.session != "" {
			args = append(args, "--resume", c.session)
		}
		cmd = exec.CommandContext(ctx, command, args...)
		cmd.Stdin = strings.NewReader(text)
	}
	cmd.Dir = c.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()

	var reply struct {
		Result    string `json:"result"`   // claude
		Response  string `json:"response"` // gemini
		SessionID string `json:"session_id"`
		IsError   bool   `json:"is_error"`
		Error     *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	detail := strings.TrimSpace(stderr.String())
	if json.Unmarshal(stdout.Bytes(), &reply) == nil {
		if reply.SessionID != "" {
			c.session = reply.SessionID
		}
		for _, s := range []string{reply.Result, reply.Response} {
			if s != "" {
				detail = s
			}
		}
		if reply.Error != nil && reply.Error.Message != "" {
			detail = reply.Error.Message
		}
	}
	if len(detail) > 1000 {
		detail = detail[:1000]
	}
	if runErr != nil {
		if detail == "" {
			detail = runErr.Error()
		}
		return "", errors.New("agent: " + detail)
	}
	c.sent = true
	if reply.IsError || reply.Error != nil {
		return "", errors.New("agent: " + detail)
	}
	if detail == "" {
		return "", errors.New("agent: empty reply")
	}
	return detail, nil
}

// RunFunc executes one anvil subcommand in-process and returns its output.
type RunFunc func(args []string) (output string, code int)
