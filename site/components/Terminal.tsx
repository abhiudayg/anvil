"use client";

import { useEffect, useState } from "react";

type Line = { kind: "cmd" | "out" | "agent" | "you" | "dim"; text: string };

const tabs: { label: string; lines: Line[] }[] = [
  {
    label: "01 teach",
    lines: [
      { kind: "cmd", text: "anvil start" },
      { kind: "out", text: "run started · guard active · tools restricted" },
      { kind: "dim", text: "" },
      { kind: "agent", text: "agent: Stage 4 — RESP protocol & bufio" },
      { kind: "agent", text: "┌ conn ──▶ scanner.ReadBytes('\\n')" },
      { kind: "agent", text: "└ partial reads: when does it block?" },
      { kind: "dim", text: "doubt? → tutor explains with a diagram" },
    ],
  },
  {
    label: "02 build",
    lines: [
      { kind: "cmd", text: "anvil build redis 4" },
      { kind: "out", text: "stage code committed to ~/workspaces/redis" },
      { kind: "out", text: "workspace locked · guard enforcing" },
      { kind: "dim", text: "" },
      { kind: "agent", text: "agent: why does the server hang" },
      { kind: "agent", text: "        after the second SET?" },
    ],
  },
  {
    label: "03 grade",
    lines: [
      { kind: "you", text: "you: the scanner buffer isn't reset" },
      { kind: "you", text: "     between reads, so the second" },
      { kind: "you", text: "     command never parses" },
      { kind: "dim", text: "" },
      { kind: "cmd", text: "anvil grade redis 4 answers/a1.txt" },
      { kind: "out", text: "PERFECT · all rubric points covered" },
      { kind: "dim", text: "" },
      { kind: "cmd", text: "anvil unlock ~/workspaces/redis" },
      { kind: "out", text: "unlocked — write the fix" },
    ],
  },
  {
    label: "04 check",
    lines: [
      { kind: "dim", text: "…you write the fix, not the agent…" },
      { kind: "dim", text: "" },
      { kind: "cmd", text: "anvil check ~/workspaces/redis" },
      { kind: "out", text: "go build ✓ go test ✓" },
      { kind: "out", text: "fix committed · stage 4 closed" },
      { kind: "dim", text: "" },
      { kind: "cmd", text: "anvil review ~/workspaces/redis" },
      { kind: "out", text: "your diff → then the reference fix" },
    ],
  },
];

const colors: Record<Line["kind"], string> = {
  cmd: "text-white",
  out: "text-emerald-400/90",
  agent: "text-neutral-300",
  you: "text-amber-200/90",
  dim: "text-neutral-600",
};

export function Terminal() {
  const [tab, setTab] = useState(0);
  const [shown, setShown] = useState(0);

  useEffect(() => {
    const lines = tabs[tab].lines;
    if (shown < lines.length) {
      const t = setTimeout(() => setShown((n) => n + 1), lines[shown].kind === "cmd" ? 750 : 380);
      return () => clearTimeout(t);
    }
    const next = setTimeout(() => {
      setTab((n) => (n + 1) % tabs.length);
      setShown(0);
    }, 3600);
    return () => clearTimeout(next);
  }, [tab, shown]);

  const pick = (i: number) => {
    setTab(i);
    setShown(0);
  };

  return (
    <div className="w-full">
      <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-1.5 font-mono text-xs text-neutral-300">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-neutral-400">
            <path d="M12 19h8" />
            <path d="m4 17 6-6-6-6" />
          </svg>
          anvil-learner-shell
        </div>
        <div className="relative flex items-center gap-1 rounded-lg border border-white/[0.06] bg-white/[0.04] p-1">
          {tabs.map((t, i) => (
            <button
              key={t.label}
              onClick={() => pick(i)}
              className="relative rounded-md px-3 py-1 font-mono text-xs transition-colors focus:outline-none"
            >
              {i === tab && <div className="absolute inset-0 rounded-md bg-white shadow-sm" />}
              <span className={`relative z-10 font-medium ${i === tab ? "font-semibold text-black" : "text-neutral-400 hover:text-white"}`}>
                {t.label}
              </span>
            </button>
          ))}
        </div>
      </div>
      <div className="overflow-hidden rounded-xl border border-white/[0.1] bg-[#0a0a0c] shadow-[0_30px_60px_-15px_rgba(0,0,0,0.9)]">
        <div className="min-h-[320px] p-5 font-mono text-[13px] leading-6">
          {tabs[tab].lines.slice(0, shown).map((l, i) =>
            l.text === "" ? (
              <div key={i} className="h-3" />
            ) : (
              <div key={i} className={colors[l.kind]}>
                {l.kind === "cmd" && <span className="mr-2 text-[#ff7a4a]">$</span>}
                {l.kind === "out" && <span className="mr-2 text-neutral-600">→</span>}
                {l.text}
              </div>
            ),
          )}
          <div>
            <span className="mr-2 text-[#ff7a4a]">$</span>
            <span className="caret inline-block h-4 w-2 translate-y-0.5 bg-[#ff7a4a]" />
          </div>
        </div>
      </div>
    </div>
  );
}
