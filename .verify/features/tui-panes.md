# Feature: tui-panes

k9s-shaped upgrade for `anvil tui`: a pane model (chat + workspaces table),
ticker-driven run-state refresh, per-pane key bindings that self-render as a
hints bar, and a help overlay. Boundary schema: `Snapshot{Header, []Workspace}`
produced once per refresh by `SnapshotFunc`.

## Sub-features

- `pane-switching`: `tab`/`/ws` moves chat → workspaces; `esc`/`q`/`enter`/`tab`
  or `/chat` returns. Pane state is a `pane` enum, not booleans.
- `workspaces-table`: workspaces pane shows a table (NAME, PROJECT, STAGE,
  STATUS, PATH) built from `Snapshot.Workspaces`, sorted by dir.
- `status-refresh`: `tea.Tick` every 2s produces one `snapshotMsg`; `View()`
  reads the cached snapshot only — no per-frame `run.json` reads.
- `help-overlay`: `f1` toggles a centered help overlay on any pane; `esc`/`f1`
  closes it; other keys are swallowed while open.
- `key-hints`: footer renders the active pane's bindings as `key: desc`.
- `stream-replies`: agent replies render as they arrive —
  `Chat.SendStream` reads provider NDJSON (`stream_event` deltas for
  claude, `step_update.text_delta` for agy) through a channel pumped into
  the update loop; gemini degrades to one chunk.
- `command-palette`: `ctrl+p` or `/` on an empty input opens a centered
  fuzzy palette fed by the same `commands` table as `/help`; arrows move,
  runes filter, `enter` picks — arg-taking commands paste into the input,
  arg-free commands run immediately.

## How to get to it (user POV)

- `anvil tui` → chat pane (default).
- `tab` or `/ws` → workspaces table; `esc`/`q`/`enter`/`tab` or `/chat` → back.
- `f1` anywhere → help overlay; `esc`/`f1` → close.
- `ctrl+p` or `/` on empty input → command palette; `esc`/`ctrl+p` → close.
- `ctrl+c`/`ctrl+d` or `/quit` → exit.

## Verification Suites (Targeted Test Commands)

- Unit: `go test ./internal/tui`
- Functional: `go test -run Functional -v ./internal/tui`
- Allowlisted: `sh .verify/run.sh tui-panes`

1. Request: `tab` key on the chat pane.
   Why: pane switching is the feature's spine.
   Expected: model pane == paneWorkspaces; `View()` contains table columns.
   Impact: input stays in chat transcript; table shows snapshot rows.
2. Request: `esc` on the workspaces pane.
   Why: every non-chat pane needs a way back (k9s idiom).
   Expected: pane == paneChat; `View()` shows input line again.
   Impact: none on run state.
3. Request: `snapshotMsg{Workspaces: [{Dir, "redis", 1, "locked"}]}`.
   Why: table is fed by the snapshot boundary, not by reading run.json directly.
   Expected: table has 1 row; header equals Snapshot.Header.
   Impact: model cache updated; next `View()` renders it without IO.
4. Request: `tickMsg` delivered to `Update`.
   Why: refresh must re-arm and refetch on a single ordered path.
   Expected: returned cmds produce a `snapshotMsg` and a future `tickMsg`.
   Impact: `View()` never calls `SnapshotFunc` itself (frame-free IO).
5. Request: `f1` then `esc`.
   Why: modal help must open, swallow keys, close cleanly.
   Expected: `View()` while open shows command help and does not type into input.
   Impact: pane unchanged after close.
6. Request: `View()` on workspaces pane.
   Why: hints must reflect the active pane's bindings.
   Expected: footer contains `esc: back` and `tab: chat`; chat footer contains
   `tab: workspaces`.
   Impact: none.
7. Request: `Chat.SendStream` against a stub emitting two delta lines plus a
   result line.
   Why: replies must surface chunk by chunk, not after the process exits.
   Expected: events carry deltas "Hel","lo" in order, channel closes, session
   captured, no Err.
   Impact: transcript gains the agent block progressively.
8. Request: `ctrl+p`, type `stat`, `enter`.
   Why: the palette must filter the command table and run arg-free picks.
   Expected: palette opens, items reduce to `/status` only, transcript gains a
   `/status` entry without touching the dispatcher.
   Impact: none on run state.
9. Request: `ctrl+p`, type `unlock`, `enter`.
   Why: arg-taking commands must hand control to the input, not run blind.
   Expected: palette closes, input holds `/unlock ` awaiting args.
   Impact: none until the user completes the command.
10. Request: `command()` / palette pick while `m.busy`.
   Why: a second concurrent `Chat.stream` would race on `session`/`sent`,
   and `/clear` mid-stream corrupts the transcript.
   Expected: refused with "still working…", runner not invoked; `/quit` exempt.
   Impact: one async operation at a time.
11. Request: failed `SendStream` (stub exits 1).
   Why: `sent` must stay false or the retry drops the lead prompt.
   Expected: `c.sent == false` after the channel closes.
   Impact: next send retries the full prompt, no resume flags.
12. Request: palette open → type → `esc` → reopen; and `down` past the window.
   Why: stale filter leaks across opens; cursor must stay inside the
   rendered 12-row window.
   Expected: reopened palette is unfiltered, cursor ≤ `paletteRows`-1.
   Impact: none.

## Gotchas

- `bubbles/table` needs `SetHeight`/`SetWidth` on `WindowSizeMsg` or it renders 0 rows.
- The ticker must re-arm inside the `tickMsg` handler, not in `Init` only.
- `SnapshotFunc` may be nil in tests — refresh must tolerate it.
- `?` cannot open help on the chat pane (it would type into the input); use `f1`.
- The `/` binding on an empty input opens the palette — a literal leading `/`
  is unreachable by typing, but palette picks paste it back into the input.
- `streamOn` tracks whether the `── agent ──` header was written, so
  stream errors before the first delta don't leave a dangling empty block.
- `Chat.stream` wraps ctx in `Timeout` (default 10m) and all sends to the
  channel are ctx-aware — a dead reader must not strand the goroutine.
- `sent` is set only after a successful turn; palette/mouse events are
  gated by `m.help || m.palette` and `m.busy` respectively.
- Claude's `stream-json` needs `--verbose` and `--include-partial-messages`
  to emit deltas; agy's needs only `--output-format stream-json`.
