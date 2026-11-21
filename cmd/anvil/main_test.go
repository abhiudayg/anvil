package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestProjectsListsBothTracks(t *testing.T) {
	code, out, _ := runCLI("projects")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{"redis", "Build your Redis", "jobq", "Build your Job Queue"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTopicsForProject(t *testing.T) {
	code, out, _ := runCLI("topics", "REDIS")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Stage 4 (level 4): Shared state with goroutines") ||
		!strings.Contains(out, "  - Starting goroutines and their lifecycle") {
		t.Errorf("expected concurrency stage under redis:\n%s", out)
	}
}

func TestTopicsUnknownProject(t *testing.T) {
	code, _, stderr := runCLI("topics", "nope")
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, `unknown project "nope"`) {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestPathShowsLevelsInOrder(t *testing.T) {
	code, out, _ := runCLI("path")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	first := strings.Index(out, "Level 1: Foundations")
	last := strings.Index(out, "Level 12: Kubernetes in production")
	if first < 0 || last < 0 || first > last {
		t.Errorf("levels missing or out of order:\n%s", out)
	}
	if !strings.Contains(out, "[jobq] Routing and middleware with chi") {
		t.Errorf("expected jobq stage in path:\n%s", out)
	}
}

func TestPathRejectsArguments(t *testing.T) {
	code, _, stderr := runCLI("path", "extra")
	if code != 2 || !strings.Contains(stderr, "usage: anvil path") {
		t.Errorf("exit = %d, stderr = %q", code, stderr)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	code, _, stderr := runCLI("bogus")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr = %q", stderr)
	}
}
