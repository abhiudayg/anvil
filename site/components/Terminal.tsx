"use client";

import { useEffect, useState } from "react";

type Line = { kind: "cmd" | "out" | "agent" | "you"; text: string };

const script: Line[] = [
  { kind: "cmd", text: "anvil start" },
  { kind: "out", text: "run started · guard active · tools restricted" },
  { kind: "agent", text: "agent: Stage 4 — how bufio.Scanner splits a RESP stream" },
  { kind: "cmd", text: "anvil build redis 4" },
  { kind: "out", text: "stage code committed · ~/workspaces/redis locked" },
  { kind: "agent", text: "agent: why does the server hang after the second SET?" },
  { kind: "you", text: "you: the scanner's buffer isn't reset between reads…" },
  { kind: "cmd", text: "anvil grade redis 4 answers/a1.txt" },
  { kind: "out", text: "PERFECT · all rubric points covered" },
  { kind: "cmd", text: "anvil unlock ~/workspaces/redis" },
  { kind: "out", text: "unlocked — write the fix" },
  { kind: "cmd", text: "anvil check ~/workspaces/redis" },
  { kind: "out", text: "go build ✓ go test ✓ committed · stage 4 closed" },
];

const colors: Record<Line["kind"], string> = {
  cmd: "text-white",
  out: "text-emerald-300/80",
  agent: "text-cyan-300/80",
  you: "text-amber-200/80",
};

export function Terminal() {
  const [shown, setShown] = useState(0);

  useEffect(() => {
    if (shown >= script.length) {
      const reset = setTimeout(() => setShown(0), 5200);
      return () => clearTimeout(reset);
    }
    const line = script[shown];
    const delay = line.kind === "cmd" ? 900 : 500;
    const t = setTimeout(() => setShown((n) => n + 1), delay);
    return () => clearTimeout(t);
  }, [shown]);

  return (
    <div className="overflow-hidden rounded-2xl border border-white/[0.08] bg-[#0a0c10] font-mono text-[13px] leading-6 shadow-[0_40px_80px_-20px_rgba(0,0,0,0.8)]">
      <div className="flex items-center gap-1.5 border-b border-white/[0.06] px-4 py-2.5">
        <span className="h-2.5 w-2.5 rounded-full bg-white/15" />
        <span className="h-2.5 w-2.5 rounded-full bg-white/15" />
        <span className="h-2.5 w-2.5 rounded-full bg-white/15" />
        <span className="ml-3 text-[11px] tracking-widest text-white/30">anvil — learner session</span>
      </div>
      <div className="min-h-[340px] p-5">
        {script.slice(0, shown).map((l, i) => (
          <div key={i} className={colors[l.kind]}>
            {l.kind === "cmd" && <span className="mr-2 text-cyan-400/70">$</span>}
            {l.kind === "out" && <span className="mr-2 text-white/20">→</span>}
            {l.text}
          </div>
        ))}
        <div>
          <span className="mr-2 text-cyan-400/70">$</span>
          <span className="caret inline-block h-4 w-2 translate-y-0.5 bg-cyan-300" />
        </div>
      </div>
    </div>
  );
}
