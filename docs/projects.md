# Anvil: projects and learning path

The source of truth is code, not this file:

- `internal/curriculum/curriculum.go`: levels, projects, stages and concepts
- `internal/curriculum/k8s.go`: the Kubernetes project
- `internal/curriculum/content.go`: themes, scenes and learner-visible bug fields
- `internal/grader`: the deterministic grader and the key cross-check
- `AGENTS.md`: the rules every agent follows
- `anvil-keys` repo (separate, at `../anvil-keys`): the hidden injections, root causes, rubric terms and reference answers

`anvil path` prints the full learning order. `anvil topics <project>` prints one
project's theme, stages and concepts. `go test ./...` checks the plan.

**Totals:** 3 projects, 31 stages, 396 concepts, 12 levels, 31 detailed bugs.

## Themes

| Project | Theme | Story |
|---------|-------|-------|
| Redis | The Lighthouse Ledger | Keep a lighthouse logbook whole, safe and truthful. |
| Job Queue | The Dispatch Office | Take delivery jobs, send couriers, never lose a parcel. |
| Kubernetes | The Fleet Harbour | Keep a fleet matching the register, as a harbour master. |

## Levels

| Level | Title |
|------:|-------|
| 1 | Foundations |
| 2 | Types, interfaces and design |
| 3 | HTTP services |
| 4 | Concurrency |
| 5 | Storage |
| 6 | Quality and performance |
| 7 | Networking and RPC |
| 8 | Advanced Go and shipping |
| 9 | Kubernetes API and clients |
| 10 | Kubernetes workloads and networking |
| 11 | Kubernetes operators |
| 12 | Kubernetes in production |

## How a stage runs

Each stage follows **Teach → Build → Break → Lock → Ask → Evaluate → Unlock → Fix → Review**.

1. **Teach:** the teacher covers the stage's concepts.
2. **Build:** the builder writes the stage's code in the learner's workspace, a
   git repository with one per project.
3. **Break:** the injector applies the stage's bug, and the bug stays in place.
4. **Lock:** the bugged code is committed, and `anvil lock <workspace>` makes
   the workspace read-only.
5. **Ask:** the learner gets the scene and the symptom, then answers the bug's
   question. Hints are given one at a time, on request. The learner can ask the
   tutor subagent (`.devin/agents/tutor.md`) about any doubt. The tutor answers
   with text and diagrams and never sees the keys.
6. **Evaluate:** each answer is graded against the rubric. The loop repeats until
   the answer is perfect.
7. **Unlock:** the learner runs `anvil unlock <workspace> <project> <stage> <answer-file>`
   in their own terminal. It prints the answer, grades it again, and restores
   write access only if it is PERFECT.
8. **Fix:** the agent writes the fix from the stage's correct build.
9. **Review:** `anvil review <workspace>` shows the learner every change since
   the bugged commit. The stage closes when the fix criteria pass and the
   learner has reviewed it.

The lock has two layers. `anvil lock` removes write permission from the
workspace and records it in `~/.anvil/locks`. While anything is locked, a
PreToolUse hook in `.claude/settings.json` runs `anvil guard` on every agent
tool call. Devin CLI and Claude Code both load that file. The guard denies by
default. It allows reading, the tutor subagent, writes under `~/.anvil/answers/`,
and a short list of read-only and test commands (`go test`, `go vet`,
`git status/diff/log/show`, `anvil grade/review`, `ls`, `cat` and similar),
checked word by word. Shell operators other than `&&`, `$`, other programs and
`anvil unlock` are all blocked, so only the learner can unlock.

Limits: code the agent wrote before the lock still runs when `go test` runs it,
and environment variables already exported in a running shell stay in effect.
An agent with no hook support is bound only by `AGENTS.md`.

Rules for agents are in `AGENTS.md`. Answers are graded by `anvil grade`. A
model judges each rubric point by meaning, at temperature 0. Each judgement is
cached by a hash of the model, prompt, stage, rubric and answer, so the same
answer always gets the same grade. It needs `ANTHROPIC_API_KEY`.

## Bugs

The learner sees each bug's symptom, reproduce command, hints, question and fix
criteria in `content.go`. The hidden parts (injection, root cause, rubric and
reference answer) are in `anvil-keys`.

`anvil check-keys --keys $ANVIL_KEYS` cross-checks every stage offline:

- the key exists, with the matching stage number and concept;
- the bug's concept is taught by that stage or an earlier one;
- the injection and root cause are filled in, and the rubric has at least 3 points;
- the reference answer has exactly one sentence per point.

`anvil check-keys --semantic` adds the judge-backed checks: the reference answer
is perfect, "I don't know" and an empty answer fail, and removing a point's own
sentence uncovers that point.

## Bug verification

`verify/` is a separate Go module with one package per stage. Each package has
the stage's code in two builds: the correct one, and one with the injected bug,
selected by the `injected` build tag. `verify/run.sh` checks that every stage's
reproduce test passes on the correct build and fails on the injected build.

## What the curriculum deliberately leaves out

"All of Go" is not a finite list, so the plan covers the language, the standard
library areas used for backend and CLI work, and the ecosystem libraries the
projects use. These are left out on purpose:

- Go assembly (Plan 9 syntax) and compiler or linker internals beyond reading output.
- WebAssembly and GopherJS targets.
- Platform-specific syscalls and Windows service APIs.
- Raw sockets, UDP-level networking and HTTP/3.
- The experimental `testing/synctest` package.
- Non-Kubernetes container platforms and service meshes.
- Cloud-provider SDKs.

Any of these can become a stage later.

## Known limits

- The grader matches terms, not meaning. An answer that uses the right words
  without understanding can pass. The per-point check keeps terms specific, but
  it doesn't stop keyword stuffing.
- The bug reproduce commands can't be run yet. The builder code for the stages
  doesn't exist, so the symptoms are unverified until it does.
