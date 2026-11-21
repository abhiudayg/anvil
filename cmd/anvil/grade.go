package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"anvil/internal/curriculum"
	"anvil/internal/grader"
	"anvil/internal/run"
)

const gradeUsage = `usage:
  anvil grade [--keys DIR] <project> <stage-number> <answer-file>
  anvil check-keys [--keys DIR] [--semantic]

--keys defaults to $ANVIL_KEYS, the keys repository.
grade and check-keys --semantic call the model. ANVIL_JUDGE picks the provider:
anthropic (ANTHROPIC_API_KEY, default) or gemini (GEMINI_API_KEY).
ANVIL_JUDGE_MODEL and ANVIL_JUDGE_BASE_URL override the model and endpoint.
`

// newGrader builds the judge client from the environment. ANVIL_JUDGE selects
// the provider: "anthropic" (default, uses ANTHROPIC_API_KEY) or "gemini" (uses
// GEMINI_API_KEY). ANVIL_JUDGE_MODEL and ANVIL_JUDGE_BASE_URL override the
// model and endpoint. ANVIL_GRADE_CACHE overrides the judgement cache.
func newGrader() (*grader.Client, error) {
	var c *grader.Client
	switch provider := os.Getenv("ANVIL_JUDGE"); provider {
	case "", "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("no ANTHROPIC_API_KEY set; grading is by the model judge")
		}
		c = grader.NewClient(key)
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("ANVIL_JUDGE=gemini needs GEMINI_API_KEY")
		}
		c = grader.NewGeminiClient(key)
	default:
		return nil, fmt.Errorf("unknown ANVIL_JUDGE %q (use anthropic or gemini)", provider)
	}
	if base := os.Getenv("ANVIL_JUDGE_BASE_URL"); base != "" {
		c.BaseURL = base
	}
	if model := os.Getenv("ANVIL_JUDGE_MODEL"); model != "" {
		c.Model = model
	}
	cache := os.Getenv("ANVIL_GRADE_CACHE")
	if cache == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		cache = filepath.Join(dir, "anvil", "grades")
	}
	c.Cache = cache
	return c, nil
}

// runGrade grades one answer. Exit codes: 0 perfect, 1 not perfect, 2 usage or setup error.
func runGrade(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("grade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keysDir := fs.String("keys", os.Getenv("ANVIL_KEYS"), "keys repository directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 3 {
		fmt.Fprint(stderr, gradeUsage)
		return 2
	}
	answer, err := os.ReadFile(rest[2])
	if err != nil {
		fmt.Fprintf(stderr, "anvil grade: %v\n", err)
		return 2
	}
	if *keysDir == "" {
		if state, ok, err := run.Load(); err == nil && ok {
			*keysDir = state.Keys
		}
	}
	return gradeStage(*keysDir, rest[0], rest[1], string(answer), stdout, stderr)
}

// gradeStage grades the answer text against the key for the project's stage.
// Exit codes: 0 perfect, 1 not perfect, 2 usage or setup error.
func gradeStage(keysDir, project, stage, answer string, stdout, stderr io.Writer) int {
	if keysDir == "" {
		fmt.Fprintln(stderr, "anvil grade: no keys directory (use --keys or set ANVIL_KEYS)")
		return 2
	}
	number, err := strconv.Atoi(stage)
	if err != nil {
		fmt.Fprintf(stderr, "anvil grade: stage number must be a number, got %q\n", stage)
		return 2
	}
	step, ok := findStep(project, number)
	if !ok {
		fmt.Fprintf(stderr, "anvil grade: no stage %d in project %q\n", number, project)
		return 2
	}
	keys, err := grader.Load(keysDir)
	if err != nil {
		fmt.Fprintf(stderr, "anvil grade: %v\n", err)
		return 2
	}
	key, ok := grader.Lookup(keys, step.Project.Slug, step.Stage.Title)
	if !ok {
		fmt.Fprintf(stderr, "anvil grade: no key for %s stage %d\n", step.Project.Slug, number)
		return 2
	}
	client, err := newGrader()
	if err != nil {
		fmt.Fprintf(stderr, "anvil grade: %v\n", err)
		return 2
	}
	res, err := client.Grade(key, answer)
	if err != nil {
		fmt.Fprintf(stderr, "anvil grade: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "%s stage %d: %s\n", step.Project.Slug, number, step.Stage.Title)
	for i, covered := range res.Covered {
		status := "missing"
		if covered {
			status = "covered"
		}
		fmt.Fprintf(stdout, "  point %d: %s\n", i+1, status)
	}
	if res.Perfect {
		fmt.Fprintln(stdout, "Result: PERFECT. Run anvil unlock so the agent can write the fix.")
		return 0
	}
	fmt.Fprintln(stdout, "Result: NOT PERFECT. Think about the missing points and answer again.")
	return 1
}

// runCheckKeys cross-checks the keys repository against the curriculum.
// Exit codes: 0 all good, 1 problems found, 2 setup error.
func runCheckKeys(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check-keys", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keysDir := fs.String("keys", os.Getenv("ANVIL_KEYS"), "keys repository directory")
	semantic := fs.Bool("semantic", false, "also judge the reference answers and points with the model")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprint(stderr, gradeUsage)
		return 2
	}
	if *keysDir == "" {
		fmt.Fprintln(stderr, "anvil check-keys: no keys directory (use --keys or set ANVIL_KEYS)")
		return 2
	}
	keys, err := grader.Load(*keysDir)
	if err != nil {
		fmt.Fprintf(stderr, "anvil check-keys: %v\n", err)
		return 2
	}
	problems := grader.Verify(keys, curriculum.Path())
	if *semantic {
		client, err := newGrader()
		if err != nil {
			fmt.Fprintf(stderr, "anvil check-keys: %v\n", err)
			return 2
		}
		problems = append(problems, grader.VerifySemantics(client, keys, curriculum.Path())...)
	}
	if len(problems) == 0 {
		mode := "offline checks"
		if *semantic {
			mode = "offline and semantic checks"
		}
		fmt.Fprintf(stdout, "OK: %d keys match %d stages (%s passed).\n", len(keys), len(curriculum.Path()), mode)
		return 0
	}
	for _, p := range problems {
		fmt.Fprintln(stdout, "PROBLEM:", p)
	}
	fmt.Fprintf(stdout, "%d problem(s)\n", len(problems))
	return 1
}

// findStep returns the stage at 1-based position number within a project.
func findStep(project string, number int) (curriculum.PathStep, bool) {
	p, ok := findProject(project)
	if !ok || number < 1 || number > len(p.Stages) {
		return curriculum.PathStep{}, false
	}
	for _, step := range curriculum.Path() {
		if step.Project.Slug == p.Slug && step.StageNumber == number {
			return step, true
		}
	}
	return curriculum.PathStep{}, false
}
