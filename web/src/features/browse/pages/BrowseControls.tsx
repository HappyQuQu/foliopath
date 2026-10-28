import { useLayoutEffect, useRef, useState, type ReactNode } from "react";

import styles from "./BrowsePage.module.css";

export function BrowseControls({
  children,
  heading,
  onHeightChange,
}: {
  children: ReactNode;
  heading: ReactNode;
  onHeightChange: (height: number) => void;
}) {
  const slotRef = useRef<HTMLDivElement>(null);
  const surfaceRef = useRef<HTMLDivElement>(null);
  const headingRef = useRef<HTMLElement>(null);
  const [compact, setCompact] = useState(false);
  const [expandedHeight, setExpandedHeight] = useState<number>();

  useLayoutEffect(() => {
    const slot = slotRef.current!;
    const surface = surfaceRef.current!;
    const headingElement = headingRef.current!;
    let threshold = 0;

    const measure = () => {
      if (!slot.hasAttribute("data-compact")) {
        setExpandedHeight(surface.offsetHeight);
        // Keep the original document footprint while the dock becomes shorter.
        threshold = slot.parentElement!.getBoundingClientRect().top + window.scrollY +
          headingElement.offsetHeight - Number.parseFloat(getComputedStyle(slot).top);
      }
      onHeightChange(surface.offsetHeight);
    };
    const update = () => setCompact(window.scrollY > threshold);
    const resize = () => {
      measure();
      update();
    };
    const observer = new ResizeObserver(measure);
    observer.observe(surface);
    measure();
    update();
    window.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", resize);
    return () => {
      observer.disconnect();
      window.removeEventListener("scroll", update);
      window.removeEventListener("resize", resize);
    };
  }, [onHeightChange]);

  return (
    <div
      className={styles.browseControlsSlot}
      data-compact={compact ? "" : undefined}
      ref={slotRef}
      style={{ height: expandedHeight }}
    >
      <div className={styles.browseControls} ref={surfaceRef}>
        <header className={styles.browseHeading} ref={headingRef}>
          {heading}
        </header>
        {children}
      </div>
    </div>
  );
}
