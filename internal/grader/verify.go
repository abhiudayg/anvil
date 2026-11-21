package grader

import (
	"fmt"
	"strings"

	"anvil/internal/curriculum"
)

// nonAnswers must never pass. The semantic check runs each of them through the judge.
var nonAnswers = []string{"", "I don't know", "no idea, skipping this one"}

// Verify runs the offline checks: every stage has a matching key, its concept
// is taught by then, the hidden fields are filled in, the rubric has at least
// three points, and the reference answer has one sentence per point.
func Verify(keys map[string]Key, steps []curriculum.PathStep) []string {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	taught := map[string]bool{}
	seen := map[string]bool{}
	for _, step := range steps {
		for _, cp := range step.Stage.Concepts {
			taught[cp.ID] = true
		}
		id := step.Project.Slug + "/" + step.Stage.Title
		seen[id] = true
		where := fmt.Sprintf("%s stage %d (%s)", step.Project.Slug, step.StageNumber, step.Stage.Title)

		key, ok := Lookup(keys, step.Project.Slug, step.Stage.Title)
		if !ok {
			add("%s: no key", where)
			continue
		}
		content, _ := curriculum.ContentFor(step.Project.Slug, step.Stage.Title)

		if key.Number != step.StageNumber {
			add("%s: key says stage %d", where, key.Number)
		}
		if key.Concept != content.Bug.Concept {
			add("%s: key concept %q differs from bug concept %q", where, key.Concept, content.Bug.Concept)
		}
		if !taught[key.Concept] {
			add("%s: bug concept %q is not taught by this stage or earlier", where, key.Concept)
		}
		if strings.TrimSpace(key.Injection) == "" {
			add("%s: injection is empty", where)
		}
		if strings.TrimSpace(key.RootCause) == "" {
			add("%s: root cause is empty", where)
		}
		if len(key.Rubric) < 3 {
			add("%s: rubric has %d points, want at least 3", where, len(key.Rubric))
		}
		for i, p := range key.Rubric {
			if strings.TrimSpace(p.Name) == "" {
				add("%s: rubric point %d is empty", where, i+1)
			}
		}
		if strings.TrimSpace(key.ReferenceAnswer) == "" {
			add("%s: reference answer is empty", where)
		} else if n := len(sentences(key.ReferenceAnswer)); n != len(key.Rubric) {
			add("%s: reference answer has %d sentences, want one per rubric point (%d)", where, n, len(key.Rubric))
		}
	}

	for _, id := range SortedIDs(keys) {
		if !seen[id] {
			add("key %q does not match any curriculum stage", id)
		}
	}
	return problems
}

// VerifySemantics runs the checks that need the judge. It returns one line per
// problem. Each stage costs one judgement per point plus the non-answers, so it
// is run on demand, not on every build.
func VerifySemantics(c *Client, keys map[string]Key, steps []curriculum.PathStep) []string {
	var problems []string
	for _, step := range steps {
		key, ok := Lookup(keys, step.Project.Slug, step.Stage.Title)
		if !ok || len(key.Rubric) == 0 {
			continue // reported by Verify
		}
		where := fmt.Sprintf("%s stage %d (%s)", step.Project.Slug, step.StageNumber, step.Stage.Title)

		res, err := c.Grade(key, key.ReferenceAnswer)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: judging the reference answer failed: %v", where, err))
			continue
		}
		if !res.Perfect {
			problems = append(problems, fmt.Sprintf("%s: reference answer is not perfect; missing points %v", where, res.Missing()))
		}
		for _, bad := range nonAnswers {
			r, err := c.Grade(key, bad)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: judging non-answer failed: %v", where, err))
				continue
			}
			if r.Perfect {
				problems = append(problems, fmt.Sprintf("%s: non-answer %q passes", where, bad))
			}
		}
		sents := sentences(key.ReferenceAnswer)
		for i := range sents {
			var rest []string
			for j, s := range sents {
				if j != i {
					rest = append(rest, s)
				}
			}
			r, err := c.Grade(key, strings.Join(rest, " ")+".")
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: judging point %d without its sentence failed: %v", where, i+1, err))
				continue
			}
			if r.Covered[i] {
				problems = append(problems, fmt.Sprintf("%s: point %d is still covered without its own sentence", where, i+1))
			}
		}
	}
	return problems
}

// sentences splits a reference answer into sentences.
func sentences(text string) []string {
	body := strings.TrimSuffix(strings.TrimSpace(text), ".")
	if body == "" {
		return nil
	}
	return strings.Split(body, ". ")
}
