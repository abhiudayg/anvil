"use client";

import { useEffect } from "react";
import Lenis from "lenis";

export function SmoothScroll({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    if ("ontouchstart" in window || navigator.maxTouchPoints > 0) return;

    const lenis = new Lenis({
      duration: 0.9,
      easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
      orientation: "vertical",
      smoothWheel: true,
      syncTouch: false,
    });

    let frameId: number;
    const raf = (time: number) => {
      lenis.raf(time);
      frameId = requestAnimationFrame(raf);
    };
    frameId = requestAnimationFrame(raf);

    const observer = new ResizeObserver(() => lenis.resize());
    observer.observe(document.body);
    lenis.resize();

    return () => {
      cancelAnimationFrame(frameId);
      observer.disconnect();
      lenis.destroy();
    };
  }, []);

  return <>{children}</>;
}
