package curriculum

import (
	"strings"
	"testing"
)

func TestEveryProjectHasTheme(t *testing.T) {
	for _, p := range Projects {
		th, ok := ThemeFor(p.Slug)
		if !ok {
			t.Errorf("project %q has no theme", p.Slug)
			continue
		}
		if strings.TrimSpace(th.Name) == "" || strings.TrimSpace(th.Story) == "" {
			t.Errorf("project %q has an empty theme name or story", p.Slug)
		}
	}
}

func TestEveryStageHasSceneAndCompleteBug(t *testing.T) {
	for _, p := range Projects {
		for _, s := range p.Stages {
			sc, ok := ContentFor(p.Slug, s.Title)
			if !ok {
				t.Errorf("stage %q in %q has no content", s.Title, p.Slug)
				continue
			}
			if strings.TrimSpace(sc.Scene) == "" {
				t.Errorf("stage %q has no scene", s.Title)
			}
			checkBug(t, s.Title, sc.Bug)
		}
	}
}

func checkBug(t *testing.T, stage string, b Bug) {
	t.Helper()
	fields := map[string]string{
		"Concept":     b.Concept,
		"Symptom":     b.Symptom,
		"Reproduce":   b.Reproduce,
		"Question":    b.Question,
		"FixCriteria": b.FixCriteria,
	}
	for name, v := range fields {
		if strings.TrimSpace(v) == "" {
			t.Errorf("stage %q: bug field %s is empty", stage, name)
		}
	}
	if len(b.Hints) != 3 {
		t.Errorf("stage %q: want exactly 3 hints, got %d", stage, len(b.Hints))
	}
	for i, h := range b.Hints {
		if strings.TrimSpace(h) == "" {
			t.Errorf("stage %q: hint %d is empty", stage, i+1)
		}
	}
}

// TestBugConceptIsTaughtByThenStage checks that every bug tests a concept that
// the path has already taught by the end of the bug's stage.
func TestBugConceptIsTaughtByItsStage(t *testing.T) {
	taught := map[string]bool{}
	for _, step := range Path() {
		for _, cp := range step.Stage.Concepts {
			taught[cp.ID] = true
		}
		sc, ok := ContentFor(step.Project.Slug, step.Stage.Title)
		if !ok {
			continue // reported by TestEveryStageHasSceneAndCompleteBug
		}
		if !taught[sc.Bug.Concept] {
			t.Errorf("stage %q: bug concept %q is not taught by this stage or earlier", step.Stage.Title, sc.Bug.Concept)
		}
	}
}

func TestNoOrphanContent(t *testing.T) {
	stages := map[string]bool{}
	for _, p := range Projects {
		for _, s := range p.Stages {
			stages[p.Slug+"/"+s.Title] = true
		}
	}
	for _, k := range stageContentKeys() {
		if !stages[k] {
			t.Errorf("content %q does not match any stage title", k)
		}
	}
}

func TestBugsAreUniqueAcrossStages(t *testing.T) {
	seen := map[string]string{}
	for _, k := range stageContentKeys() {
		sc := stageContents[k]
		if prev, ok := seen[sc.Bug.Symptom]; ok {
			t.Errorf("symptom of %q duplicates %q", k, prev)
		}
		seen[sc.Bug.Symptom] = k
	}
}

func TestKubernetesCoverageDepth(t *testing.T) {
	var k8s Project
	for _, p := range Projects {
		if p.Slug == "k8s" {
			k8s = p
		}
	}
	if k8s.Slug == "" {
		t.Fatal("k8s project missing")
	}
	if len(k8s.Stages) < 8 {
		t.Errorf("k8s stages = %d, want at least 8", len(k8s.Stages))
	}
	if n := k8s.TopicCount(); n < 80 {
		t.Errorf("k8s concepts = %d, want at least 80", n)
	}
}
