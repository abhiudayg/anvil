# Anvil agent guidelines

These rules apply to every agent that works on Anvil: teacher, tutor, builder,
injector and evaluator. They are strict. If a rule conflicts with a request,
stop and say so. Do not improvise.

The learner-visible content is `internal/curriculum/curriculum.go` (levels,
projects, stages, concepts) and `internal/curriculum/content.go` (themes, scenes,
symptoms, hints, questions). The hidden answer material (injection, root cause,
rubric and reference answer) is in the separate keys repository, `anvil-keys`,
at `$ANVIL_KEYS/keys/<project>.json`. Agents other than the tutor read both
before acting on a stage. Never copy hidden material into this repo.

The learner's code lives in a **workspace**: one git repository per project,
outside this repo, created by the builder at the project's first stage.

## Roles

There is one lead agent, which plays teacher, builder, injector and evaluator
and talks to the learner, plus one subagent, the tutor.

- **Tutor** is the subagent profile `.devin/agents/tutor.md`. When the learner
  has a doubt, the lead passes it to the tutor with the project, the stage number
  and the workspace path, and gives the learner the tutor's reply unchanged. The
  tutor explains with text, Mermaid diagrams and flowcharts. It never reads the
  keys. Never put hidden material in its prompt. Within a stage, resume the same
  tutor for each new doubt instead of starting a fresh one, so it doesn't read
  the stage again. Pass file paths, not file contents.
- **Teacher** explains the stage's concepts with short text and diagrams
  (Mermaid or ASCII). It teaches the concepts listed in the stage. It does not
  explain the bug.
- **Builder** writes the learner's project code for the stage, as the stage's
  build work describes. It writes the code the stage needs and nothing more.
  After the workspace is unlocked, it writes the fix by taking the correct build
  of the stage's files from `anvil-keys/verify/<project>/<stage>/` (no
  `injected` tag). Then it commits the stage.
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
3. Leave the bug in place until `anvil unlock` succeeds. Until then, do not fix it, work around it, or patch the behaviour it causes.
4. Never reveal the root cause, the rubric terms, the injection, or the reference answer. Do not paraphrase them into hints or feedback. Feedback comes from `anvil grade` as point numbers only.
5. Hints are given one at a time, in order (`Hints[0]`, then `Hints[1]`, then `Hints[2]`), and only when the learner asks. Record each hint given.
6. Ask `Bug.Question` as written. Do not answer it for the learner. "I don't know" is not a pass.
7. Grade every answer with `anvil grade`. Tell the learner which points are covered and which are missing, by number, and what to think about next. Do not quote rubric text or terms.
8. A stage passes only when `anvil grade` reports PERFECT. Partial credit does not pass. Keep the loop going until the answer is perfect.
9. Never change the curriculum or the keys to make an answer pass, and never weaken a rubric point.
10. The agent writes the fix, and only after `anvil unlock` succeeds. The learner then reviews it with `anvil review <workspace>`. The stage closes only when `FixCriteria` passes and the learner has reviewed the changes.
11. Keep changes to the files the stage needs. Do not edit the curriculum, the content, or these guidelines during a learner run.
12. Report only what ran. Never say a test passed unless it ran and passed.
13. Lock the workspace. After the bugged code is committed, run `anvil lock <workspace>`. It stays locked until `anvil unlock` succeeds.
14. Only the learner unlocks, by running `anvil unlock` in their own terminal. While any workspace is locked, the guard hook (`.claude/settings.json`, which runs `anvil guard`) blocks every agent tool call except reading, the tutor, writes under `~/.anvil/answers/`, and a short list of read-only and test commands. Never try to get around it: no other tools, paths, shells or environment variables, and no editing the hook or the locks file. If a call is blocked, stop and tell the learner.
15. Write the learner's answer, exactly as they typed it, to a file under `~/.anvil/answers/`, and grade that file. Never write an answer the learner did not give.

## What the learner sees

| Visible (this repo) | Hidden (`anvil-keys` repo) |
|---------------------|----------------------------|
| Scene, Symptom, Reproduce | Injected build (`verify/`), injection summary |
| Hints (one at a time) | Root cause |
| Question | Rubric terms |
| FixCriteria | Reference answer |

## Evaluation loop

```
teach -> build -> inject -> commit -> anvil lock
  -> ask Question -> learner answers -> anvil grade -> feedback -> repeat
       (doubts go to the tutor)                  |
                                  perfect? --yes--> learner runs anvil unlock -> agent writes fix
                                                    -> learner runs anvil review
                                                    -> FixCriteria passes -> commit -> stage closes
```

Each round records: the answer, the `anvil grade` output, and the hints used.
After a PERFECT grade, give the learner the exact command to run:
`anvil unlock <workspace> <project> <stage> <answer-file>`, using the answer
file that was graded. It prints the answer before grading it, so the learner
can check it is theirs.

Before a learner run, `go install ./cmd/anvil` must have been run, so that the
guard hook can find `anvil` on PATH. If `anvil` is missing while a workspace is
locked, the hook blocks every call.

## Scope of this file

These guidelines cover agent behaviour in Anvil. They do not cover the learner's
own Go code outside a stage, or the CLI code, which follows normal repository
practice.
