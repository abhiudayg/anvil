package grader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"anvil/internal/curriculum"
)

// VerifyProjects checks the on-disk project layout the CLI builds from:
// every project has base/go.mod, and every stage has non-empty correct/
// and injected/ overlays where each injected file exists in correct/.
func VerifyProjects(keysDir string, steps []curriculum.PathStep) []string {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}
	seen := map[string]bool{}
	for _, step := range steps {
		slug := step.Project.Slug
		where := fmt.Sprintf("%s stage %d (%s)", slug, step.StageNumber, step.Stage.Title)
		root := filepath.Join(keysDir, "projects", slug)
		if !seen[slug] {
			seen[slug] = true
			if _, err := os.Stat(filepath.Join(root, "base", "go.mod")); err != nil {
				add("%s: base/go.mod is missing", slug)
			}
		}
		for _, overlay := range []string{"correct", "injected"} {
			dir := filepath.Join(root, "stages", fmt.Sprintf("%02d", step.StageNumber), overlay)
			empty := true
			err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					empty = false
				}
				if overlay == "injected" && !d.IsDir() {
					rel, err := filepath.Rel(dir, path)
					if err != nil {
						return err
					}
					counterpart := filepath.Join(root, "stages", fmt.Sprintf("%02d", step.StageNumber), "correct", rel)
					if _, err := os.Stat(counterpart); err != nil {
						add("%s: injected file %s has no correct counterpart", where, rel)
					}
				}
				return nil
			})
			if err != nil || empty {
				add("%s: stages/%02d/%s is missing or empty", where, step.StageNumber, overlay)
			}
		}
	}
	return problems
}
