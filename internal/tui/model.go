package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var refreshInterval = 2 * time.Second

const helpText = `commands:
  /path                     full learning order
  /projects                 list projects
  /topics [project]         concepts by stage
  /status                   run state
  /start [--keys DIR]       begin a run
  /stop                     end the run
  /build <ws> <proj> <n>    lay stage code and lock the workspace
  /unlock <ws> <file>       grade an answer file; PERFECT unlocks
  /reveal <ws>              reference answer after 5 graded tries
  /check <ws>               build, test and commit the fix
  /review <ws>              learner diff, then the reference fix
  /grade <proj> <n> <file>  grade an answer file
  /check-keys [--keys DIR]  cross-check the keys repository
  /ws  /chat  /clear  /help  /quit
anything else goes to the agent`

var (
	headerStyle = lipgloss.NewStyle().Bold(true).
			BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	whoStyle    = lipgloss.NewStyle().Bold(true)
	helpStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2)
)

type pane int

const (
	paneChat pane = iota
	paneWorkspaces
)

// Workspace is one row of the workspaces table: a learner workspace dir and
// the stage state the run recorded for it.
type Workspace struct {
	Dir     string
	Project string
	Stage   int
	Status  string
}

// Snapshot is the boundary schema between the run state and the screen: one
// load feeds both the header line and the workspaces table.
type Snapshot struct {
	Header     string
	Workspaces []Workspace
}

// SnapshotFunc loads the current run state; called once per refresh tick.
type SnapshotFunc func() Snapshot

type binding struct {
	key  string
	desc string
	act  func(Model) (tea.Model, tea.Cmd)
}

// Model is the whole screen: a chat pane (transcript + input) and a
// workspaces pane (run-state table), plus a help overlay. Plain text goes to
// the agent; "/x args" runs anvil x args in-process.
type Model struct {
	chat   *Chat
	run    RunFunc
	snapFn SnapshotFunc

	pane   pane
	help   bool
	header string
	width  int
	height int

	vp    viewport.Model
	input textinput.Model
	table table.Model
	log   *strings.Builder // shared across value copies; a copied Builder would panic
	busy  bool

	ctx    context.Context
	cancel context.CancelFunc
}

