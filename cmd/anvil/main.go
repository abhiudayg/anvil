// Command anvil is the terminal interface for learning Go by building projects.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"anvil/internal/curriculum"
)

const usage = `anvil - learn Go by building projects

Usage:
  anvil path                  Show the full learning order, level by level
  anvil projects              List the build projects
  anvil topics [project]      List the concepts of each project, by stage
  anvil grade [--keys DIR] <project> <stage> <answer-file>
                              Grade an answer deterministically against the keys
  anvil check-keys [--keys DIR]
                              Cross-check the keys repository against the curriculum
  anvil start [--keys DIR]    Begin a run: restricts agent tool calls
  anvil stop                  End the run and unlock every workspace
  anvil build <workspace> <project> <stage>
                              Copy the stage into the workspace and lock it
  anvil unlock <workspace> <answer-file>
                              Grade the answer and unlock the workspace on PERFECT
  anvil reveal <workspace>    Show the reference answer after 5 graded tries
  anvil check <workspace>     Test the learner's fix and commit it
  anvil review <workspace>    Show the learner's diff, then the reference fix
  anvil guard                 Hook entry point: restricts agent tool calls while a run is active
  anvil help                  Show this help
`

func main() {
	os.Exit(dispatch(os.Args[1:], os.Stdout, os.Stderr))
}

func dispatch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	case "path":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "usage: anvil path")
			return 2
		}
		return listPath(stdout)
	case "projects":
		return listProjects(stdout)
	case "grade":
		return runGrade(args[1:], stdout, stderr)
	case "check-keys":
		return runCheckKeys(args[1:], stdout, stderr)
	case "start":
		return runStart(args[1:], stdout, stderr)
	case "stop":
		return runStop(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "unlock":
		return runUnlock(args[1:], stdout, stderr)
	case "reveal":
		return runReveal(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "review":
		return runReview(args[1:], stdout, stderr)
	case "guard":
		return runGuard(args[1:], stdout, stderr)
	case "topics":
		if len(args) > 2 {
			fmt.Fprintln(stderr, "usage: anvil topics [project]")
			return 2
		}
		if len(args) == 2 {
			return listTopics(args[1], stdout, stderr)
		}
		for _, p := range curriculum.Projects {
			listTopics(p.Slug, stdout, stderr)
		}
		return 0
	default:
		fmt.Fprintf(stderr, "anvil: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func listPath(stdout io.Writer) int {
	current := 0
	for i, step := range curriculum.Path() {
		if step.Level.Number != current {
			current = step.Level.Number
			fmt.Fprintf(stdout, "\nLevel %d: %s\n  %s\n", current, step.Level.Title, step.Level.Goal)
		}
		fmt.Fprintf(stdout, "\n  %2d. [%s] %s (%d concepts)\n", i+1, step.Project.Slug, step.Stage.Title, len(step.Stage.Concepts))
		for _, cp := range step.Stage.Concepts {
			fmt.Fprintf(stdout, "        - %s\n", cp.Name)
		}
	}
	fmt.Fprintln(stdout)
	return 0
}

func listProjects(stdout io.Writer) int {
	for _, p := range curriculum.Projects {
		fmt.Fprintf(stdout, "%-8s %s\n         %s\n         %d stages, %d concepts\n\n",
			p.Slug, p.Name, p.Summary, len(p.Stages), p.TopicCount())
	}
	return 0
}

func listTopics(slug string, stdout, stderr io.Writer) int {
	p, ok := findProject(slug)
	if !ok {
		fmt.Fprintf(stderr, "anvil: unknown project %q (try: anvil projects)\n", slug)
		return 1
	}
	fmt.Fprintf(stdout, "%s (%d concepts)\n", p.Name, p.TopicCount())
	if th, ok := curriculum.ThemeFor(p.Slug); ok {
		fmt.Fprintf(stdout, "Theme: %s. %s\n", th.Name, th.Story)
	}
	for i, s := range p.Stages {
		fmt.Fprintf(stdout, "\nStage %d (level %d): %s\n", i+1, s.Level, s.Title)
		if sc, ok := curriculum.ContentFor(p.Slug, s.Title); ok {
			fmt.Fprintf(stdout, "  %s\n", sc.Scene)
		}
		for _, cp := range s.Concepts {
			fmt.Fprintf(stdout, "  - %s\n", cp.Name)
		}
	}
	fmt.Fprintln(stdout)
	return 0
}

func findProject(slug string) (curriculum.Project, bool) {
	for _, p := range curriculum.Projects {
		if strings.EqualFold(p.Slug, slug) {
			return p, true
		}
	}
	return curriculum.Project{}, false
}
