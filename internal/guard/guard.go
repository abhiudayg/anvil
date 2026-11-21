// Package guard decides whether an agent tool call may run while a
// workspace is locked. Pure logic: Check is called by anvil guard only
// when the locks file is non-empty.
package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const lockedNote = "Only the learner can unlock, with anvil unlock in their own terminal."

func blocked(format string, args ...any) error {
	return fmt.Errorf("anvil guard: a workspace is locked; %s. %s", fmt.Sprintf(format, args...), lockedNote)
}

var allowTools = map[string]bool{
	"read": true, "Read": true,
	"grep": true, "Grep": true,
	"glob": true, "Glob": true,
	"LS":            true,
	"notebook_read": true, "NotebookRead": true,
	"webfetch": true, "WebFetch": true,
	"web_search": true, "WebSearch": true,
	"todo_write": true, "TodoWrite": true,
	"ask_user_question": true, "AskUserQuestion": true,
	"get_output": true, "BashOutput": true,
	"kill_shell": true, "KillShell": true, "KillBash": true,
	"read_subagent": true,
	"skill":         true, "Skill": true,
	"exit_plan_mode": true, "ExitPlanMode": true,
	"mcp_list_servers":  true,
	"mcp_list_tools":    true,
	"mcp_read_resource": true,
	"read_page_content": true,
}

var writeTools = map[string]bool{
	"edit": true, "write": true,
	"Edit": true, "Write": true, "MultiEdit": true,
	"NotebookEdit": true, "notebook_edit": true,
}

var envNames = map[string]bool{"ANVIL_KEYS": true, "ANVIL_TEST_PG_DSN": true}
var assignment = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=`)

// Check returns nil when the tool call is allowed while a workspace is
// locked, or an error whose text is shown to the agent.
func Check(tool string, input map[string]any, answersDir string) error {
	if allowTools[tool] {
		return nil
	}
	switch tool {
	case "run_subagent":
		if input["profile"] == "tutor" {
			return nil
		}
		return blocked("only the tutor subagent is allowed")
	case "Task":
		if input["subagent_type"] == "tutor" {
			return nil
		}
		return blocked("only the tutor subagent is allowed")
	}
	if writeTools[tool] {
		return checkWrite(input, answersDir)
	}
	if tool == "exec" || tool == "Bash" {
		return checkExec(input)
	}
	return blocked("%q is not allowed", tool)
}

func checkWrite(input map[string]any, answersDir string) error {
	target, _ := input["file_path"].(string)
	if target == "" {
		target, _ = input["notebook_path"].(string)
	}
	if target == "" {
		return blocked("write calls need a file_path")
	}
	if !filepath.IsAbs(target) {
		return blocked("write target %q is not an absolute path; answers go under the answers directory", target)
	}
	answers, err := filepath.EvalSymlinks(answersDir)
	if err != nil {
		return blocked("cannot resolve the answers directory: %v", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(filepath.Clean(target)))
	if err != nil {
		return blocked("cannot resolve %q: %v", target, err)
	}
	if parent != answers && !strings.HasPrefix(parent, answers+string(os.PathSeparator)) {
		return blocked("write target %q is outside the answers directory", target)
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return blocked("write target %q is a symlink", target)
	}
	return nil
}

func checkExec(input map[string]any) error {
	if env, ok := input["env"]; ok {
		obj, ok := env.(map[string]any)
		if !ok {
			return blocked("env must be an object")
		}
		for k := range obj {
			if !envNames[k] {
				return blocked("environment variable %q is not allowed; only ANVIL_KEYS and ANVIL_TEST_PG_DSN are", k)
			}
		}
	}
	switch flavor, _ := input["shell_flavor"].(string); flavor {
	case "", "bash", "zsh":
	default:
		return blocked("shell_flavor %q is not allowed", flavor)
	}
	cmd, _ := input["command"].(string)
	if cmd == "" {
		return blocked("the command is empty")
	}
	return checkCommand(cmd)
}

// checkCommand parses a shell command into &&-separated segments and
// checks each against a small allowlist of programs.
func checkCommand(cmd string) error {
	if strings.ContainsAny(cmd, "\n\r`\\$") {
		return blocked("the command contains a forbidden character")
	}
	segments, err := splitSegments(cmd)
	if err != nil {
		return err
	}
	for _, words := range segments {
		if err := checkSegment(words); err != nil {
			return err
		}
	}
	return nil
}