// New builds the model around the chat session, the command runner and the
// run-state snapshot provider.
func New(chat *Chat, run RunFunc, snap SnapshotFunc) Model {
	in := textinput.New()
	in.Placeholder = "talk to the agent, or /command"
	in.Focus()
	tbl := table.New(
		table.WithColumns([]table.Column{
			{Title: "NAME", Width: 16},
			{Title: "PROJECT", Width: 12},
			{Title: "STAGE", Width: 6},
			{Title: "STATUS", Width: 10},
			{Title: "PATH", Width: 40},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	ctx, cancel := context.WithCancel(context.Background())
	m := Model{
		chat: chat, run: run, snapFn: snap,
		pane: paneChat, header: "anvil",
		input: in, table: tbl, log: &strings.Builder{},
		ctx: ctx, cancel: cancel,
	}
	m.say("anvil", "Type /help for commands; anything else goes to the agent.")
	return m
}

type doneMsg struct {
	text string
	err  error
}

type snapshotMsg Snapshot

type tickMsg struct{}

func (m *Model) say(who, text string) {
	fmt.Fprintf(m.log, "%s\n%s\n\n", whoStyle.Render("── "+who+" ──"), strings.TrimSpace(text))
}

func (m *Model) flush() {
	m.vp.SetContent(m.log.String())
	m.vp.GotoBottom()
}

func (m Model) loadSnapshot() tea.Cmd {
	return func() tea.Msg {
		if m.snapFn == nil {
			return snapshotMsg{}
		}
		return snapshotMsg(m.snapFn())
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadSnapshot(), tick())
}

func send(m Model) (tea.Model, tea.Cmd) {
	line := strings.TrimSpace(m.input.Value())
	if line == "" || m.busy {
		return m, nil
	}
	m.input.SetValue("")
	m.say("you", line)
	if strings.HasPrefix(line, "/") {
		return m.command(line[1:])
	}
	if m.chat == nil {
		m.say("error", "no agent configured")
		m.flush()
		return m, nil
	}
	m.busy = true
	m.flush()
	return m, func() tea.Msg {
		out, err := m.chat.Send(m.ctx, line)
		return doneMsg{out, err}
	}
}

func quit(m Model) (tea.Model, tea.Cmd) {
	m.cancel()
	return m, tea.Quit
}

func (m Model) bindings() []binding {
	if m.pane == paneWorkspaces {
		return []binding{
			{"esc", "back", func(m Model) (tea.Model, tea.Cmd) { m.pane = paneChat; return m, nil }},
			{"q", "back", func(m Model) (tea.Model, tea.Cmd) { m.pane = paneChat; return m, nil }},
			{"enter", "back", func(m Model) (tea.Model, tea.Cmd) { m.pane = paneChat; return m, nil }},
			{"tab", "chat", func(m Model) (tea.Model, tea.Cmd) { m.pane = paneChat; return m, nil }},
			{"f1", "help", func(m Model) (tea.Model, tea.Cmd) { m.help = true; return m, nil }},
			{"ctrl+c", "quit", quit},
			{"ctrl+d", "quit", quit},
		}
	}
	return []binding{
		{"enter", "send", send},
		{"tab", "workspaces", func(m Model) (tea.Model, tea.Cmd) { m.pane = paneWorkspaces; return m, nil }},
		{"f1", "help", func(m Model) (tea.Model, tea.Cmd) { m.help = true; return m, nil }},
		{"ctrl+c", "quit", quit},
		{"ctrl+d", "quit", quit},
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.vp.Width = msg.Width
		m.vp.Height = max(msg.Height-4, 1) // header + input + footer
		m.input.Width = msg.Width - 2
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(max(msg.Height-4, 1))
		m.flush()
	case snapshotMsg:
		s := Snapshot(msg)
		if s.Header != "" {
			m.header = s.Header
		}
		var rows []table.Row
		for _, w := range s.Workspaces {
			rows = append(rows, table.Row{
				filepath.Base(w.Dir), w.Project,
				strconv.Itoa(w.Stage), w.Status, w.Dir,
			})
		}
		m.table.SetRows(rows)
	case tickMsg:
		return m, tea.Batch(m.loadSnapshot(), tick())
	case doneMsg:
		m.busy = false
		if msg.err != nil {
			m.say("error", msg.err.Error())
		} else {
			m.say("agent", msg.text)
		}
		m.flush()
	case tea.KeyMsg:
		if m.help {
			switch msg.String() {
			case "ctrl+c", "ctrl+d":
				return quit(m)
			case "esc", "f1", "enter", "q":
				m.help = false
			}
			return m, nil
		}
		for _, b := range m.bindings() {
			if b.key == msg.String() {
				return b.act(m)
			}
		}
	}
	if m.help {
		return m, nil
	}
	var c1, c2 tea.Cmd
	if m.pane == paneWorkspaces {
		m.table, c1 = m.table.Update(msg)
		return m, c1
	}
	m.vp, c1 = m.vp.Update(msg)
	m.input, c2 = m.input.Update(msg)
	return m, tea.Batch(c1, c2)
}

// command handles "/x args": meta verbs inline, everything else goes to
// the anvil dispatcher asynchronously so the screen stays live.
func (m Model) command(line string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		m.flush()
		return m, nil
	}
	switch fields[0] {
	case "quit", "q", "exit":
		return quit(m)
	case "help":
		m.say("anvil", helpText)
	case "clear":
		m.log.Reset()
	case "status":
		m.say("anvil", m.header)
	case "ws", "workspaces":
		m.pane = paneWorkspaces
	case "chat":
		m.pane = paneChat
	case "tui", "guard":
		m.say("anvil", "/"+fields[0]+" is not available here")
	default:
		m.busy = true
		m.flush()
		return m, func() tea.Msg {
			out, code := m.run(fields)
			if out == "" {
				out = fmt.Sprintf("(exit %d)", code)
			} else if code != 0 {
				out += fmt.Sprintf("\n(exit %d)", code)
			}
			return doneMsg{out, nil}
		}
	}
	m.flush()
	return m, nil
}

func (m Model) hints() string {
	if m.busy {
		return "working…"
	}
	var parts []string
	for _, b := range m.bindings() {
		parts = append(parts, b.key+": "+b.desc)
	}
	return strings.Join(parts, " · ")
}

func (m Model) View() string {
	body := lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), m.input.View())
	if m.pane == paneWorkspaces {
		body = m.table.View()
	}
	screen := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render(m.header),
		body,
		footerStyle.Render(m.hints()))
	if m.help {
		screen = lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, helpStyle.Render(helpText))
	}
	return screen
}
