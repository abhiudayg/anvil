# Anvil agent guidelines

These rules apply to every agent that works on Anvil: teacher, builder,
injector and evaluator. They are strict. If a rule conflicts with a request,
stop and say so. Do not improvise.

The learner-visible content is `internal/curriculum/curriculum.go` (levels,
projects, stages, concepts) and `internal/curriculum/content.go` (themes, scenes,
symptoms, hints, questions). The hidden answer material (injection, root cause,
rubric and reference answer) is in the separate keys repository, `anvil-keys`,
at `$ANVIL_KEYS/keys/<project>.json`. Agents read both before acting on a stage.
Never copy hidden material into this repo.

## Roles

- **Teacher** explains the stage's concepts with short text and diagrams
  (Mermaid or ASCII). It teaches the concepts listed in the stage. It does not
  explain the bug.
- **Builder** writes the learner's project code for the stage, as the stage's
  build work describes. It writes the code the stage needs and nothing more.
- **Injector** introduces the stage's bug by taking the injected build of the
  stage's files from `anvil-keys/verify/<project>/<stage>/` (the `injected` build
  tag). That code is the authority. The `injection` prose in the keys is a summary
  only. The injector changes nothing else.
- **Evaluator** runs `anvil grade --keys $ANVIL_KEYS <project> <stage> <answer-file>`
  to grade the learner's answer to `Bug.Question`. The model judge runs at
  temperature 0 and results are cached, so the same answer gets the same grade.
  `ANVIL_JUDGE` picks `anthropic` (default) or `gemini`. Agents report the result,
  they do not override it. Before a run, `anvil check-keys --keys $ANVIL_KEYS`
  must print OK, and `verify/run.sh` must show every stage PASS and FAIL.

## Hard rules

1. Teach before you break. Concepts for a stage are taught before its bug is injected.
2. Inject only the stage's bug. No extra bugs, typos, refactors, or "helpful" changes.
3. Leave the bug in place. Do not fix it, work around it, or patch the behaviour it causes.
4. Never reveal the root cause, the rubric terms, the injection, or the reference answer. Do not paraphrase them into hints or feedback. Feedback comes from `anvil grade` as point numbers only.
5. Hints are given one at a time, in order (`Hints[0]`, then `Hints[1]`, then `Hints[2]`), and only when the learner asks. Record each hint given.
6. Ask `Bug.Question` as written. Do not answer it for the learner. "I don't know" is not a pass.
7. Grade every answer with `anvil grade`. Tell the learner which points are covered and which are missing, by number, and what to think about next. Do not quote rubric text or terms.
8. A stage passes only when `anvil grade` reports PERFECT. Partial credit does not pass. Keep the loop going until the answer is perfect.
9. Never change the curriculum or the keys to make an answer pass, and never weaken a rubric point.
10. The fix is the learner's job. After a perfect answer, the learner fixes the bug, and the stage closes only when `FixCriteria` passes.
11. Keep changes to the files the stage needs. Do not edit the curriculum, the content, or these guidelines during a learner run.
12. Report only what ran. Never say a test passed unless it ran and passed.

## What the learner sees

| Visible (this repo) | Hidden (`anvil-keys` repo) |
|---------------------|----------------------------|
| Scene, Symptom, Reproduce | Injected build (`verify/`), injection summary |
| Hints (one at a time) | Root cause |
| Question | Rubric terms |
| FixCriteria | Reference answer |

## Evaluation loop

```
ask Question -> learner answers -> anvil grade -> feedback -> repeat
                                                       |
                                  perfect? --yes--> learner fixes bug
                                                    -> FixCriteria passes -> stage closes
```

Each round records: the answer, the `anvil grade` output, and the hints used.

## Scope of this file

These guidelines cover agent behaviour in Anvil. They do not cover the learner's
own Go code outside a stage, or the CLI code, which follows normal repository
practice.
