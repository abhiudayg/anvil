"use client";

import { useRef, useState } from "react";

export function SpotlightCard({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const [hovered, setHovered] = useState(false);

  return (
    <div
      ref={cardRef}
      onMouseMove={(e) => {
        const rect = cardRef.current?.getBoundingClientRect();
        if (rect) setPos({ x: e.clientX - rect.left, y: e.clientY - rect.top });
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      className={`relative overflow-hidden rounded-xl border border-white/[0.08] bg-[#0a0a0c] p-6 transition-colors hover:border-white/[0.22] ${className}`}
    >
      <div
        className="pointer-events-none absolute -inset-px transition-opacity duration-300"
        style={{
          opacity: hovered ? 1 : 0,
          background: `radial-gradient(560px circle at ${pos.x}px ${pos.y}px, rgba(255, 83, 26, 0.10), transparent 45%)`,
        }}
      />
      <div className="relative z-10">{children}</div>
    </div>
  );
}