// splitSegments tokenizes on whitespace outside quotes and splits
// segments on unquoted &&. Single and double quotes are literal.
func splitSegments(cmd string) ([][]string, error) {
	var segments [][]string
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune
	runes := []rune(cmd)
	flush := func() {
		if inWord {
			words = append(words, word.String())
			word.Reset()
			inWord = false
		}
	}
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			} else {
				word.WriteRune(c)
			}
			continue
		}
		switch {
		case c == '\'' || c == '"':
			quote = c
			inWord = true
		case c == ' ' || c == '\t':
			flush()
		case c == '&':
			if i+1 < len(runes) && runes[i+1] == '&' {
				flush()
				segments = append(segments, words)
				words = nil
				i++
			} else {
				return nil, blocked("the command contains a forbidden operator")
			}
		case strings.ContainsRune(";|<>(){}*?[]", c):
			return nil, blocked("the command contains a forbidden operator")
		default:
			word.WriteRune(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, blocked("the command has an unterminated quote")
	}
	flush()
	segments = append(segments, words)
	return segments, nil
}

func checkSegment(words []string) error {
	for len(words) > 0 {
		m := assignment.FindStringSubmatch(words[0])
		if m == nil {
			break
		}
		if !envNames[m[1]] {
			return blocked("environment variable %q is not allowed; only ANVIL_KEYS and ANVIL_TEST_PG_DSN are", m[1])
		}
		words = words[1:]
	}
	if len(words) == 0 {
		return blocked("the command is empty")
	}
	switch words[0] {
	case "pwd", "ls", "cat", "head", "tail", "wc", "echo":
		return nil
	case "cd":
		if len(words) > 2 {
			return blocked("cd takes at most one argument")
		}
		return nil
	case "git":
		return checkGit(words[1:])
	case "go":
		return checkGo(words[1:])
	case "anvil":
		return checkAnvil(words[1:])
	default:
		return blocked("%q is not allowed", words[0])
	}
}

func checkGit(args []string) error {
	if len(args) >= 2 && args[0] == "-C" {
		args = args[2:]
	}
	if len(args) == 0 {
		return blocked("git needs a subcommand")
	}
	switch args[0] {
	case "status", "diff", "log", "show":
	default:
		return blocked("git %q is not allowed", args[0])
	}
	for _, a := range args[1:] {
		if a == "-c" || strings.HasPrefix(a, "--output") || strings.HasPrefix(a, "--ext-diff") {
			return blocked("git option %q is not allowed", a)
		}
	}
	return nil
}

var goFlags = map[string]bool{
	"run": true, "race": true, "count": true, "v": true, "timeout": true,
	"short": true, "bench": true, "benchmem": true, "benchtime": true,
	"cover": true, "failfast": true, "cpu": true, "parallel": true,
	"tags": true, "list": true, "shuffle": true, "json": true, "vet": true,
}

var goValueFlags = map[string]bool{
	"run": true, "count": true, "timeout": true, "bench": true,
	"benchtime": true, "cpu": true, "parallel": true, "tags": true,
	"list": true, "shuffle": true, "vet": true,
}

func checkGo(args []string) error {
	if len(args) == 0 {
		return blocked("go needs a subcommand")
	}
	switch args[0] {
	case "test", "vet", "version":
	default:
		return blocked("go %q is not allowed", args[0])
	}
	for i := 1; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name := strings.TrimLeft(a, "-")
		hasValue := false
		if j := strings.Index(name, "="); j >= 0 {
			name = name[:j]
			hasValue = true
		}
		if !goFlags[name] {
			return blocked("go option %q is not allowed", a)
		}
		if !hasValue && goValueFlags[name] {
			i++ // the next word is the flag's value
		}
	}
	return nil
}

func checkAnvil(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "grade", "review", "path", "projects", "topics", "help", "check-keys", "lock":
		return nil
	default:
		return blocked("anvil %q is not allowed", args[0])
	}
}
