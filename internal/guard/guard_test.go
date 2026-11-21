package guard

import (
	"os"
	"path/filepath"
	"testing"
)

func shell(cmd string) map[string]any {
	return map[string]any{"command": cmd}
}

func TestCheck(t *testing.T) {
	home := t.TempDir()
	answers := filepath.Join(home, ".anvil", "answers")
	if err := os.MkdirAll(answers, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(home, "outside.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(answers, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		tool  string
		input map[string]any
		allow bool
	}{
		{"Read", "Read", map[string]any{"file_path": "/w/main.go"}, true},
		{"Grep", "Grep", map[string]any{"pattern": "x"}, true},
		{"go test", "exec", shell("go test ./... -run TestX"), true},
		{"go test flags", "exec", shell("go test -race -count=20 ./... -run 'TestGetMissing|TestSetThenGet'"), true},
		{"env assignment", "exec", shell("ANVIL_TEST_PG_DSN=postgres://x go test ./... -run TestClaims"), true},
		{"cd then go", "exec", shell("cd /tmp/ws && go test ./..."), true},
		{"git -C diff", "exec", shell("git -C /w diff HEAD"), true},
		{"anvil grade", "exec", shell("anvil grade redis 1 " + filepath.Join(answers, "a.txt")), true},
		{"anvil review", "exec", shell("anvil review /w"), true},
		{"Write answer", "Write", map[string]any{"file_path": filepath.Join(answers, "a.txt")}, true},
		{"tutor subagent", "run_subagent", map[string]any{"profile": "tutor"}, true},
		{"tutor Task", "Task", map[string]any{"subagent_type": "tutor"}, true},

		{"Edit outside", "Edit", map[string]any{"file_path": outside}, false},
		{"Write relative", "Write", map[string]any{"file_path": "answers/a.txt"}, false},
		{"Write via symlink", "Write", map[string]any{"file_path": link}, false},
		{"apply_patch", "apply_patch", map[string]any{}, false},
		{"write_to_process", "write_to_process", map[string]any{}, false},
		{"sidekick", "sidekick", map[string]any{}, false},
		{"mcp__x__y", "mcp__x__y", map[string]any{}, false},
		{"general subagent", "run_subagent", map[string]any{"profile": "subagent_general"}, false},
		{"anvil unlock", "exec", shell("anvil unlock /w redis 1 a"), false},
		{"anvil guard", "exec", shell("anvil guard"), false},
		{"chmod", "exec", shell("chmod -R u+w /w"), false},
		{"go run unlock", "exec", shell("go run ./cmd/anvil unlock"), false},
		{"go build", "exec", shell("go build ./..."), false},
		{"go test -exec", "exec", shell("go test -exec sh ./..."), false},
		{"go test -toolexec", "exec", shell("go test -toolexec=x ./..."), false},
		{"go test -o", "exec", shell("go test -o " + filepath.Join(home, ".anvil", "locks") + " ./..."), false},
		{"semicolon", "exec", shell("go test ./... ; rm x"), false},
		{"pipe", "exec", shell("go test ./... | tee x"), false},
		{"command substitution", "exec", shell("echo $(id)"), false},
		{"redirect", "exec", shell("echo hi > f"), false},
		{"embedded newline", "exec", shell("ls\nrm x"), false},
		{"GOFLAGS env", "exec", shell("GOFLAGS=-toolexec=x go test ./..."), false},
		{"judge base url", "exec", shell("ANVIL_JUDGE_BASE_URL=http://x anvil grade redis 1 a"), false},
		{"grade cache", "exec", shell("ANVIL_GRADE_CACHE=/tmp/c anvil grade redis 1 a"), false},
		{"judge model", "exec", shell("ANVIL_JUDGE_MODEL=x anvil grade redis 1 a"), false},
		{"env judge url", "exec", map[string]any{"command": "anvil grade redis 1 a", "env": map[string]any{"ANVIL_JUDGE_BASE_URL": "x"}}, false},
		{"git -c", "exec", shell("git -c core.pager=sh log"), false},
		{"git --output", "exec", shell("git diff --output=f"), false},
		{"git commit", "exec", shell("git commit"), false},
		{"abs path anvil", "exec", shell("/usr/bin/anvil unlock"), false},
		{"bash -c", "exec", shell("bash -c 'ls'"), false},
		{"env GOFLAGS", "exec", map[string]any{"command": "go test ./...", "env": map[string]any{"GOFLAGS": "x"}}, false},
		{"powershell", "exec", map[string]any{"command": "ls", "shell_flavor": "powershell"}, false},
		{"unterminated quote", "exec", shell("echo 'hi"), false},
		{"empty command", "exec", shell(""), false},
	}
	for _, tc := range cases {
		err := Check(tc.tool, tc.input, answers)
		if tc.allow && err != nil {
			t.Errorf("%s: blocked: %v", tc.name, err)
		}
		if !tc.allow && err == nil {
			t.Errorf("%s: allowed, want block", tc.name)
		}
	}
}
