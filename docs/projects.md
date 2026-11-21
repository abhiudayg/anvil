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

A learner run starts with `anvil start --keys <anvil-keys>` and ends with
`anvil stop`. The learner runs both in their own terminal. Each stage follows
**Teach → Build → Ask → Evaluate → Unlock → Fix → Review**.

1. **Teach:** the agent covers the stage's concepts from the curriculum.
2. **Build:** `anvil build <workspace> <project> <stage>` copies the stage's
   injected build from `verify/` into the workspace (one git repository per
   project). It commits it and makes the workspace read-only. The agent never
   sees the keys.
3. **Ask:** the learner gets the scene and the symptom, then answers the bug's
   question. Hints are given one at a time, on request. The learner can ask the
   tutor subagent (`.devin/agents/tutor.md`) about any doubt. The tutor answers
   with text and diagrams.
4. **Evaluate:** `anvil grade` judges the answer and the agent reports which
   points are covered, by number. The loop repeats until the answer is perfect.
5. **Unlock:** the learner runs `anvil unlock <workspace> <answer-file>` in their
   own terminal. It prints the answer, grades it again, and restores write
   access only if it is PERFECT.
6. **Fix:** `anvil fix <workspace>` writes the correct build, then runs
   `go build ./...` and `go test -race ./<stage>/...`, and commits. A skipped
   test means the fix is not verified, and the stage stays open.
7. **Review:** `anvil review <workspace>` shows the learner every change since
   the build commit.

While a run is active (`~/.anvil/run.json` exists), a PreToolUse hook runs
`anvil guard` on every agent tool call. Claude Code and Devin CLI load it from
`.claude/settings.json`, Codex from `.codex/hooks.json`, and Cursor from
`.cursor/hooks.json`. The guard denies by default. It allows:

- reads and searches outside the keys repository;
- the tutor subagent;
- writes under `~/.anvil/answers/`;
- `anvil grade/build/fix/review` and the listing commands;
- `git -C <dir> status/diff/log/show`;
- `cat/head/tail/wc/ls` on absolute paths outside the keys repository.

Every command is checked word by word. `go`, `cd`, `$`, shell operators other
than `&&`, other programs, and `anvil unlock` and `anvil stop` are all blocked.

Limits:

- An agent with no hook support, or a session started outside this repo (so the
  project hooks don't load), is bound only by `AGENTS.md`.
- Codex runs project hooks only after the user trusts them.
- The Cursor `Read` and `Write` input field names haven't been checked. If they
  differ, those calls are blocked, not allowed.
- The agent can still read the bugged workspace code, as the learner does, so it
  could spot the bug itself. The rules forbid hinting at it.

Rules for agents are in `AGENTS.md`. A model judges each rubric point by the
answer's intent: paraphrase and different wording count, and naming terms
without the meaning does not. It runs at temperature 0. Each judgement is cached
by a hash of the provider, model, endpoint, prompt, stage, rubric and answer, so
the same answer always gets the same grade. It needs `ANTHROPIC_API_KEY` (or
`GEMINI_API_KEY` with `ANVIL_JUDGE=gemini`).

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

- Stage code is the per-stage package from `verify/`, not one growing project.
  Each stage is its own directory in the workspace.
- `jobq/j06` needs `ANVIL_TEST_PG_DSN` and `k8s/k8` needs `KUBEBUILDER_ASSETS`.
  Without them their tests skip, and `anvil fix` reports the fix as not verified.
