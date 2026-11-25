// Package tui is the terminal interface to an Anvil run: a chat with the
// lead agent plus every anvil subcommand, in one screen.
package tui

import (
	"bufio"
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
// Supported dialects: claude (Claude Code -p stdin mode), gemini (Gemini
// CLI headless) and agy (-p prompt-value mode). It runs in Dir, the anvil
// repository, so project hook files and AGENTS.md load; the guard applies
// to hooked agents while a run is active.
type Chat struct {
	Command string        // agent binary; defaults to "claude"
	Dir     string        // anvil repository directory
	Model   string        // empty uses the CLI's default
	Timeout time.Duration // per message; defaults to 10 minutes

	session string // provider session id, once known
	sent    bool   // a message was already sent (Gemini --resume latest)
}

type StreamEvent struct {
	Delta string
	Err   error
}

// provider selects the CLI dialect from the binary name.
func provider(command string) string {
	base := strings.ToLower(filepath.Base(command))
	switch {
	case strings.Contains(base, "gemini"):
		return "gemini"
	case strings.Contains(base, "agy"):
		return "agy"
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
	case "agy":
		args := []string{"-p", c.promptText(text), "--output-format", "json",
			"--dangerously-skip-permissions", "--disable-slash-commands"}
		if c.Model != "" {
			args = append(args, "--model", c.Model)
		}
		if c.session != "" {
			args = append(args, "--conversation", c.session)
		} else if c.sent {
			args = append(args, "-c")
		}
		cmd = exec.CommandContext(ctx, command, args...)
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
		args = append([]string{c.promptText(text)}, args...)
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
		Result         string `json:"result"`          // claude
		Response       string `json:"response"`        // gemini, agy
		SessionID      string `json:"session_id"`      // claude, gemini
		ConversationID string `json:"conversation_id"` // agy
		Status         string `json:"status"`          // agy: "SUCCESS" on ok
		IsError        bool   `json:"is_error"`
		Error          *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	detail := strings.TrimSpace(stderr.String())
	if json.Unmarshal(stdout.Bytes(), &reply) == nil {
		if reply.SessionID != "" {
			c.session = reply.SessionID
		} else if reply.ConversationID != "" {
			c.session = reply.ConversationID
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
	if reply.IsError || reply.Error != nil || (reply.Status != "" && reply.Status != "SUCCESS") {
		return "", errors.New("agent: " + detail)
	}
	if detail == "" {
		return "", errors.New("agent: empty reply")
	}
	return detail, nil
}

func (c *Chat) promptText(text string) string {
	if c.sent {
		return text
	}
	return leadPrompt + "\n\n---\n\n" + text
}

type streamResult struct {
	session string
	errText string
}

func parseClaudeStream(line []byte) (delta string, res *streamResult) {
	var ev struct {
		Type  string `json:"type"`
		Event struct {
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		} `json:"event"`
		SessionID string `json:"session_id"`
		IsError   bool   `json:"is_error"`
		Result    string `json:"result"`
	}
	if json.Unmarshal(line, &ev) != nil {
		return "", nil
	}
	if ev.Type == "stream_event" && ev.Event.Delta.Text != "" {
		return ev.Event.Delta.Text, nil
	}
	if ev.Type == "result" {
		res := &streamResult{session: ev.SessionID}
		if ev.IsError {
			res.errText = ev.Result
			if res.errText == "" {
				res.errText = "turn failed"
			}
		}
		return "", res
	}
	return "", nil
}

func parseAgyStream(line []byte) (delta string, res *streamResult) {
	var ev struct {
		Event      string `json:"event"`
		StepUpdate struct {
			StepType  string `json:"step_type"`
			TextDelta string `json:"text_delta"`
		} `json:"step_update"`
		Result struct {
			ConversationID string `json:"conversation_id"`
			Status         string `json:"status"`
			Response       string `json:"response"`
		} `json:"result"`
	}
	if json.Unmarshal(line, &ev) != nil {
		return "", nil
	}
	if ev.Event == "step_update" && ev.StepUpdate.StepType == "agent_response" && ev.StepUpdate.TextDelta != "" {
		return ev.StepUpdate.TextDelta, nil
	}
	if ev.Event == "result" {
		res := &streamResult{session: ev.Result.ConversationID}
		if ev.Result.Status != "" && ev.Result.Status != "SUCCESS" {
			res.errText = ev.Result.Response
			if res.errText == "" {
				res.errText = "status " + ev.Result.Status
			}
		}
		return "", res
	}
	return "", nil
}

func (c *Chat) streamCmd(ctx context.Context, text string) (*exec.Cmd, func([]byte) (string, *streamResult)) {
	command := c.Command
	switch provider(command) {
	case "agy":
		args := []string{"-p", c.promptText(text), "--output-format", "stream-json",
			"--dangerously-skip-permissions", "--disable-slash-commands"}
		if c.Model != "" {
			args = append(args, "--model", c.Model)
		}
		if c.session != "" {
			args = append(args, "--conversation", c.session)
		} else if c.sent {
			args = append(args, "-c")
		}
		return exec.CommandContext(ctx, command, args...), parseAgyStream
	default:
		args := []string{"-p", "--output-format", "stream-json", "--verbose",
			"--include-partial-messages", "--strict-mcp-config",
			"--disable-slash-commands", "--allowedTools", allowedTools,
			"--append-system-prompt", leadPrompt}
		if c.Model != "" {
			args = append(args, "--model", c.Model)
		}
		if c.session != "" {
			args = append(args, "--resume", c.session)
		}
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Stdin = strings.NewReader(text)
		return cmd, parseClaudeStream
	}
}

func (c *Chat) SendStream(ctx context.Context, text string) <-chan StreamEvent {
	out := make(chan StreamEvent, 64)
	go c.stream(ctx, text, out)
	return out
}

func (c *Chat) stream(ctx context.Context, text string, out chan<- StreamEvent) {
	defer close(out)
	if provider(c.Command) == "gemini" {
		reply, err := c.Send(ctx, text)
		if err != nil {
			out <- StreamEvent{Err: err}
			return
		}
		out <- StreamEvent{Delta: reply}
		return
	}
	cmd, parse := c.streamCmd(ctx, text)
	cmd.Dir = c.Dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		out <- StreamEvent{Err: errors.New("agent: " + err.Error())}
		return
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		out <- StreamEvent{Err: errors.New("agent: " + err.Error())}
		return
	}
	r := bufio.NewReader(stdout)
	var failed error
	for {
		line, err := r.ReadBytes('\n')
		if len(line) != 0 {
			if delta, res := parse(line); delta != "" {
				out <- StreamEvent{Delta: delta}
			} else if res != nil {
				if res.session != "" {
					c.session = res.session
				}
				if res.errText != "" {
					failed = errors.New("agent: " + res.errText)
				}
			}
		}
		if err != nil {
			break
		}
	}
	runErr := cmd.Wait()
	c.sent = true
	if failed == nil && runErr != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = runErr.Error()
		}
		failed = errors.New("agent: " + detail)
	}
	if failed != nil {
		out <- StreamEvent{Err: failed}
	}
}

// RunFunc executes one anvil subcommand in-process and returns its output.
type RunFunc func(args []string) (output string, code int)
