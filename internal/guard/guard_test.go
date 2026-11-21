package guard

import (
	"os"
	"path/filepath"
	"testing"
)

func shell(cmd string) map[string]any {
	return map[string]any{"command": cmd}
}

func canonical(t *testing.T, dir string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func TestCheck(t *testing.T) {
	keys := canonical(t, t.TempDir())
	if err := os.MkdirAll(filepath.Join(keys, "keys"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keys, "keys", "redis.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	answers := canonical(t, t.TempDir())
	ws := canonical(t, t.TempDir())
	if err := os.MkdirAll(filepath.Join(ws, "s01"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "s01", "stage.go"), []byte("package stage"), 0o644); err != nil {
		t.Fatal(err)
	}
	peek := filepath.Join(ws, "peek")
	if err := os.Symlink(filepath.Join(keys, "keys", "redis.json"), peek); err != nil {
		t.Fatal(err)
	}
	p := Policy{Keys: keys, Answers: answers}

	cases := []struct {
		name  string
		call  Call
		allow bool
	}{
		{"read ws file", Call{"Read", map[string]any{"file_path": filepath.Join(ws, "s01", "stage.go")}, ""}, true},
		{"grep ws", Call{"Grep", map[string]any{"path": ws, "pattern": "x"}, ""}, true},
		{"glob ws", Call{"Glob", map[string]any{"path": ws, "pattern": ws + "/**/*.go"}, ""}, true},
		{"write answer", Call{"Write", map[string]any{"file_path": filepath.Join(answers, "a.txt")}, ""}, true},
		{"cat stage", Call{"exec", shell("cat " + filepath.Join(ws, "s01", "stage.go")), ""}, true},
		{"head -n", Call{"exec", shell("head -n 5 " + filepath.Join(ws, "f")), ""}, true},
		{"git diff", Call{"exec", shell("git -C " + ws + " diff HEAD"), ""}, true},
		{"anvil grade", Call{"exec", shell("anvil grade redis 1 " + filepath.Join(answers, "a.txt")), ""}, true},
		{"anvil build", Call{"exec", shell("anvil build " + ws + " redis 1"), ""}, true},
		{"anvil fix", Call{"exec", shell("anvil fix " + ws), ""}, true},
		{"anvil review", Call{"exec", shell("anvil review " + ws), ""}, true},
		{"pg dsn", Call{"exec", shell("ANVIL_TEST_PG_DSN=x anvil fix " + ws), ""}, true},
		{"tutor subagent", Call{"run_subagent", map[string]any{"profile": "tutor"}, ""}, true},
		{"patch answer", Call{"apply_patch", map[string]any{"command": "*** Update File: " + filepath.Join(answers, "a.txt") + "\n@@ x"}, ""}, true},

		{"read keys", Call{"Read", map[string]any{"file_path": filepath.Join(keys, "keys", "redis.json")}, ""}, false},
		{"read via symlink", Call{"Read", map[string]any{"file_path": peek}, ""}, false},
		{"read relative", Call{"Read", map[string]any{"file_path": "s01/stage.go"}, ""}, false},
		{"grep no path", Call{"Grep", map[string]any{"pattern": "x"}, ""}, false},
		{"grep keys parent", Call{"Grep", map[string]any{"path": filepath.Dir(keys), "pattern": "x"}, ""}, false},
		{"glob keys", Call{"Glob", map[string]any{"path": ws, "pattern": keys + "/**"}, ""}, false},
		{"cat keys", Call{"exec", shell("cat " + filepath.Join(keys, "keys", "redis.json")), ""}, false},
		{"cat relative", Call{"exec", shell("cat stage.go"), ""}, false},
		{"ls no args", Call{"exec", shell("ls"), ""}, false},
		{"cat tilde", Call{"exec", shell("cat ~/x"), ""}, false},
		{"git no -C", Call{"exec", shell("git diff"), ""}, false},
		{"git keys", Call{"exec", shell("git -C " + keys + " log -p"), ""}, false},
		{"review keys", Call{"exec", shell("anvil review " + keys), ""}, false},
		{"anvil unlock", Call{"exec", shell("anvil unlock " + ws + " a"), ""}, false},
		{"anvil stop", Call{"exec", shell("anvil stop"), ""}, false},
		{"anvil check-keys", Call{"exec", shell("anvil check-keys"), ""}, false},
		{"anvil guard", Call{"exec", shell("anvil guard"), ""}, false},
		{"go test", Call{"exec", shell("go test ./..."), ""}, false},
		{"cd", Call{"exec", shell("cd /tmp"), ""}, false},
		{"patch ws file", Call{"apply_patch", map[string]any{"command": "*** Update File: " + filepath.Join(ws, "s01", "stage.go")}, ""}, false},
		{"patch relative no cwd", Call{"apply_patch", map[string]any{"command": "*** Update File: a.txt"}, ""}, false},
		{"mcp_read_resource", Call{"mcp_read_resource", map[string]any{}, ""}, false},
		{"read_page_content", Call{"read_page_content", map[string]any{}, ""}, false},
		{"general subagent", Call{"run_subagent", map[string]any{"profile": "subagent_general"}, ""}, false},
		{"mcp__x__y", Call{"mcp__x__y", map[string]any{}, ""}, false},
		{"sidekick", Call{"sidekick", map[string]any{}, ""}, false},
		{"write_to_process", Call{"write_to_process", map[string]any{}, ""}, false},
		{"chmod", Call{"exec", shell("chmod -R u+w " + ws), ""}, false},
		{"semicolon", Call{"exec", shell("ls " + ws + " ; rm x"), ""}, false},
		{"pipe", Call{"exec", shell("ls " + ws + " | tee x"), ""}, false},
		{"substitution", Call{"exec", shell("echo $(id)"), ""}, false},
		{"redirect", Call{"exec", shell("echo hi > f"), ""}, false},
		{"embedded newline", Call{"exec", shell("ls\nrm x"), ""}, false},
		{"judge base url", Call{"exec", shell("ANVIL_JUDGE_BASE_URL=http://x anvil grade redis 1 a"), ""}, false},
		{"grade cache", Call{"exec", shell("ANVIL_GRADE_CACHE=/tmp/c anvil grade redis 1 a"), ""}, false},
		{"env goflags", Call{"exec", map[string]any{"command": "anvil fix " + ws, "env": map[string]any{"GOFLAGS": "x"}}, ""}, false},
		{"env judge url", Call{"exec", map[string]any{"command": "anvil grade redis 1 a", "env": map[string]any{"ANVIL_JUDGE_BASE_URL": "x"}}, ""}, false},
		{"powershell", Call{"exec", map[string]any{"command": "ls " + ws, "shell_flavor": "powershell"}, ""}, false},
		{"unterminated quote", Call{"exec", shell("echo 'hi"), ""}, false},
		{"empty command", Call{"exec", shell(""), ""}, false},
		{"write outside", Call{"Write", map[string]any{"file_path": filepath.Join(ws, "x.txt")}, ""}, false},
		{"write relative", Call{"Write", map[string]any{"file_path": "answers/a.txt"}, ""}, false},
	}
	for _, tc := range cases {
		err := Check(tc.call, p)
		if tc.allow && err != nil {
			t.Errorf("%s: blocked: %v", tc.name, err)
		}
		if !tc.allow && err == nil {
			t.Errorf("%s: allowed, want block", tc.name)
		}
	}
}
