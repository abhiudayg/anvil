package grader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"anvil/internal/curriculum"
)

var twoSteps = []curriculum.PathStep{
	{Project: curriculum.Project{Slug: "redis"}, StageNumber: 1, Stage: curriculum.Stage{Title: "One"}},
	{Project: curriculum.Project{Slug: "redis"}, StageNumber: 2, Stage: curriculum.Stage{Title: "Two"}},
}

func putFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyProjectsGood(t *testing.T) {
	dir := t.TempDir()
	putFile(t, dir, "projects/redis/base/go.mod", "module redis\n")
	putFile(t, dir, "projects/redis/stages/01/correct/a.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/01/injected/a.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/02/correct/a.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/02/correct/b.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/02/injected/b.go", "package redis\n")
	if problems := VerifyProjects(dir, twoSteps); len(problems) != 0 {
		t.Errorf("problems = %v", problems)
	}
}

func TestVerifyProjectsMissingCounterpart(t *testing.T) {
	dir := t.TempDir()
	putFile(t, dir, "projects/redis/base/go.mod", "module redis\n")
	putFile(t, dir, "projects/redis/stages/01/correct/a.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/01/injected/a.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/01/injected/extra.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/02/correct/b.go", "package redis\n")
	putFile(t, dir, "projects/redis/stages/02/injected/b.go", "package redis\n")
	problems := VerifyProjects(dir, twoSteps)
	if len(problems) != 1 || !strings.Contains(problems[0], "extra.go") {
		t.Errorf("problems = %v", problems)
	}
}

func TestVerifyProjectsMissingBase(t *testing.T) {
	dir := t.TempDir()
	problems := VerifyProjects(dir, twoSteps)
	if len(problems) == 0 {
		t.Error("want problems for a missing base")
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "base/go.mod") {
		t.Errorf("problems = %v", problems)
	}
}
