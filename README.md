# Anvil

Learn Go by building real tools and debugging them.

Anvil walks you through three projects, stage by stage. Each stage adds real
code to a growing codebase, with one bug in it on purpose. You find the bug,
explain it in your own words, then write the fix. An AI agent teaches the
concepts and answers your questions, but it never sees the answers, and it
can't unlock the code for you.

| Project | You build | Stages | Concepts |
|---------|-----------|-------:|---------:|
| `redis` | A RESP-speaking, concurrent, persistent in-memory key-value server | 10 | 185 |
| `jobq`  | An HTTP job API backed by Postgres, with a worker pool, retries, realtime progress, gRPC and Connect services, and a CLI | 13 | 120 |
| `k8s`   | A Kubernetes operator for the job queue, built with controller-runtime | 8 | 91 |

The 31 stages cover 12 levels, from language foundations to Kubernetes in
production. Run `anvil path` to see the full learning order.

## How a stage works

```
teach -> build (code with a bug, locked) -> you explain the bug -> graded by intent
      -> PERFECT? you unlock -> you write the fix -> anvil check runs the tests
      -> anvil review: your diff, then the reference fix
```

1. **Teach.** The agent explains the stage's concepts with short text and
   diagrams.
2. **Build.** `anvil build` adds the stage's code, bug included, to your
   project. It commits it and makes the workspace read-only.
3. **Explain.** You get the scene, the symptom and a question. You answer in
   your own words. A model judge grades each rubric point by meaning, not
   wording: paraphrase counts, keyword-dropping doesn't. You see which points
   you've covered, by number, and try again until every point is covered.
   Hints come one at a time, when you ask.
4. **Unlock.** You run `anvil unlock` in your own terminal. It grades your
   answer again and unlocks only on PERFECT. If you're stuck after 5 graded
   answers, `anvil reveal` shows the reference answer, and the stage is
   recorded as revealed, not passed.
5. **Fix.** You write the fix. `anvil check` runs `go build ./...` and
   `go test -race ./...`, and commits your fix when they pass.
6. **Review.** `anvil review` shows your changes, then the reference fix, so you
   can compare.

Questions go to a **tutor** subagent at any point. It explains with text,
Mermaid diagrams and flowcharts, but it won't point at the bug or write the fix.

## Requirements

- Go 1.27 or later, and git.
- The **keys repository** (`anvil-keys`). It holds the stage code, the answer
  keys and the rubrics. It's kept separate and private so the answers aren't
  public.
- An API key for the judge: `ANTHROPIC_API_KEY` (the default), or
  `GEMINI_API_KEY` with `ANVIL_JUDGE=gemini`. Alternatively
  `ANVIL_JUDGE=claude-cli` uses your logged-in Claude Code CLI (a Claude
  subscription works). Its temperature can't be pinned to 0, but every grade
  is cached, so a repeated answer gets the same grade.
- An agent that supports PreToolUse hooks: Claude Code, Devin CLI, Codex or
  Cursor.

## Install

```sh
git clone git@github.com:abhiudayg/anvil.git
cd anvil
go install ./cmd/anvil
export PATH="$(go env GOPATH)/bin:$PATH"   # the guard hook needs anvil on PATH
```

## Quick start

In your own terminal:

```sh
anvil start --keys /path/to/anvil-keys
```

Then start your agent **from this repo**, so it loads `AGENTS.md` and the
hooks, and ask it to begin a stage, for example: "start redis stage 1 in
~/anvil-work/redis".

When the agent tells you your answer is PERFECT, unlock in your own terminal:

```sh
anvil unlock ~/anvil-work/redis ~/.anvil/answers/<answer-file>
```

Write your fix. Then ask the agent to run `anvil check`, and read
`anvil review`. When you're done for the day:

```sh
anvil stop
```

## Commands

