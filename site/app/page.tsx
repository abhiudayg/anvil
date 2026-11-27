import { CopyCmd } from "@/components/CopyCmd";
import { SpotlightCard } from "@/components/SpotlightCard";
import { Terminal } from "@/components/Terminal";

const stats = ["3 projects", "31 stages", "12 levels", "396 concepts", "MIT license"];

const pipeline = [
  { n: "01", name: "Teach", desc: "The agent explains the stage's concepts with text and diagrams — from the curriculum only." },
  { n: "02", name: "Build", desc: "anvil build commits the stage's real code — bug included — into your workspace and locks it." },
  { n: "03", name: "Answer & Grade", desc: "You explain the bug in your own words. anvil grade judges intent, prints point numbers only." },
  { n: "04", name: "Unlock", desc: "On PERFECT you run anvil unlock yourself. Five misses and you may anvil reveal instead." },
  { n: "05", name: "Fix & Check", desc: "You write the fix — not the agent. anvil check runs go build + go test and commits it." },
  { n: "06", name: "Review", desc: "anvil review shows your diff, then the reference fix. The stage closes." },
];

const pillars = [
  {
    title: "Guarded sessions",
    desc: "A PreToolUse hook restricts the agent while a run is active: reads allowed, lesson commands allowed, everything else denied.",
    mono: "deny: go, cat, shell ops",
    span: "md:col-span-2",
  },
  {
    title: "Hidden answer keys",
    desc: "Grading runs inside the CLI against a private keys repo. The agent never reads it — it cannot leak the solution.",
    mono: "keys: CLI-only",
    span: "",
  },
  {
    title: "Real workspaces",
    desc: "One git repository per project on your machine. Stage code comes from anvil build; every fix is written by you.",
    mono: "git: 1 repo / project",
    span: "",
  },
  {
    title: "Tutor subagent",
    desc: "Doubts route to a tutor profile that explains with diagrams — resumed within a stage so it never re-reads your code.",
    mono: "tutor: resumed per stage",
    span: "",
  },
  {
    title: "A TUI, not a doc",
    desc: "k9s-style panes: streaming agent chat, a live workspaces table, a fuzzy command palette, a help overlay.",
    mono: "anvil tui: chat + table + palette",
    span: "md:col-span-2",
  },
];

const quickstart = [
  { n: "01", cmd: "brew install abhiudayg/tap/anvil", desc: "Install the CLI" },
  { n: "02", cmd: "anvil start", desc: "Begin a run — the guard turns on" },
  { n: "03", cmd: "anvil tui", desc: "Chat, workspaces table, command palette — one screen" },
];

