// Package grader grades a learner's answer by meaning.
//
// Each rubric point is a statement of something a correct answer must convey.
// A language model judges, for each point, whether the answer conveys that
// meaning. Paraphrase counts. Naming the right words without the meaning does
// not. The model runs at temperature 0, and every judgement is cached on disk
// by a hash of the model, prompt, stage, rubric and answer, so the same answer
// gets the same grade.
//
// The rubric and the reference answers are hidden. They live in the separate
// keys repository and are loaded from a directory at run time.
package grader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Point is one rubric point: the meaning a correct answer must convey.
type Point struct {
	Name string `json:"name"`
}

// Key is the hidden answer key for one stage.
type Key struct {
	Project         string  `json:"-"`
	Number          int     `json:"number"`
	Stage           string  `json:"stage"`
	Concept         string  `json:"concept"`
	Injection       string  `json:"injection"`
	RootCause       string  `json:"root_cause"`
	Rubric          []Point `json:"rubric"`
	ReferenceAnswer string  `json:"reference_answer"`
}

// Result is the grading of one answer.
type Result struct {
	Covered []bool // one entry per rubric point, in order
	Perfect bool   // every point covered
}

// Missing returns the 1-based numbers of the rubric points not covered.
func (r Result) Missing() []int {
	var out []int
	for i, ok := range r.Covered {
		if !ok {
			out = append(out, i+1)
		}
	}
	return out
}

// Client talks to a model API and caches judgements.
type Client struct {
	Provider string // "anthropic" (default), "gemini" or "claude-cli"
	APIKey   string
	BaseURL  string // defaults to the provider's endpoint; empty for claude-cli
	Model    string // defaults to the provider's default model
	Cache    string // directory for cached judgements; empty disables caching
	HTTP     *http.Client
	Command  string // the claude-cli binary; defaults to "claude"
}

const (
	defaultBaseURL       = "https://api.anthropic.com"
	defaultModel         = "claude-sonnet-5-5"
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	defaultGeminiModel   = "gemini-3.8-flash"
	defaultClaudeModel   = "sonnet"
	promptVersion        = "meaning-v1"
	apiVersion           = "2023-06-01"
)

const systemPrompt = `You grade a learner's answer to a Go or Kubernetes question against numbered rubric points.

For each point, decide whether the answer conveys the meaning of that point.
- Paraphrase, different wording, or a different order counts.
- Naming the right terms without stating what they mean does not count.
- Wrong claims that contradict the point do not count.
- Judge only what the answer says. Do not reward guesses or hedging.

Reply with JSON only, in exactly this form, with one boolean per point in order:
{"covered": [true, false, ...]}`

// NewClient returns an Anthropic client for the given API key with defaults filled in.
func NewClient(apiKey string) *Client {
	return &Client{Provider: "anthropic", APIKey: apiKey, BaseURL: defaultBaseURL, Model: defaultModel,
		HTTP: &http.Client{Timeout: 120 * time.Second}}
}

// NewGeminiClient returns a Gemini client for the given API key.
func NewGeminiClient(apiKey string) *Client {
	return &Client{Provider: "gemini", APIKey: apiKey, BaseURL: defaultGeminiBaseURL, Model: defaultGeminiModel,
		HTTP: &http.Client{Timeout: 120 * time.Second}}
}

// NewClaudeCLIClient returns a client that grades through the user's
// logged-in Claude Code CLI; no API key is involved.
func NewClaudeCLIClient() *Client {
	return &Client{Provider: "claude-cli", Model: defaultClaudeModel, Command: "claude"}
}

func (c *Client) provider() string {
	if c.Provider == "" {
		return "anthropic"
	}
	return c.Provider
}

// Grade judges answer against the key's rubric.
func (c *Client) Grade(key Key, answer string) (Result, error) {
	covered, err := c.judge(key, answer)
	if err != nil {
		return Result{}, err
	}
	res := Result{Covered: covered, Perfect: len(covered) > 0}
	for _, ok := range covered {
		if !ok {
			res.Perfect = false
		}
	}
	return res, nil
}

func (c *Client) judge(key Key, answer string) ([]bool, error) {
	if len(key.Rubric) == 0 {
		return nil, errors.New("key has no rubric points")
	}
	if strings.TrimSpace(answer) == "" {
		return make([]bool, len(key.Rubric)), nil
	}
	cacheFile := c.cachePath(key, answer)
	if cacheFile != "" {
		if data, err := os.ReadFile(cacheFile); err == nil {
			var covered []bool
			if json.Unmarshal(data, &covered) == nil && len(covered) == len(key.Rubric) {
				return covered, nil
			}
		}
	}

	covered, err := c.callModel(key, answer)
	if err != nil {
		return nil, err
	}
	if cacheFile != "" {
		if err := os.MkdirAll(c.Cache, 0o755); err != nil {
			return nil, err
		}
		data, _ := json.Marshal(covered)
		if err := os.WriteFile(cacheFile, data, 0o644); err != nil {
			return nil, err
		}
	}
	return covered, nil
}

func (c *Client) cachePath(key Key, answer string) string {
	if c.Cache == "" {
		return ""
	}
	h := sha256.New()
	for _, part := range []string{c.provider(), c.model(), c.BaseURL, promptVersion, key.Project, key.Stage, answer} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	for _, p := range key.Rubric {
		h.Write([]byte(p.Name))
		h.Write([]byte{0})
	}
	return filepath.Join(c.Cache, hex.EncodeToString(h.Sum(nil))+".json")
}

