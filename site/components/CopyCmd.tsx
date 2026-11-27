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
      className={`group inline-flex w-fit items-center gap-2.5 rounded-lg border border-white/[0.1] bg-[#0a0a0c] px-4 py-2.5 font-mono text-sm text-neutral-200 transition-colors hover:border-white/[0.25] ${className}`}
    >
      <span className="text-[#ff7a4a]">$</span>
      <span>{cmd}</span>
      <span className="text-neutral-600 transition-colors group-hover:text-white">
        {copied ? "✓" : "⧉"}
      </span>
    </button>
  );
}
