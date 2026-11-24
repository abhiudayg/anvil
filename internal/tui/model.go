package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
  /clear  /help  /quit
anything else goes to the agent`

var (
	headerStyle = lipgloss.NewStyle().Bold(true).
			BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	whoStyle    = lipgloss.NewStyle().Bold(true)
)

// Model is the whole screen: a transcript on top and one input line.
// Plain text goes to the agent; "/x args" runs anvil x args in-process.
type Model struct {
	chat   *Chat
	run    RunFunc
	status StatusFunc

	vp    viewport.Model
	input textinput.Model
	log   *strings.Builder // shared across value copies; a copied Builder would panic
	busy  bool
}

// New builds the model around the chat session and the command runner.
func New(chat *Chat, run RunFunc, status StatusFunc) Model {
	in := textinput.New()
	in.Placeholder = "talk to the agent, or /command"
	in.Focus()
	m := Model{chat: chat, run: run, status: status, input: in, log: &strings.Builder{}}
	m.say("anvil", "Type /help for commands; anything else goes to the agent.")
	return m
}

type doneMsg struct {
	text string
	err  error
}

func (m *Model) say(who, text string) {
	fmt.Fprintf(m.log, "%s\n%s\n\n", whoStyle.Render("── "+who+" ──"), strings.TrimSpace(text))
}

func (m *Model) flush() {
	m.vp.SetContent(m.log.String())
	m.vp.GotoBottom()
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vp.Width = msg.Width
		m.vp.Height = msg.Height - 5 // header + input + footer
		m.input.Width = msg.Width - 2
		m.flush()
	case doneMsg:
		m.busy = false
		if msg.err != nil {
			m.say("error", msg.err.Error())
		} else {
			m.say("agent", msg.text)
		}
		m.flush()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "enter":
			line := strings.TrimSpace(m.input.Value())
			if line == "" || m.busy {
				return m, nil
			}
			m.input.SetValue("")
			m.say("you", line)
			if strings.HasPrefix(line, "/") {
				return m.command(line[1:])
			}
			m.busy = true
			m.flush()
			return m, func() tea.Msg {
				out, err := m.chat.Send(context.Background(), line)
				return doneMsg{out, err}
			}
		}
	}
	var c1, c2 tea.Cmd
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
		return m, tea.Quit
	case "help":
		m.say("anvil", helpText)
	case "clear":
		m.log.Reset()
	case "status":
		if m.status != nil {
			m.say("anvil", m.status())
		}
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

func (m Model) View() string {
	status := "no run"
	if m.status != nil {
		status = m.status()
	}
	footer := "enter sends · /help for commands · ctrl+c quits"
	if m.busy {
		footer = "working…"
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render(status),
		m.vp.View(),
		m.input.View(),
		footerStyle.Render(footer))
}
