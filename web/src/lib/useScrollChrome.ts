import { useEffect, useState, type RefObject } from "react";

const hideDistance = 48;
const revealDistance = 12;

export function useScrollChrome(enabled: boolean, blocked: boolean, shell: RefObject<HTMLElement | null>): boolean {
  const [hidden, setHidden] = useState(false);

  useEffect(() => {
    setHidden(false);
    if (!enabled || blocked) return;
    let previousY = window.scrollY;
    let direction = 0;
    let distance = 0;
    let keyboard = false;
    const reveal = () => { distance = 0; direction = 0; setHidden(false); };
    const onScroll = () => {
      const y = Math.max(0, Math.min(window.scrollY, document.documentElement.scrollHeight - window.innerHeight));
      const delta = y - previousY;
      previousY = y;
      if (y < hideDistance || chromeIsBusy(keyboard)) { reveal(); return; }
      if (delta === 0) return;
      const nextDirection = Math.sign(delta);
      distance = nextDirection === direction ? distance + Math.abs(delta) : Math.abs(delta);
      direction = nextDirection;
      if (direction < 0 && distance >= revealDistance) reveal();
      if (direction > 0 && distance >= hideDistance) {
        const collection = shell.current?.querySelector('[data-layout="seamless"]');
        if (collection && collection.getBoundingClientRect().top < window.innerHeight && collection.getBoundingClientRect().bottom > 0) setHidden(true);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Tab" || event.key === "Escape") { keyboard = true; reveal(); }
    };
    const onPointer = () => { keyboard = false; };
    const onFocus = () => { if (chromeIsBusy(keyboard)) reveal(); };
    const onResize = () => { previousY = window.scrollY; reveal(); };
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("wheel", onPointer, { passive: true });
    window.addEventListener("touchstart", onPointer, { passive: true });
    window.addEventListener("pointerdown", onPointer, { passive: true });
    window.addEventListener("keydown", onKeyDown);
    document.addEventListener("focusin", onFocus);
    window.addEventListener("resize", onResize);
    return () => {
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("wheel", onPointer);
      window.removeEventListener("touchstart", onPointer);
      window.removeEventListener("pointerdown", onPointer);
      window.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("focusin", onFocus);
      window.removeEventListener("resize", onResize);
    };
  }, [blocked, enabled, shell]);

  return enabled && !blocked && hidden;
}

function chromeIsBusy(keyboard: boolean): boolean {
  if (document.querySelector('[role="menu"], [role="dialog"], [role="listbox"]')) return true;
  const focused = document.activeElement;
  if (!(focused instanceof HTMLElement) || !focused.closest("[data-scroll-chrome]")) return false;
  return keyboard || focused.matches('input[type="search"], input[type="text"], input:not([type]), textarea, [contenteditable="true"]');
}