func (c *Client) model() string {
	if c.Model != "" {
		return c.Model
	}
	switch c.provider() {
	case "gemini":
		return defaultGeminiModel
	case "claude-cli":
		return defaultClaudeModel
	}
	return defaultModel
}

// userPrompt lists the rubric points and the learner's answer for the model.
func userPrompt(key Key, answer string) string {
	var user strings.Builder
	fmt.Fprintf(&user, "Question topic: %s\n\nRubric points:\n", key.Stage)
	for i, p := range key.Rubric {
		fmt.Fprintf(&user, "%d. %s\n", i+1, p.Name)
	}
	fmt.Fprintf(&user, "\nLearner's answer:\n<answer>\n%s\n</answer>\n", answer)
	return user.String()
}

func (c *Client) callModel(key Key, answer string) ([]bool, error) {
	switch c.provider() {
	case "gemini":
		return c.callGemini(key, answer)
	case "claude-cli":
		return c.callClaudeCLI(key, answer)
	}
	user := userPrompt(key, answer)

	body, err := json.Marshal(map[string]any{
		"model":       c.model(),
		"max_tokens":  300,
		"temperature": 0,
		"system":      systemPrompt,
		"messages":    []map[string]string{{"role": "user", "content": user}},
	})
	if err != nil {
		return nil, err
	}
	base := c.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(base, "/")+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", apiVersion)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grading request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grading request returned %s", resp.Status)
	}
	var msg struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("grading response is not JSON: %w", err)
	}
	var text string
	for _, block := range msg.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return parseCovered(text, len(key.Rubric))
}

// callClaudeCLI grades through the logged-in Claude Code CLI. The prompt
// goes on stdin, the judgement comes back in the JSON envelope's result
// field. The command runs in the temp dir so no project configuration is
// loaded. No --bare flag: it would skip the keychain read and break
// subscription auth.
func (c *Client) callClaudeCLI(key Key, answer string) ([]bool, error) {
	command := c.Command
	if command == "" {
		command = "claude"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command,
		"-p", "--output-format", "json",
		"--model", c.model(),
		"--system-prompt", systemPrompt,
		"--tools", "",
		"--strict-mcp-config",
		"--no-session-persistence",
		"--disable-slash-commands")
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(userPrompt(key, answer))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	var reply struct {
		Result  string `json:"result"`
		IsError bool   `json:"is_error"`
	}
	var detail string
	if err := json.Unmarshal(stdout.Bytes(), &reply); err == nil {
		detail = reply.Result
	}
	if detail == "" {
		detail = stderr.String()
	}
	if len(detail) > 500 {
		detail = detail[:500]
	}
	if runErr != nil {
		return nil, fmt.Errorf("claude CLI failed: %v: %s", runErr, detail)
	}
	if reply.IsError {
		return nil, fmt.Errorf("claude CLI returned an error: %s", detail)
	}
	return parseCovered(reply.Result, len(key.Rubric))
}

// parseCovered reads {"covered": [...]} and checks it has one boolean per point.
func parseCovered(text string, points int) ([]bool, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("grading reply has no JSON object")
	}
	var v struct {
		Covered []bool `json:"covered"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &v); err != nil {
		return nil, fmt.Errorf("grading reply is not valid: %w", err)
	}
	if len(v.Covered) != points {
		return nil, fmt.Errorf("grading reply has %d judgements for %d points", len(v.Covered), points)
	}
	return v.Covered, nil
}

// Load reads every project key file in dir/keys. Keys are keyed by "<project>/<stage title>".
func Load(dir string) (map[string]Key, error) {
	files, err := filepath.Glob(filepath.Join(dir, "keys", "*.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no key files in %s/keys", dir)
	}
	keys := map[string]Key{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var file struct {
			Project string `json:"project"`
			Stages  []Key  `json:"stages"`
		}
		if err := json.Unmarshal(data, &file); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		for _, k := range file.Stages {
			k.Project = file.Project
			id := file.Project + "/" + k.Stage
			if _, dup := keys[id]; dup {
				return nil, fmt.Errorf("%s: duplicate key for %q", f, id)
			}
			keys[id] = k
		}
	}
	return keys, nil
}

// Lookup returns the key for a stage.
func Lookup(keys map[string]Key, project, stage string) (Key, bool) {
	k, ok := keys[project+"/"+stage]
	return k, ok
}

// SortedIDs returns the keys' identifiers in order, for stable reports.
func SortedIDs(keys map[string]Key) []string {
	ids := make([]string, 0, len(keys))
	for id := range keys {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// callGemini sends the same prompt to the Gemini generateContent API at temperature 0.
func (c *Client) callGemini(key Key, answer string) ([]bool, error) {
	body, err := json.Marshal(map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": systemPrompt}}},
		"contents":          []map[string]any{{"role": "user", "parts": []map[string]string{{"text": userPrompt(key, answer)}}}},
		"generationConfig": map[string]any{
			"temperature":      0,
			"maxOutputTokens":  300,
			"responseMimeType": "application/json",
		},
	})
	if err != nil {
		return nil, err
	}
	base := c.BaseURL
	if base == "" {
		base = defaultGeminiBaseURL
	}
	url := strings.TrimRight(base, "/") + "/v1beta/models/" + c.model() + ":generateContent"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grading request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grading request returned %s", resp.Status)
	}
	var msg struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("grading response is not JSON: %w", err)
	}
	if len(msg.Candidates) == 0 {
		return nil, errors.New("grading response has no candidates")
	}
	var text string
	for _, part := range msg.Candidates[0].Content.Parts {
		text += part.Text
	}
	return parseCovered(text, len(key.Rubric))
}
