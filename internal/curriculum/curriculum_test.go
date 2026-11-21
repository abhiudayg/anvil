package curriculum

import (
	"strings"
	"testing"
)

// minConcepts is the coverage floor for the whole curriculum.
const minConcepts = 300

func TestConceptCountMeetsFloor(t *testing.T) {
	total := 0
	for _, p := range Projects {
		total += p.TopicCount()
	}
	if total < minConcepts {
		t.Errorf("total concepts = %d, want at least %d", total, minConcepts)
	}
}

func TestConceptIDsAndNamesAreUnique(t *testing.T) {
	ids := map[string]string{}
	names := map[string]string{}
	for _, step := range Path() {
		for _, cp := range step.Stage.Concepts {
			if strings.TrimSpace(cp.ID) == "" || strings.TrimSpace(cp.Name) == "" {
				t.Errorf("stage %q has a concept with an empty id or name", step.Stage.Title)
				continue
			}
			if prev, ok := ids[cp.ID]; ok {
				t.Errorf("concept id %q appears in %q and %q", cp.ID, prev, step.Stage.Title)
			}
			ids[cp.ID] = step.Stage.Title

			key := strings.ToLower(cp.Name)
			if prev, ok := names[key]; ok {
				t.Errorf("concept name %q appears in %q and %q", cp.Name, prev, step.Stage.Title)
			}
			names[key] = step.Stage.Title
		}
	}
}

// TestPrerequisitesComeFirst checks the path order: every concept's Needs
// must be introduced earlier in the flattened path.
func TestPrerequisitesComeFirst(t *testing.T) {
	position := map[string]int{}
	pos := 0
	for _, step := range Path() {
		for _, cp := range step.Stage.Concepts {
			position[cp.ID] = pos
			pos++
		}
	}

	pos = 0
	for _, step := range Path() {
		for _, cp := range step.Stage.Concepts {
			for _, need := range cp.Needs {
				p, ok := position[need]
				if !ok {
					t.Errorf("concept %q needs unknown concept %q", cp.ID, need)
					continue
				}
				if p >= pos {
					t.Errorf("concept %q (%s) needs %q, which is taught later", cp.ID, step.Stage.Title, need)
				}
			}
			pos++
		}
	}
}

func TestPathIsOrderedByLevel(t *testing.T) {
	last := 0
	for _, step := range Path() {
		if step.Level.Number < last {
			t.Errorf("stage %q (level %d) follows a level %d stage", step.Stage.Title, step.Level.Number, last)
		}
		last = step.Level.Number
	}
}

func TestEveryStageHasLevelAndConcepts(t *testing.T) {
	levels := map[int]bool{}
	for _, lv := range Levels {
		levels[lv.Number] = true
	}
	covered := map[int]bool{}
	for _, p := range Projects {
		for _, s := range p.Stages {
			if strings.TrimSpace(s.Title) == "" {
				t.Errorf("project %q has a stage with no title", p.Slug)
			}
			if !levels[s.Level] {
				t.Errorf("stage %q has unknown level %d", s.Title, s.Level)
			}
			if len(s.Concepts) == 0 {
				t.Errorf("stage %q has no concepts", s.Title)
			}
			covered[s.Level] = true
		}
	}
	for _, lv := range Levels {
		if !covered[lv.Number] {
			t.Errorf("level %d (%s) has no stages", lv.Number, lv.Title)
		}
	}
}

func TestProjectSlugsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Projects {
		if seen[p.Slug] {
			t.Errorf("duplicate project slug %q", p.Slug)
		}
		seen[p.Slug] = true
		if len(p.Stages) == 0 {
			t.Errorf("project %q has no stages", p.Slug)
		}
	}
}
