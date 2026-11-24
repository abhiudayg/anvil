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

## How to get to it (user POV)

- `anvil tui` → chat pane (default).
- `tab` or `/ws` → workspaces table; `esc`/`q`/`enter`/`tab` or `/chat` → back.
- `f1` anywhere → help overlay; `esc`/`f1` → close.
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

## Gotchas

- `bubbles/table` needs `SetHeight`/`SetWidth` on `WindowSizeMsg` or it renders 0 rows.
- The ticker must re-arm inside the `tickMsg` handler, not in `Init` only.
- `SnapshotFunc` may be nil in tests — refresh must tolerate it.
- `?` cannot open help on the chat pane (it would type into the input); use `f1`.
