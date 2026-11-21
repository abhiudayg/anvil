package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const fixtureKeys = `{
  "project": "redis",
  "stages": [
    {
      "number": 1,
      "stage": "RESP reader and TCP server",
      "concept": "bufio-scanner",
      "injection": "x",
      "root_cause": "y",
      "rubric": [
        {"name": "Bulk strings are length prefixed."},
        {"name": "The reader reads exactly the announced number of bytes."},
        {"name": "The trailing CRLF follows the payload."}
      ],
      "reference_answer": "Bulk strings are length prefixed. The reader reads exactly the announced number of bytes. The trailing CRLF follows the payload."
    }
  ]
}
`

func writeKeys(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "keys"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keys", "redis.json"), []byte(fixtureKeys), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeAnswer(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "answer.txt")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// setupModel points the CLI at a fake model that returns the given judgements.
func setupModel(t *testing.T, covered []bool) *int32 {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		io.ReadAll(r.Body)
		reply, _ := json.Marshal(map[string]any{"covered": covered})
		json.NewEncoder(w).Encode(map[string]any{"content": []map[string]string{{"type": "text", "text": string(reply)}}})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("ANVIL_JUDGE", "anthropic")
	t.Setenv("ANVIL_JUDGE_BASE_URL", srv.URL)
	t.Setenv("ANVIL_GRADE_CACHE", t.TempDir())
	return &calls
}

func TestGradePerfectAnswerExitsZero(t *testing.T) {
	setupModel(t, []bool{true, true, true})
	keys := writeKeys(t)
	ans := writeAnswer(t, "Bulk strings are length prefixed, so the reader reads exactly that many bytes, then the CRLF.")
	code, out, _ := runCLI("grade", "--keys", keys, "redis", "1", ans)
	if code != 0 || !strings.Contains(out, "PERFECT") {
		t.Errorf("exit = %d, out = %s", code, out)
	}
}

func TestGradePartialAnswerExitsOneAndNamesPointsByNumber(t *testing.T) {
	setupModel(t, []bool{true, false, true})
	keys := writeKeys(t)
	ans := writeAnswer(t, "It is length prefixed.")
	code, out, _ := runCLI("grade", "--keys", keys, "redis", "1", ans)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(out, "point 1: covered") || !strings.Contains(out, "point 2: missing") {
		t.Errorf("unexpected output:\n%s", out)
	}
	if strings.Contains(out, "announced") || strings.Contains(out, "CRLF") {
		t.Error("output must not reveal rubric text")
	}
}

func TestGradeNeedsAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	keys := writeKeys(t)
	ans := writeAnswer(t, "x")
	if code, _, stderr := runCLI("grade", "--keys", keys, "redis", "1", ans); code != 2 || !strings.Contains(stderr, "ANTHROPIC_API_KEY") {
		t.Errorf("exit = %d, stderr = %q", code, stderr)
	}
}

func TestGradeUsageErrors(t *testing.T) {
	t.Setenv("ANVIL_KEYS", "")
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	keys := writeKeys(t)
	ans := writeAnswer(t, "x")
	cases := map[string][]string{
		"no keys dir":     {"grade", "redis", "1", ans},
		"bad stage":       {"grade", "--keys", keys, "redis", "99", ans},
		"bad number":      {"grade", "--keys", keys, "redis", "one", ans},
		"unknown project": {"grade", "--keys", keys, "nope", "1", ans},
		"missing answer":  {"grade", "--keys", keys, "redis", "1", filepath.Join(t.TempDir(), "absent.txt")},
	}
	for name, args := range cases {
		if code, _, _ := runCLI(args...); code != 2 {
			t.Errorf("%s: exit = %d, want 2", name, code)
		}
	}
}

// TestKeysRepoCrossCheck runs the offline cross-check against the real keys repository.
// It needs ANVIL_KEYS, so it is skipped otherwise.
func TestKeysRepoCrossCheck(t *testing.T) {
	dir := os.Getenv("ANVIL_KEYS")
	if dir == "" {
		t.Skip("set ANVIL_KEYS to the keys repository to run the cross-check")
	}
	code, out, _ := runCLI("check-keys", "--keys", dir)
	if code != 0 {
		t.Errorf("check-keys exit = %d:\n%s", code, out)
	}
}

// TestKeysRepoSemanticCheck runs the judge-backed checks against the real keys repository.
// It needs ANVIL_KEYS and ANTHROPIC_API_KEY, and costs one judgement per point.
func TestKeysRepoSemanticCheck(t *testing.T) {
	dir := os.Getenv("ANVIL_KEYS")
	if dir == "" || os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("set ANVIL_KEYS and ANTHROPIC_API_KEY to run the semantic cross-check")
	}
	code, out, _ := runCLI("check-keys", "--keys", dir, "--semantic")
	if code != 0 {
		t.Errorf("semantic check exit = %d:\n%s", code, out)
	}
}

// TestGradeWithGeminiJudge runs the same grade through the Gemini provider.
func TestGradeWithGeminiJudge(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Header.Get("x-goog-api-key") != "gemini-key" {
			http.Error(w, "unauthorised", http.StatusUnauthorized)
			return
		}
		io.ReadAll(r.Body)
		reply, _ := json.Marshal(map[string]any{"covered": []bool{true, true, true}})
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": string(reply)}}}}},
		})
	}))
	defer srv.Close()
	t.Setenv("ANVIL_JUDGE", "gemini")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("ANVIL_JUDGE_BASE_URL", srv.URL)
	t.Setenv("ANVIL_GRADE_CACHE", t.TempDir())
	keys := writeKeys(t)
	ans := writeAnswer(t, "Bulk strings are length prefixed; the reader reads exactly that many bytes, then the CRLF.")
	code, out, _ := runCLI("grade", "--keys", keys, "redis", "1", ans)
	if code != 0 || !strings.Contains(out, "PERFECT") || hits != 1 {
		t.Errorf("exit = %d, hits = %d, out = %s", code, hits, out)
	}
}

func TestGradeRejectsUnknownJudge(t *testing.T) {
	t.Setenv("ANVIL_JUDGE", "nope")
	keys := writeKeys(t)
	ans := writeAnswer(t, "x")
	if code, _, stderr := runCLI("grade", "--keys", keys, "redis", "1", ans); code != 2 || !strings.Contains(stderr, "unknown ANVIL_JUDGE") {
		t.Errorf("exit = %d, stderr = %q", code, stderr)
	}
}
