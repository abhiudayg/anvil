package grader

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

var sampleKey = Key{
	Project: "redis",
	Stage:   "RESP reader and TCP server",
	Rubric: []Point{
		{Name: "Bulk strings are length prefixed."},
		{Name: "Values may contain any byte, so no delimiter marks the end."},
		{Name: "The reader must read exactly the announced number of bytes."},
	},
}

// fakeModel answers every request with the given judgement vector and counts calls.
func fakeModel(t *testing.T, covered []bool, calls *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		if r.Header.Get("x-api-key") == "" || r.Header.Get("anthropic-version") == "" {
			http.Error(w, "missing headers", http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model       string  `json:"model"`
			Temperature float64 `json:"temperature"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Temperature != 0 || req.Model == "" {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}
		reply, _ := json.Marshal(map[string]any{"covered": covered})
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": string(reply)}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestClient(t *testing.T, base string) *Client {
	t.Helper()
	c := NewClient("test-key")
	c.BaseURL = base
	c.Cache = t.TempDir()
	return c
}

func TestGradeReadsJudgementsPerPoint(t *testing.T) {
	var calls int32
	srv := fakeModel(t, []bool{true, false, true}, &calls)
	res, err := newTestClient(t, srv.URL).Grade(sampleKey, "Some answer.")
	if err != nil {
		t.Fatal(err)
	}
	if res.Perfect {
		t.Error("two of three points covered must not be perfect")
	}
	if m := res.Missing(); len(m) != 1 || m[0] != 2 {
		t.Errorf("Missing = %v, want [2]", m)
	}
}

func TestGradePerfectWhenEveryPointCovered(t *testing.T) {
	var calls int32
	srv := fakeModel(t, []bool{true, true, true}, &calls)
	res, err := newTestClient(t, srv.URL).Grade(sampleKey, "A full answer.")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Perfect {
		t.Error("all points covered should be perfect")
	}
}

func TestGradeCachesSameAnswer(t *testing.T) {
	var calls int32
	srv := fakeModel(t, []bool{true, true, true}, &calls)
	c := newTestClient(t, srv.URL)
	for i := 0; i < 3; i++ {
		if _, err := c.Grade(sampleKey, "Same answer."); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Errorf("model called %d times for one answer, want 1", calls)
	}
	entries, _ := os.ReadDir(c.Cache)
	if len(entries) != 1 {
		t.Errorf("cache has %d entries, want 1", len(entries))
	}
}

func TestGradeCacheKeyDependsOnAnswer(t *testing.T) {
	var calls int32
	srv := fakeModel(t, []bool{true, true, true}, &calls)
	c := newTestClient(t, srv.URL)
	c.Grade(sampleKey, "First answer.")
	c.Grade(sampleKey, "Second answer.")
	if calls != 2 {
		t.Errorf("model called %d times for two answers, want 2", calls)
	}
}

func TestEmptyAnswerNeverReachesModel(t *testing.T) {
	var calls int32
	srv := fakeModel(t, []bool{true, true, true}, &calls)
	res, err := newTestClient(t, srv.URL).Grade(sampleKey, "   ")
	if err != nil {
		t.Fatal(err)
	}
	if res.Perfect || calls != 0 {
		t.Errorf("empty answer: perfect=%v calls=%d, want false and 0", res.Perfect, calls)
	}
}

func TestGradeRejectsWrongJudgementCount(t *testing.T) {
	var calls int32
	srv := fakeModel(t, []bool{true}, &calls)
	if _, err := newTestClient(t, srv.URL).Grade(sampleKey, "answer"); err == nil {
		t.Error("a judgement list of the wrong length must be an error, not a pass")
	}
}

func TestGradeReportsHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := newTestClient(t, srv.URL).Grade(sampleKey, "answer"); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("HTTP failure must be an error mentioning the status, got %v", err)
	}
}

// claudeCLIStub writes a fake `claude` binary executing script body, and
// returns a client pointed at it.
func claudeCLIStub(t *testing.T, script string) *Client {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-stub")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	c := NewClaudeCLIClient()
	c.Command = path
	c.Cache = t.TempDir()
	return c
}

func TestClaudeCLIGradesFromJSONEnvelope(t *testing.T) {
	c := claudeCLIStub(t, `case " $* " in *" --tools "*) ;; *) echo "missing --tools" >&2; exit 3;; esac
case " $* " in *" --system-prompt "*) ;; *) echo "missing --system-prompt" >&2; exit 3;; esac
cat >/dev/null
printf '%s\n' '{"result":"Here you go:\n`+"```"+`json\n{\"covered\":[true,false]}\n`+"```"+`","is_error":false}'`)
	key := sampleKey
	key.Rubric = key.Rubric[:2]
	res, err := c.Grade(key, "answer")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Covered) != 2 || !res.Covered[0] || res.Covered[1] {
		t.Errorf("Covered = %v, want [true false]", res.Covered)
	}
}

func TestClaudeCLIReportsCLIError(t *testing.T) {
	c := claudeCLIStub(t, `cat >/dev/null
printf '%s\n' '{"result":"Not logged in","is_error":true}'
exit 1`)
	_, err := c.Grade(sampleKey, "answer")
	if err == nil || !strings.Contains(err.Error(), "Not logged in") {
		t.Fatalf("error = %v, want one containing the CLI's error text", err)
	}
}

func TestClaudeCLICachesSameAnswer(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "count")
	c := claudeCLIStub(t, `cat >/dev/null
echo x >> "`+counter+`"
printf '%s\n' '{"result":"{\"covered\":[true,true,true]}","is_error":false}'`)
	for i := 0; i < 2; i++ {
		if _, err := c.Grade(sampleKey, "Same answer."); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "x"); n != 1 {
		t.Errorf("claude CLI ran %d times for one answer, want 1", n)
	}
}

func TestParseCoveredAcceptsProseAroundJSON(t *testing.T) {
	got, err := parseCovered("Here you go: {\"covered\": [true, false]} Done.", 2)
	if err != nil || len(got) != 2 || !got[0] || got[1] {
		t.Errorf("parseCovered = %v, %v", got, err)
	}
	if _, err := parseCovered("no json here", 2); err == nil {
		t.Error("reply without JSON must be an error")
	}
}

func TestCachePathIsStableAndDistinct(t *testing.T) {
	c := NewClient("k")
	c.Cache = filepath.Join("x")
	a := c.cachePath(sampleKey, "one")
	if a != c.cachePath(sampleKey, "one") {
		t.Error("same inputs must give the same cache path")
	}
	if a == c.cachePath(sampleKey, "two") {
		t.Error("different answers must give different cache paths")
	}
}

func TestCachePathIncludesBaseURL(t *testing.T) {
	c := NewClient("k")
	c.Cache = filepath.Join("x")
	a := c.cachePath(sampleKey, "one")
	c.BaseURL = "http://fake-judge.example"
	if a == c.cachePath(sampleKey, "one") {
		t.Error("a different BaseURL must give a different cache path")
	}
}
