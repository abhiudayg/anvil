// Package guard decides whether an agent tool call may run during an
// Anvil run. Pure logic: Check is called by anvil guard only while a run
// is active.
package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const runNote = "An Anvil run is active; only the learner can unlock or stop it, in their own terminal."

func blocked(format string, args ...any) error {
	return fmt.Errorf("anvil guard: %s. %s", fmt.Sprintf(format, args...), runNote)
}

// Policy holds the canonical directories a Check call is evaluated against.
type Policy struct {
	Keys    string // canonical keys repo dir
	Answers string // canonical answers dir
}

// Call is one agent tool call. Cwd is the hook payload's cwd, may be empty.
type Call struct {
	Tool  string
	Input map[string]any
	Cwd   string
}

// resolve evaluates symlinks in path; when path does not exist it resolves
// the nearest existing ancestor and re-joins the rest.
func resolve(path string) (string, error) {
	rest := ""
	p := filepath.Clean(path)
	for {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(real, rest), nil
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", fmt.Errorf("cannot resolve %q", path)
		}
		rest = filepath.Join(filepath.Base(p), rest)
		p = parent
	}
}

// inside reports whether path resolves to dir itself or beneath it.
// dir is already canonical.
func inside(path, dir string) bool {
	real, err := resolve(path)
	if err != nil {
		return false
	}
	return real == dir || strings.HasPrefix(real, dir+string(os.PathSeparator))
}

var readTools = map[string]bool{
	"read": true, "Read": true, "notebook_read": true, "NotebookRead": true,
}

var searchTools = map[string]bool{
	"grep": true, "Grep": true, "glob": true, "Glob": true, "LS": true,
}

var allowTools = map[string]bool{
	"webfetch": true, "WebFetch": true,
	"web_search": true, "WebSearch": true,
	"todo_write": true, "TodoWrite": true,
	"ask_user_question": true, "AskUserQuestion": true,
	"get_output": true, "BashOutput": true,
	"kill_shell": true, "KillShell": true, "KillBash": true,
	"read_subagent": true,
	"skill":         true, "Skill": true,
	"exit_plan_mode": true, "ExitPlanMode": true,
	"mcp_list_servers": true,
	"mcp_list_tools":   true,
}

var writeTools = map[string]bool{
	"edit": true, "write": true,
	"Edit": true, "Write": true, "MultiEdit": true,
	"NotebookEdit": true, "notebook_edit": true,
}

