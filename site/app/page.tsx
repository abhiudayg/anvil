import Link from "next/link";
import { CopyCmd } from "@/components/CopyCmd";
import { SpotlightCard } from "@/components/SpotlightCard";
import { Terminal } from "@/components/Terminal";

const pipeline = [
  { n: "01", name: "Teach", desc: "Concepts from the curriculum, short text + diagrams" },
  { n: "02", name: "Build", desc: "Real stage code committed, bug included, workspace locked" },
  { n: "03", name: "Answer & Grade", desc: "Explain the bug; anvil grade judges intent" },
  { n: "04", name: "Unlock", desc: "PERFECT → anvil unlock. 5 misses → anvil reveal" },
  { n: "05", name: "Fix & Check", desc: "Your fix. anvil check builds, tests, commits" },
  { n: "06", name: "Review", desc: "Your diff, then the reference fix" },
];

const projects = [
  { name: "redis", desc: "RESP-speaking, concurrent, persistent key-value server", stages: "10 stages", concepts: "185 concepts" },
  { name: "jobq", desc: "HTTP job API on Postgres — workers, retries, gRPC, a CLI", stages: "13 stages", concepts: "120 concepts" },
  { name: "k8s", desc: "A Kubernetes operator for the job queue, controller-runtime", stages: "8 stages", concepts: "91 concepts" },
];

const pillars = [
  {
    n: "01",
    title: "Guarded sessions",
    desc: "A PreToolUse hook restricts the agent while a run is active. Reads and lesson commands pass — everything else is denied.",
    code: ["allow: Read, Grep, anvil grade", "deny:  go, cat, shell ops, keys repo"],
  },
  {
    n: "02",
    title: "Hidden answer keys",
    desc: "Grading runs inside the CLI against a private keys repo. The agent never reads it — it cannot leak the solution.",
    code: ["$ anvil grade redis 4 a1.txt", "→ points 1,3 covered · 2 missing"],
  },
  {
    n: "03",
    title: "Real workspaces",
    desc: "One git repository per project on your machine. Stage code comes only from anvil build; every fix is written by you.",
    code: ["~/workspaces/redis  [locked]", "git log: stage 04 · bug seeded"],
  },
  {
    n: "04",
    title: "Tutor subagent",
    desc: "Doubts route to a tutor profile that explains with diagrams — resumed within a stage so it never re-reads your code.",
    code: ["tutor: resumed per stage", "passes file paths, not contents"],
  },
  {
    n: "05",
    title: "A TUI, not a doc",
    desc: "k9s-style panes: streaming agent chat, a live workspaces table, a fuzzy command palette, a help overlay.",
    code: ["$ anvil tui", "tab: workspaces · ctrl+p: palette"],
  },
  {
    n: "06",
    title: "Deterministic grades",
    desc: "A pinned-temperature judge grades intent against rubric points — every grade is cached, so repeated answers score identically.",
    code: ["temperature: 0 · cached", "PERFECT → unlock"],
  },
];

const numbers = [
  { v: "3", k: "projects", d: "redis → jobq → k8s operator" },
  { v: "31", k: "stages", d: "each ships one deliberate bug" },
  { v: "12", k: "levels", d: "foundations → production" },
  { v: "396", k: "concepts", d: "taught before every break" },
];

const faqs = [
  {
    q: "Does the AI agent write my fix?",
    a: "No — and it can't. The guard hook restricts the agent's tools while a run is active: it may teach and grade, but it cannot edit your workspace. Every fix is written by you, in your own git repository.",
  },
  {
    q: "How does grading work if the agent can't see the answers?",
    a: "The anvil CLI grades, not the agent. It reads the hidden keys repository — rubric points, root cause, reference answer — and reports only point numbers. PERFECT unlocks the stage; five graded misses let you reveal.",
  },
  {
    q: "Which agent CLIs does it work with?",
    a: "The guard hooks Claude Code, Devin CLI, Codex, Cursor and Gemini CLI. In the TUI you can chat with claude, agy or gemini — replies stream live and sessions resume across turns.",
  },
  {
    q: "What do I need to run it?",
    a: "brew install abhiudayg/tap/anvil, Go 1.27+ for the workspaces, and an API key for the judge (ANTHROPIC_API_KEY, GEMINI_API_KEY, or a logged-in Claude CLI).",
  },
  {
    q: "Where do the answer keys live?",
    a: "In a separate private repository, anvil-keys. Only the anvil CLI reads it — agents are hard-blocked from it, which is what keeps grading honest.",
  },
];

