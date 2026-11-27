"use client";

import { useState } from "react";

export function CopyCmd({ cmd, className = "" }: { cmd: string; className?: string }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    await navigator.clipboard.writeText(cmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 1400);
  };

  return (
    <button
      onClick={copy}
      className={`group inline-flex items-center gap-2.5 rounded-lg border border-white/[0.1] bg-[#0c0e14] px-4 py-2.5 font-mono text-sm text-white/80 transition-colors hover:border-cyan-400/40 hover:text-white ${className}`}
    >
      <span className="text-cyan-400/70">$</span>
      <span>{cmd}</span>
      <span className="text-white/30 transition-colors group-hover:text-cyan-300">
        {copied ? "✓" : "⧉"}
      </span>
    </button>
  );
}