export default function Home() {
  return (
    <main className="relative min-h-screen overflow-x-clip">
      <div className="grid-mesh pointer-events-none absolute inset-0" />
      <div className="pointer-events-none absolute -top-40 left-1/2 h-[560px] w-[900px] -translate-x-1/2 rounded-full bg-gradient-to-tr from-cyan-500/20 via-purple-600/15 to-transparent blur-[120px]" />

      <nav className="relative z-10 mx-auto flex max-w-6xl items-center justify-between px-6 py-6">
        <span className="text-lg font-semibold tracking-tight">Anvil</span>
        <a
          href="https://github.com/abhiudayg/anvil"
          className="rounded-lg border border-white/[0.1] px-4 py-1.5 font-mono text-sm text-white/70 transition-colors hover:border-white/30 hover:text-white"
        >
          GitHub →
        </a>
      </nav>

      <section className="relative z-10 mx-auto grid max-w-6xl items-center gap-12 px-6 pb-24 pt-16 lg:grid-cols-2">
        <div className="rise">
          <div className="mb-5 flex flex-wrap gap-2 font-mono text-xs text-white/50">
            {stats.map((s) => (
              <span key={s} className="rounded-full border border-white/[0.08] bg-white/[0.03] px-3 py-1">
                {s}
              </span>
            ))}
          </div>
          <h1 className="text-5xl font-semibold leading-[1.05] tracking-tight md:text-6xl">
            Learn Go by fixing{" "}
            <span className="bg-gradient-to-r from-cyan-300 to-purple-400 bg-clip-text text-transparent">
              real systems.
            </span>
          </h1>
          <p className="mt-6 max-w-md text-lg leading-8 text-white/60">
            Anvil builds three real projects stage by stage — each with one bug on
            purpose. You find it, explain it, fix it. The agent teaches the
            concepts, but it never sees the answers.
          </p>
          <div className="mt-8 flex flex-col gap-3">
            <CopyCmd cmd="brew install abhiudayg/tap/anvil" />
            <span className="font-mono text-xs text-white/30">
              macOS &amp; Linux · brew or source · free &amp; open source
            </span>
          </div>
        </div>
        <div className="rise" style={{ animationDelay: "0.15s" }}>
          <Terminal />
        </div>
      </section>

      <section className="relative z-10 mx-auto max-w-6xl px-6 py-20">
        <p className="font-mono text-xs tracking-[0.3em] text-cyan-300/60">THE LOOP</p>
        <h2 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">
          One stage, six moves.
        </h2>
        <div className="mt-10 grid gap-4 md:grid-cols-3">
          {pipeline.map((s) => (
            <SpotlightCard key={s.n}>
              <div className="font-mono text-xs text-cyan-300/60">{s.n}</div>
              <div className="mt-2 text-lg font-medium">{s.name}</div>
              <p className="mt-2 text-sm leading-6 text-white/55">{s.desc}</p>
            </SpotlightCard>
          ))}
        </div>
      </section>

      <section className="relative z-10 mx-auto max-w-6xl px-6 py-20">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="rounded-2xl border border-white/[0.06] bg-[#0a0b0f] p-8 opacity-80">
            <p className="font-mono text-xs tracking-[0.3em] text-white/30">THE TUTORIAL TRAP</p>
            <h3 className="mt-3 text-2xl font-semibold tracking-tight text-white/70">
              Watch. Nod. Forget.
            </h3>
            <ul className="mt-6 space-y-3 text-sm leading-6 text-white/45">
              <li>× 40-minute video, zero lines you wrote</li>
              <li>× Copy-pasted code you don&rsquo;t own and can&rsquo;t debug</li>
              <li>× An assistant that just hands you the answer</li>
              <li>× Nothing in your git history to show for it</li>
            </ul>
          </div>
          <SpotlightCard>
            <p className="font-mono text-xs tracking-[0.3em] text-cyan-300/60">THE ANVIL LOOP</p>
            <h3 className="mt-3 text-2xl font-semibold tracking-tight">
              Build. Break. Explain. Fix.
            </h3>
            <ul className="mt-6 space-y-3 text-sm leading-6 text-white/65">
              <li>✓ A real git workspace per project, on your machine</li>
              <li>✓ One deliberate bug per stage — you diagnose it</li>
              <li>✓ Graded by intent against hidden keys, not vibes</li>
              <li>✓ Every fix committed to your own history</li>
            </ul>
          </SpotlightCard>
        </div>
      </section>

      <section className="relative z-10 mx-auto max-w-6xl px-6 py-20">
        <p className="font-mono text-xs tracking-[0.3em] text-cyan-300/60">WHY IT WORKS</p>
        <h2 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">
          The agent can teach. It can&rsquo;t cheat for you.
        </h2>
        <div className="mt-10 grid gap-4 md:grid-cols-3">
          {pillars.map((p) => (
            <SpotlightCard key={p.title} className={p.span}>
              <div className="text-lg font-medium">{p.title}</div>
              <p className="mt-2 text-sm leading-6 text-white/55">{p.desc}</p>
              <div className="mt-4 font-mono text-xs text-cyan-300/50">{p.mono}</div>
            </SpotlightCard>
          ))}
        </div>
      </section>

      <section className="relative z-10 mx-auto max-w-6xl px-6 py-20">
        <p className="font-mono text-xs tracking-[0.3em] text-cyan-300/60">QUICKSTART</p>
        <h2 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">
          Up in sixty seconds.
        </h2>
        <div className="mt-10 grid gap-4 md:grid-cols-3">
          {quickstart.map((q) => (
            <SpotlightCard key={q.n}>
              <div className="font-mono text-xs text-cyan-300/60">{q.n}</div>
              <div className="mt-3 break-all rounded-lg border border-white/[0.08] bg-[#0a0c10] px-3 py-2.5 font-mono text-[13px] text-white/80">
                <span className="mr-2 text-cyan-400/70">$</span>
                {q.cmd}
              </div>
              <p className="mt-3 text-sm text-white/55">{q.desc}</p>
            </SpotlightCard>
          ))}
        </div>
      </section>

      <footer className="relative z-10 border-t border-white/[0.06]">
        <div className="mx-auto flex max-w-6xl flex-col items-start justify-between gap-4 px-6 py-10 text-sm text-white/40 md:flex-row md:items-center">
          <div>
            <span className="font-semibold text-white/70">Anvil</span>
            <span className="ml-3">Learn Go by building real tools and debugging them.</span>
          </div>
          <div className="flex gap-6 font-mono text-xs">
            <a href="https://github.com/abhiudayg/anvil" className="transition-colors hover:text-white">
              github
            </a>
            <span>MIT license</span>
            <span>© 2026</span>
          </div>
        </div>
      </footer>
    </main>
  );
}