| Command | Who runs it | What it does |
|---------|-------------|--------------|
| `anvil path` | anyone | The full learning order, level by level |
| `anvil projects` | anyone | The build projects |
| `anvil topics [project]` | anyone | Each project's concepts, by stage |
| `anvil start [--keys DIR]` | learner | Begin a run and turn the guard on |
| `anvil stop` | learner | End the run and unlock every workspace |
| `anvil build <ws> <project> <stage>` | agent | Add the stage's code, with its bug, and lock the workspace |
| `anvil grade <project> <stage> <answer-file>` | agent | Grade an answer and print the covered and missing points |
| `anvil unlock <ws> <answer-file>` | learner | Grade again; unlock on PERFECT |
| `anvil reveal <ws>` | learner | After 5 graded tries, show the reference answer and unlock |
| `anvil check <ws>` | either | Build and test the fix, and commit it |
| `anvil review <ws>` | either | Show the learner's diff, then the reference fix |
| `anvil check-keys [--keys DIR]` | maintainer | Cross-check the keys repo against the curriculum |
| `anvil guard` | hook | Decide whether an agent tool call is allowed |
| `anvil tui [--repo DIR] [--agent claude\|gemini]` | learner | Chat with the lead agent; every command above is a /command. `tab` opens the workspaces table, `f1` shows help |

`--keys` defaults to `$ANVIL_KEYS`, then to the active run's keys directory.

## Keeping the agent honest

The point is that **you** do the thinking. While a run is active
(`~/.anvil/run.json` exists), every agent tool call goes through `anvil guard`,
a PreToolUse hook that blocks by default. It's wired in for each agent:

| Agent | Hook file |
|-------|-----------|
| Claude Code, Devin CLI | `.claude/settings.json` |
| Codex | `.codex/hooks.json` |
| Cursor | `.cursor/hooks.json` |
| Gemini CLI | `.gemini/settings.json` |

During a run the agent may:

- read and search files outside the keys repository;
- start the tutor subagent;
- write answer files under `~/.anvil/answers/`;
- run single `anvil grade/build/check/review/path/projects/topics` commands and
  `git -C <dir> status/diff/log/show`.

Everything else is blocked: reading the keys, editing your code, running `go`,
shell operators, and `anvil unlock`, `reveal` and `stop`. Only you can unlock.
The workspace is also made read-only while it's locked.

The full agent rules are in [`AGENTS.md`](AGENTS.md).

Live tests with Claude Code, Devin CLI and Cursor confirmed this: reading the
keys, `go test` and `anvil unlock` were blocked, and reading the workspace,
writing answers and the `anvil` listing commands were allowed. Codex hasn't
been tested live. An agent without hook support, or a session started outside
this repo, is bound only by `AGENTS.md`.

## Repository layout

```
cmd/anvil/            the CLI: run flow (start, build, unlock, check, review), grading, guard
internal/curriculum/  levels, projects, stages, concepts, and the learner-visible bug content
internal/grader/      the model judge, its cache, and the keys cross-checks
internal/guard/       the tool-call policy behind anvil guard
internal/run/         run state in ~/.anvil/run.json
.devin/agents/        the tutor subagent profile
docs/projects.md      the learning path, the stage flow, and known limits
AGENTS.md             rules every agent follows during a learner run
```

The keys repository has this layout:

```
keys/<project>.json                      injection, root cause, rubric, reference answer per stage
projects/<project>/base/                 project skeleton (go.mod)
projects/<project>/stages/NN/correct/    full files after stage NN is fixed
projects/<project>/stages/NN/injected/   the bugged versions of the files that change
projects/run.sh                          checks every stage
```

## Development

```sh
go vet ./...
go test ./...
anvil check-keys --keys /path/to/anvil-keys     # must print OK
bash /path/to/anvil-keys/projects/run.sh        # every stage: CORRECT PASS, INJECTED FAIL
```

`projects/run.sh` builds every stage in a temp dir. Each stage's correct code
must pass all tests, and its bugged code must fail the stage's reproduce test.
It also fails a stage if the bugged files' comments differ from the correct
ones (so comments can't give the bug away), or if the stage doesn't really
change any earlier file. `jobq/06` needs `ANVIL_TEST_PG_DSN` and `k8s/08` needs
`KUBEBUILDER_ASSETS`. Without them, those stages are reported as BLOCKED.

## Known limits

- The agent can read the bugged code in your workspace, just as you do, so it
  might spot the bug. The rules forbid it from hinting.
- If a stage rewrites a file you fixed earlier, `anvil build` merges your
  version in. If the edits clash, it uses the reference version and lists the
  files; your version stays in git history.
- The judge is a model. It runs at temperature 0 and caches every grade, so the
  same answer always gets the same grade, but it can still be wrong. That's
  what `anvil reveal` is for.
