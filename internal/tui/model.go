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

type cmdItem struct {
	name  string
	usage string
	desc  string
}

var commands = []cmdItem{
	{"/path", "", "full learning order"},
	{"/projects", "", "list projects"},
	{"/topics", "[project]", "concepts by stage"},
	{"/status", "", "run state"},
	{"/start", "[--keys DIR]", "begin a run"},
	{"/stop", "", "end the run"},
	{"/build", "<ws> <proj> <n>", "lay stage code and lock the workspace"},
	{"/unlock", "<ws> <file>", "grade an answer file; PERFECT unlocks"},
	{"/reveal", "<ws>", "reference answer after 5 graded tries"},
	{"/check", "<ws>", "build, test and commit the fix"},
	{"/review", "<ws>", "learner diff, then the reference fix"},
	{"/grade", "<proj> <n> <file>", "grade an answer file"},
	{"/check-keys", "[--keys DIR]", "cross-check the keys repository"},
	{"/ws", "", "workspaces pane"},
	{"/chat", "", "chat pane"},
	{"/clear", "", "clear the transcript"},
	{"/help", "", "this help"},
	{"/quit", "", "exit the TUI"},
}

var helpText = func() string {
	var b strings.Builder
	b.WriteString("commands:\n")
	for _, c := range commands {
		fmt.Fprintf(&b, "  %-26s %s\n", strings.TrimSpace(c.name+" "+c.usage), c.desc)
	}
	b.WriteString("anything else goes to the agent")
	return b.String()
}()

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

	pane      pane
	help      bool
	header    string
	width     int
	height    int
	palette   bool
	palFilter string
	palCursor int
	streamOn  bool

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

type streamEventMsg struct {
	ch <-chan StreamEvent
	ev StreamEvent
}

type streamDoneMsg struct{}

func awaitStream(ch <-chan StreamEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return streamEventMsg{ch, ev}
	}
}

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
	return m, awaitStream(m.chat.SendStream(m.ctx, line))
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
			{"ctrl+p", "commands", func(m Model) (tea.Model, tea.Cmd) { m.palette = true; m.pane = paneChat; return m, nil }},
			{"f1", "help", func(m Model) (tea.Model, tea.Cmd) { m.help = true; return m, nil }},
			{"ctrl+c", "quit", quit},
			{"ctrl+d", "quit", quit},
		}
	}
	return []binding{
		{"enter", "send", send},
		{"tab", "workspaces", func(m Model) (tea.Model, tea.Cmd) { m.pane = paneWorkspaces; return m, nil }},
		{"ctrl+p", "commands", func(m Model) (tea.Model, tea.Cmd) { m.palette = true; return m, nil }},
		{"/", "commands", func(m Model) (tea.Model, tea.Cmd) {
			if m.input.Value() != "" {
				m.input.SetValue(m.input.Value() + "/")
				return m, nil
			}
			m.palette = true
			return m, nil
		}},
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
			m.say("anvil", msg.text)
		}
		m.flush()
	case streamEventMsg:
		if msg.ev.Err != nil {
			if m.streamOn {
				fmt.Fprint(m.log, "\n\n")
				m.streamOn = false
			}
			m.busy = false
			m.say("error", msg.ev.Err.Error())
			m.flush()
			return m, nil
		}
		if !m.streamOn {
			fmt.Fprintf(m.log, "%s\n", whoStyle.Render("── agent ──"))
			m.streamOn = true
		}
		fmt.Fprint(m.log, msg.ev.Delta)
		m.flush()
		return m, awaitStream(msg.ch)
	case streamDoneMsg:
		if m.streamOn {
			fmt.Fprint(m.log, "\n\n")
			m.streamOn = false
		}
		m.busy = false
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
		if m.palette {
			return m.paletteKey(msg)
		}
		for _, b := range m.bindings() {
			if b.key == msg.String() {
				return b.act(m)
			}
		}
	}
	if m.help || m.palette {
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
	}
	if m.busy {
		m.say("anvil", "still working…")
		m.flush()
		return m, nil
	}
	switch fields[0] {
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

func (m Model) paletteItems() []cmdItem {
	f := strings.ToLower(m.palFilter)
	var out []cmdItem
	for _, c := range commands {
		if f == "" || strings.Contains(strings.ToLower(c.name+" "+c.usage+" "+c.desc), f) {
			out = append(out, c)
		}
	}
	return out
}

func (m Model) paletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.paletteItems()
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return quit(m)
	case "esc", "ctrl+p":
		m.palette = false
		m.palFilter = ""
		m.palCursor = 0
	case "up":
		if m.palCursor > 0 {
			m.palCursor--
		}
	case "down":
		if m.palCursor < min(len(items), paletteRows)-1 {
			m.palCursor++
		}
	case "backspace", "ctrl+h":
		if n := len(m.palFilter); n > 0 {
			m.palFilter = m.palFilter[:n-1]
		}
	case "enter":
		return m.pick(items)
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.palFilter += string(msg.Runes)
		}
	}
	if m.palCursor > min(len(m.paletteItems()), paletteRows)-1 {
		m.palCursor = max(min(len(m.paletteItems()), paletteRows)-1, 0)
	}
	return m, nil
}

func (m Model) pick(items []cmdItem) (tea.Model, tea.Cmd) {
	m.palette = false
	m.palFilter = ""
	if len(items) == 0 {
		m.palCursor = 0
		return m, nil
	}
	it := items[m.palCursor]
	m.palCursor = 0
	m.pane = paneChat
	if strings.ContainsAny(it.usage, "<[") {
		m.input.SetValue(it.name + " ")
		return m, nil
	}
	m.say("you", it.name)
	m.flush()
	return m.command(it.name[1:])
}

const paletteRows = 12

func (m Model) paletteView() string {
	var b strings.Builder
	b.WriteString(" commands\n\n")
	fmt.Fprintf(&b, "  > %s\n\n", m.palFilter)
	items := m.paletteItems()
	shown := items
	if len(shown) > paletteRows {
		shown = shown[:paletteRows]
	}
	for i, it := range shown {
		line := fmt.Sprintf("  %-26s %s", strings.TrimSpace(it.name+" "+it.usage), it.desc)
		if i == m.palCursor {
			line = whoStyle.Render("▸" + line[1:])
		}
		b.WriteString(line + "\n")
	}
	if len(items) == 0 {
		b.WriteString("  no matches\n")
	}
	return helpStyle.Render(b.String())
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
	if m.palette {
		screen = lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, m.paletteView())
	} else if m.help {
		screen = lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, helpStyle.Render(helpText))
	}
	return screen
}