var envNames = map[string]bool{
	"ANVIL_KEYS": true, "ANVIL_TEST_PG_DSN": true, "KUBEBUILDER_ASSETS": true,
}
var assignment = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=`)

// Check returns nil when the tool call is allowed during a run, or an
// error whose text is shown to the agent.
func Check(c Call, p Policy) error {
	input := c.Input
	switch {
	case readTools[c.Tool]:
		target, _ := input["file_path"].(string)
		if target == "" {
			target, _ = input["notebook_path"].(string)
		}
		if target == "" {
			target, _ = input["path"].(string)
		}
		return checkReadPath(target, p)
	case searchTools[c.Tool]:
		if err := checkSearchPath(input["path"], p); err != nil {
			return err
		}
		if pattern, _ := input["pattern"].(string); filepath.IsAbs(pattern) {
			prefix := pattern
			if i := strings.IndexAny(pattern, "*?[{"); i >= 0 {
				prefix = pattern[:i]
			}
			if err := checkSearchPath(prefix, p); err != nil {
				return err
			}
		}
		return nil
	case allowTools[c.Tool]:
		return nil
	case c.Tool == "run_subagent":
		if input["profile"] == "tutor" {
			return nil
		}
		return blocked("only the tutor subagent is allowed")
	case c.Tool == "Task":
		if input["subagent_type"] == "tutor" {
			return nil
		}
		return blocked("only the tutor subagent is allowed")
	case writeTools[c.Tool]:
		target, _ := input["file_path"].(string)
		if target == "" {
			target, _ = input["notebook_path"].(string)
		}
		return checkWrite(target, p)
	case c.Tool == "apply_patch":
		text, _ := input["command"].(string)
		if text == "" {
			text, _ = input["patch"].(string)
		}
		if text == "" {
			text, _ = input["input"].(string)
		}
		return checkPatch(text, c.Cwd, p)
	case c.Tool == "exec" || c.Tool == "Bash":
		return checkExec(input, p)
	default:
		return blocked("%q is not allowed", c.Tool)
	}
}

// checkReadPath requires an absolute path outside the keys repo.
func checkReadPath(target string, p Policy) error {
	if target == "" {
		return blocked("%s needs a path", "read calls")
	}
	if !filepath.IsAbs(target) {
		return blocked("path %q is not absolute", target)
	}
	if inside(target, p.Keys) {
		return blocked("path %q is inside the keys repository", target)
	}
	return nil
}

// checkSearchPath requires an absolute path outside the keys repo that
// does not contain it either.
func checkSearchPath(v any, p Policy) error {
	target, _ := v.(string)
	if target == "" {
		return blocked("search calls need a path")
	}
	if !filepath.IsAbs(target) {
		return blocked("path %q is not absolute", target)
	}
	if inside(target, p.Keys) {
		return blocked("path %q is inside the keys repository", target)
	}
	if inside(p.Keys, target) {
		return blocked("path %q contains the keys repository", target)
	}
	return nil
}

func checkWrite(target string, p Policy) error {
	if target == "" {
		return blocked("write calls need a file_path")
	}
	if !filepath.IsAbs(target) {
		return blocked("write target %q is not an absolute path; answers go under the answers directory", target)
	}
	parent, err := resolve(filepath.Dir(filepath.Clean(target)))
	if err != nil {
		return blocked("cannot resolve %q: %v", target, err)
	}
	if parent != p.Answers && !strings.HasPrefix(parent, p.Answers+string(os.PathSeparator)) {
		return blocked("write target %q is outside the answers directory", target)
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return blocked("write target %q is a symlink", target)
	}
	return nil
}

var patchPrefix = []string{"*** Add File: ", "*** Update File: ", "*** Delete File: ", "*** Move to: "}

func checkPatch(text, cwd string, p Policy) error {
	var paths []string
	for _, line := range strings.Split(text, "\n") {
		for _, pre := range patchPrefix {
			if rest, ok := strings.CutPrefix(line, pre); ok {
				paths = append(paths, strings.TrimSpace(rest))
			}
		}
	}
	if len(paths) == 0 {
		return blocked("the patch names no files")
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			if cwd == "" {
				return blocked("the patch path %q is relative and the call has no cwd", path)
			}
			path = filepath.Join(cwd, path)
		}
		if err := checkWrite(path, p); err != nil {
			return err
		}
	}
	return nil
}

func checkExec(input map[string]any, p Policy) error {
	if env, ok := input["env"]; ok {
		obj, ok := env.(map[string]any)
		if !ok {
			return blocked("env must be an object")
		}
		for k := range obj {
			if !envNames[k] {
				return blocked("environment variable %q is not allowed; only ANVIL_KEYS, ANVIL_TEST_PG_DSN and KUBEBUILDER_ASSETS are", k)
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
	return checkCommand(cmd, p)
}

// forbidden are the characters no shell command may contain.
const forbidden = "\n\r`\\$;&|<>(){}*?[]~"

// checkCommand tokenizes the command and checks it against a tiny
// allowlist: anvil subcommands and read-only git.
func checkCommand(cmd string, p Policy) error {
	if strings.ContainsAny(cmd, forbidden) {
		return blocked("the command contains a forbidden character")
	}
	words, err := tokenize(cmd)
	if err != nil {
		return err
	}
	if len(words) == 0 {
		return blocked("the command is empty")
	}
	for _, w := range words {
		if assignment.MatchString(w) {
			return blocked("environment assignments are not allowed")
		}
	}
	switch words[0] {
	case "anvil":
		return checkAnvil(words[1:], p)
	case "git":
		return checkGit(words[1:], p)
	default:
		return blocked("%q is not allowed", words[0])
	}
}

// tokenize splits on whitespace outside single or double quotes, which
// are literal. An unterminated quote is an error.
func tokenize(cmd string) ([]string, error) {
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune
	for _, c := range cmd {
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
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, blocked("the command has an unterminated quote")
	}
	if inWord {
		words = append(words, word.String())
	}
	return words, nil
}

func checkGit(args []string, p Policy) error {
	if len(args) < 2 || args[0] != "-C" {
		return blocked("git needs -C with an absolute directory")
	}
	if err := checkReadPath(args[1], p); err != nil {
		return err
	}
	args = args[2:]
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

func checkAnvil(args []string, p Policy) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "grade", "review", "path", "projects", "topics", "help", "build", "check", "start":
	default:
		return blocked("anvil %q is not allowed", args[0])
	}
	for _, a := range args[1:] {
		if filepath.IsAbs(a) && inside(a, p.Keys) {
			return blocked("path %q is inside the keys repository", a)
		}
	}
	return nil
}
