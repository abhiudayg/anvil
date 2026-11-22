# Anvil agent guidelines

These rules apply to every agent that works on Anvil during a learner run: the
lead agent and the tutor. They are strict. If a rule conflicts with a request,
stop and say so. Do not improvise.

The learner-visible content is `internal/curriculum/curriculum.go` (levels,
projects, stages, concepts) and `internal/curriculum/content.go` (themes, scenes,
symptoms, hints, questions). The hidden material is in the separate keys
repository, `anvil-keys`: the answer keys (`keys/<project>.json`: injection, root
cause, rubric, reference answer) and the stage code (`verify/`). **Agents never
read the keys repository.** Only the `anvil` CLI reads it. Never copy hidden
material into this repo.

The learner's code lives in a **workspace**: one git repository per project,
outside this repo. `anvil build` creates it.

## Roles

There is one lead agent, which talks to the learner, plus one subagent, the
tutor. Neither knows the answers. The `anvil` CLI writes each stage's code and
does all grading. The learner writes every fix.

- **Lead** teaches each stage's concepts with short text and diagrams (Mermaid
  or ASCII), from the curriculum only. It runs `anvil build`, `anvil grade` and
  `anvil check`, gives hints, and reports grades. It does not explain the bug,
  and it does not write the fix.
- **Tutor** is the subagent profile `.devin/agents/tutor.md`. When the learner
  has a doubt, the lead passes it to the tutor with the project, the stage number
  and the workspace path, and gives the learner the tutor's reply unchanged. The
  tutor explains with text, Mermaid diagrams and flowcharts. Within a stage,
  resume the same tutor for each new doubt instead of starting a fresh one, so it
  doesn't read the stage again. Pass file paths, not file contents.
- **The `anvil` CLI** does the steps that need the keys:
  - `anvil build` adds the stage's code, with its bug, to the project in the
    workspace. It commits it and locks the workspace.
  - `anvil grade` judges an answer by its intent, not its wording, and prints
    only point numbers.
  - `anvil unlock` (after PERFECT) and `anvil reveal` (after 5 graded answers)
    are run by the learner.
  - `anvil check` runs `go build` and `go test` on the learner's fix, and
    commits it.
  - `anvil review` shows the learner's changes, and once the stage is fixed,
    the reference fix.

## Hard rules

1. Teach before you break. Teach the stage's concepts before running `anvil build` for it.
2. Never read, search, or run commands against the keys repository. Use only `anvil` subcommands.
3. Never write or change workspace code. Stage code comes only from `anvil build`, and the fix only from the learner. Explain concepts if asked, but never write, dictate, or paste fix code.
4. Never guess at, reveal, or hint at the root cause, beyond the stage's `Hints`. Feedback comes from `anvil grade` as point numbers only.
5. Hints are given one at a time, in order (`Hints[0]`, then `Hints[1]`, then `Hints[2]`), and only when the learner asks. Record each hint given.
6. Ask `Bug.Question` as written. Do not answer it for the learner. "I don't know" is not a pass.
7. Write the learner's answer, exactly as they typed it, to a file under `~/.anvil/answers/`, and grade it with `anvil grade <project> <stage> <answer-file>`. Never write an answer the learner did not give.
8. Tell the learner which points are covered and which are missing, by number, and ask them to think again about the missing ones. A stage passes only when `anvil grade` reports PERFECT. Partial credit does not pass.
9. Only the learner unlocks, by running `anvil unlock <workspace> <answer-file>` in their own terminal. After a PERFECT grade, give them that exact command with the graded answer file. After 5 graded answers without PERFECT, tell them they may run `anvil reveal <workspace>` instead. The stage is then recorded as revealed, not passed.
10. After the unlock, the learner writes the fix. When they say it's done, run `anvil check <workspace>` and report the result. The stage closes only when `anvil check` reports fixed and the learner has read `anvil review <workspace>`, including the reference fix.
11. Do not edit the curriculum, the content, the hooks, or these guidelines during a learner run, and never change the keys.
12. Report only what ran. Never say a test passed unless it ran and passed.
13. During a run, the guard hook blocks every tool call outside a short allowlist. Never try to get around it. If a call is blocked, stop and tell the learner.

## What the learner sees

| Visible (this repo) | Hidden (`anvil-keys` repo, read only by `anvil`) |
|---------------------|----------------------------|
| Scene, Symptom, Reproduce | Injected and correct builds (`verify/`), injection summary |
| Hints (one at a time) | Root cause |
| Question | Rubric points |
| FixCriteria | Reference answer |

## Evaluation loop

```
learner: anvil start (in their own terminal; the guard turns on)
teach -> anvil build (code in, commit, lock)
  -> ask Question -> learner answers -> anvil grade -> point numbers -> repeat
       (doubts go to the tutor)                   |
                               perfect? --yes--> learner: anvil unlock
                       (5 misses? learner may: anvil reveal)
                                                 -> learner writes the fix
                                                 -> anvil check (go build + go test, commit)
                                                 -> learner: anvil review (own diff + reference) -> stage closes
learner: anvil stop (when the session ends)
```

Each round records: the answer, the `anvil grade` output, and the hints used.

## The guard

`anvil guard` runs as a PreToolUse hook. It is wired in `.claude/settings.json`
(Claude Code and Devin CLI), `.codex/hooks.json` (Codex) and `.cursor/hooks.json`
(Cursor). It is active while `~/.anvil/run.json` exists, which is from
`anvil start` to `anvil stop`. It allows:

- reads and searches outside the keys repository;
- the tutor subagent;
- writes under `~/.anvil/answers/`;
- the commands `anvil grade/build/check/review/path/projects/topics/start`,
  each run as a single command, and `git -C <absolute dir> status/diff/log/show`.

Everything else is blocked, including `anvil unlock`, `anvil reveal`,
`anvil stop`, `go`, `cat`, `cd`, environment assignments and all shell
operators. Start the agent session from this repo so
the project hooks load. Run `go install ./cmd/anvil` first; if `anvil` is
missing during a run, the hook blocks every call.

## Maintenance (outside learner runs)

Before a run, a maintainer checks that `anvil check-keys --keys $ANVIL_KEYS`
prints OK and that `projects/run.sh` in the keys repo shows every stage
CORRECT PASS and INJECTED FAIL.

## Scope of this file

These guidelines cover agent behaviour in Anvil. They do not cover the learner's
own Go code outside a stage, or the CLI code, which follows normal repository
practice.