export default function Home() {
  return (
    <main className="relative min-h-screen overflow-x-clip bg-black text-white">
      <header className="fixed inset-x-0 top-0 z-50 border-b border-white/[0.06] bg-black/70 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3.5 sm:px-6 lg:px-8">
          <Link href="/" className="flex items-center gap-2.5">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-gradient-to-b from-[#ff8c5a] to-[#ff4500] font-mono text-sm font-bold text-white shadow-[0_0_14px_rgba(255,83,26,0.5)]">
              A
            </span>
            <span className="text-[15px] font-semibold tracking-tight">Anvil</span>
          </Link>
          <nav className="hidden items-center gap-7 lg:flex">
            <a href="#loop" className="text-sm font-medium text-neutral-400 transition-colors hover:text-white">Loop</a>
            <a href="#projects" className="text-sm font-medium text-neutral-400 transition-colors hover:text-white">Projects</a>
            <a href="#pillars" className="text-sm font-medium text-neutral-400 transition-colors hover:text-white">Why</a>
            <a href="#faq" className="text-sm font-medium text-neutral-400 transition-colors hover:text-white">FAQ</a>
          </nav>
          <div className="flex items-center gap-3">
            <a href="https://github.com/abhiudayg/anvil" className="hidden items-center gap-1.5 rounded-lg border border-white/[0.1] px-3 py-1.5 text-sm text-neutral-300 transition-colors hover:border-white/[0.25] hover:text-white sm:flex">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor"><path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.55v-2.17c-3.2.7-3.87-1.36-3.87-1.36-.52-1.33-1.28-1.68-1.28-1.68-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.23-1.28-5.23-5.68 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.46.11-3.04 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.58.23 2.75.11 3.04.74.81 1.19 1.83 1.19 3.09 0 4.41-2.69 5.38-5.25 5.66.41.36.78 1.05.78 2.12v3.15c0 .3.21.67.8.55A11.51 11.51 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5Z"/></svg>
              GitHub
            </a>
            <a href="#quickstart" className="rounded-lg bg-white px-3.5 py-1.5 text-sm font-semibold text-black transition-colors hover:bg-neutral-200">
              Quickstart
            </a>
          </div>
        </div>
      </header>

      <section className="relative overflow-hidden bg-[#000000] pb-20 pt-36">
        <div className="bg-grid-pattern pointer-events-none absolute inset-0 opacity-60" />
        <div className="pointer-events-none absolute -top-32 left-1/3 h-[480px] w-[700px] rounded-full bg-[radial-gradient(circle,rgba(255,83,26,0.22)_0%,rgba(255,83,26,0.07)_45%,transparent_70%)] blur-[100px]" />
        <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="rise mb-8 flex flex-wrap items-center gap-2 font-mono text-xs text-neutral-400">
            <a href="https://github.com/abhiudayg/anvil/releases" className="text-emerald-400 hover:underline">v0.1.0 latest release</a>
            <span className="text-neutral-600">•</span>
            <span className="font-medium text-white">Guarded Learning Sessions</span>
            <span className="text-neutral-600">•</span>
            <span>CLI + TUI</span>
            <span className="text-neutral-600">•</span>
            <span className="text-neutral-500">MIT Open Source</span>
          </div>
          <div className="grid grid-cols-1 items-start gap-10 lg:grid-cols-12">
            <div className="rise flex flex-col space-y-6 lg:col-span-6">
              <h1 className="text-5xl font-black leading-[1.02] tracking-tight text-white sm:text-7xl">
                Anvil.
                <span className="mt-2 block font-bold text-neutral-500">Learn Go by fixing real systems.</span>
              </h1>
              <p className="max-w-xl text-lg leading-8 text-neutral-400">
                Three real projects, built stage by stage — each stage ships code
                with one bug on purpose. You diagnose it, explain it in your own
                words, then write the fix. The agent teaches the concepts, but it
                never sees the answers and can never touch your code.
              </p>
              <div className="flex flex-col gap-4">
                <CopyCmd cmd="brew install abhiudayg/tap/anvil" />
                <div className="flex flex-wrap gap-x-5 gap-y-2 font-mono text-xs text-neutral-500">
                  <span><span className="text-emerald-400">✓</span> macOS &amp; Linux</span>
                  <span><span className="text-emerald-400">✓</span> Guarded agent sessions</span>
                  <span><span className="text-emerald-400">✓</span> Deterministic grading</span>
                </div>
              </div>
            </div>
            <div className="rise lg:col-span-6" style={{ animationDelay: "0.15s" }}>
              <Terminal />
            </div>
          </div>
        </div>
      </section>

      <section id="loop" className="border-y border-white/[0.08] bg-[#000000] py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="mb-3 font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">DETERMINISTIC LOOP</p>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">One stage, six moves.</h2>
          <div className="mt-10 grid gap-px overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.06] sm:grid-cols-2 lg:grid-cols-6">
            {pipeline.map((s) => (
              <div key={s.n} className="bg-[#0a0a0c] p-5 transition-colors hover:bg-[#111114]">
                <div className="font-mono text-xs text-[#ff7a4a]">{s.n}</div>
                <div className="mt-2 text-sm font-semibold">{s.name}</div>
                <p className="mt-2 text-xs leading-5 text-neutral-500">{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section id="projects" className="border-b border-white/[0.08] bg-[#000000] py-24">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="mb-3 font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">THE PROJECTS</p>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Three real systems, not exercises.</h2>
          <p className="mt-4 max-w-2xl text-neutral-400">
            Each project is a working codebase you grow stage by stage in your own
            git workspace — from language foundations to Kubernetes in production.
          </p>
          <div className="mt-10 grid gap-4 md:grid-cols-3">
            {projects.map((p) => (
              <SpotlightCard key={p.name}>
                <div className="flex items-center justify-between">
                  <span className="font-mono text-sm text-[#ff8c5a]">{p.name}</span>
                  <span className="font-mono text-[10px] text-neutral-500">{p.stages} · {p.concepts}</span>
                </div>
                <p className="mt-3 text-sm leading-6 text-neutral-400">{p.desc}</p>
              </SpotlightCard>
            ))}
          </div>
        </div>
      </section>

      <section id="drift" className="relative overflow-hidden border-b border-white/[0.08] bg-[#000000] py-24">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="mb-3 font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">WHY IT WORKS</p>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Stop watching. Start debugging.</h2>
          <div className="mt-10 grid gap-4 md:grid-cols-2">
            <div className="rounded-xl border border-white/[0.06] bg-[#070709] p-8 opacity-75">
              <p className="font-mono text-xs tracking-[0.25em] text-neutral-500">THE TUTORIAL TRAP</p>
              <h3 className="mt-3 text-xl font-semibold text-neutral-300">Watch. Nod. Forget.</h3>
              <ul className="mt-6 space-y-3 text-sm leading-6 text-neutral-500">
                <li>× 40-minute video, zero lines you wrote</li>
                <li>× Copy-pasted code you can&rsquo;t debug</li>
                <li>× An assistant that hands you the answer</li>
                <li>× Nothing in your git history to show for it</li>
              </ul>
            </div>
            <div className="rounded-xl border border-white/[0.14] bg-[#0a0a0c] p-8 shadow-[0_0_50px_-12px_rgba(255,83,26,0.25)]">
              <p className="font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">THE ANVIL LOOP</p>
              <h3 className="mt-3 text-xl font-semibold">Build. Break. Explain. Fix.</h3>
              <ul className="mt-6 space-y-3 text-sm leading-6 text-neutral-300">
                <li><span className="text-emerald-400">✓</span> A real git workspace per project</li>
                <li><span className="text-emerald-400">✓</span> One deliberate bug per stage — you diagnose it</li>
                <li><span className="text-emerald-400">✓</span> Graded by intent against hidden keys</li>
                <li><span className="text-emerald-400">✓</span> Every fix committed to your history</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      <section id="pillars" className="relative overflow-hidden border-b border-white/[0.08] bg-[#000000] py-24">
        <div className="pointer-events-none absolute -bottom-40 right-0 h-[400px] w-[600px] rounded-full bg-[radial-gradient(circle,rgba(255,83,26,0.14)_0%,transparent_70%)] blur-[100px]" />
        <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="mb-3 font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">THE 6 PILLARS</p>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">The agent can teach. It can&rsquo;t cheat for you.</h2>
          <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {pillars.map((p) => (
              <SpotlightCard key={p.n}>
                <div className="font-mono text-xs text-[#ff7a4a]">{p.n}</div>
                <div className="mt-2 text-base font-semibold">{p.title}</div>
                <p className="mt-2 text-sm leading-6 text-neutral-400">{p.desc}</p>
                <div className="mt-4 space-y-1 rounded-lg border border-white/[0.08] bg-black/60 p-3 font-mono text-[11px] leading-5 text-neutral-400">
                  {p.code.map((c) => <div key={c}>{c}</div>)}
                </div>
              </SpotlightCard>
            ))}
          </div>
        </div>
      </section>

      <section id="numbers" className="border-b border-white/[0.08] bg-[#000000] py-24">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="mb-3 font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">BY THE NUMBERS</p>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Counted, not estimated.</h2>
          <div className="mt-10 grid gap-px overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.06] sm:grid-cols-2 lg:grid-cols-4">
            {numbers.map((n) => (
              <div key={n.k} className="bg-[#0a0a0c] p-8 transition-colors hover:bg-[#111114]">
                <div className="text-4xl font-black tracking-tight text-white">{n.v}</div>
                <div className="mt-1 font-mono text-xs uppercase tracking-widest text-[#ff8c5a]">{n.k}</div>
                <p className="mt-3 text-sm text-neutral-500">{n.d}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section id="faq" className="border-b border-white/[0.08] bg-[#000000] py-24">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="mb-3 font-mono text-xs tracking-[0.25em] text-[#ff7a4a]">FAQ</p>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Frequently Asked Questions</h2>
          <div className="mt-10 divide-y divide-white/[0.08] overflow-hidden rounded-xl border border-white/[0.08]">
            {faqs.map((f) => (
              <details key={f.q} className="group bg-[#0a0a0c] transition-colors open:bg-[#111114]">
                <summary className="flex cursor-pointer list-none items-center justify-between px-6 py-5 text-sm font-medium text-neutral-200 transition-colors hover:text-white [&::-webkit-details-marker]:hidden">
                  {f.q}
                  <span className="ml-4 font-mono text-neutral-500 transition-transform duration-200 group-open:rotate-45">+</span>
                </summary>
                <p className="px-6 pb-6 text-sm leading-7 text-neutral-400">{f.a}</p>
              </details>
            ))}
          </div>
        </div>
      </section>

      <section id="quickstart" className="relative overflow-hidden bg-[#000000] py-24">
        <div className="pointer-events-none absolute left-1/2 top-0 h-[300px] w-[700px] -translate-x-1/2 rounded-full bg-[radial-gradient(circle,rgba(255,83,26,0.18)_0%,transparent_70%)] blur-[90px]" />
        <div className="relative mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
          <h2 className="text-4xl font-black tracking-tight md:text-5xl">Start with Anvil.</h2>
          <p className="mx-auto mt-4 max-w-md text-neutral-400">
            Three real Go systems. Thirty-one deliberate bugs. Every fix is yours.
          </p>
          <div className="mt-8 flex justify-center">
            <CopyCmd cmd="brew install abhiudayg/tap/anvil" />
          </div>
          <p className="mt-4 font-mono text-xs text-neutral-600">then: anvil start && anvil tui</p>
        </div>
      </section>

      <footer className="border-t border-white/[0.08] bg-[#000000]">
        <div className="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
          <div className="grid gap-10 md:grid-cols-4">
            <div className="md:col-span-1">
              <div className="flex items-center gap-2.5">
                <span className="flex h-7 w-7 items-center justify-center rounded-md bg-gradient-to-b from-[#ff8c5a] to-[#ff4500] font-mono text-sm font-bold text-white">A</span>
                <span className="text-[15px] font-semibold">Anvil</span>
              </div>
              <p className="mt-4 text-sm leading-6 text-neutral-500">
                Guarded learning sessions for Go — the agent teaches, the CLI grades, you fix.
              </p>
            </div>
            <div>
              <p className="font-mono text-xs uppercase tracking-widest text-neutral-500">Projects</p>
              <ul className="mt-4 space-y-2.5 text-sm text-neutral-400">
                <li><a href="#projects" className="hover:text-white">redis — key-value server</a></li>
                <li><a href="#projects" className="hover:text-white">jobq — job API + workers</a></li>
                <li><a href="#projects" className="hover:text-white">k8s — operator</a></li>
              </ul>
            </div>
            <div>
              <p className="font-mono text-xs uppercase tracking-widest text-neutral-500">Product</p>
              <ul className="mt-4 space-y-2.5 text-sm text-neutral-400">
                <li><a href="#loop" className="hover:text-white">The loop</a></li>
                <li><a href="#pillars" className="hover:text-white">Why it works</a></li>
                <li><a href="#quickstart" className="hover:text-white">Quickstart</a></li>
              </ul>
            </div>
            <div>
              <p className="font-mono text-xs uppercase tracking-widest text-neutral-500">Context</p>
              <ul className="mt-4 space-y-2.5 text-sm text-neutral-400">
                <li><a href="https://github.com/abhiudayg/anvil" className="hover:text-white">GitHub repository</a></li>
                <li><a href="https://github.com/abhiudayg/anvil/releases" className="hover:text-white">Releases</a></li>
                <li><a href="https://github.com/abhiudayg/homebrew-tap" className="hover:text-white">Homebrew tap</a></li>
              </ul>
            </div>
          </div>
          <div className="mt-12 flex flex-col gap-2 border-t border-white/[0.06] pt-6 font-mono text-xs text-neutral-600 sm:flex-row sm:items-center sm:justify-between">
            <span>© 2026 Anvil · Open source under the MIT license</span>
            <span>31 stages · 12 levels · 396 concepts</span>
          </div>
        </div>
      </footer>
    </main>
  );
}
