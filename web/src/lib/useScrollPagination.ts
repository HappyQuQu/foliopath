import { useCallback, useEffect, useRef, useState } from "react";

const prefetchDistance = 320;
const wheelGestureIdleMs = 250;

interface ScrollPaginationOptions {
  enabled?: boolean;
  failed: boolean;
  hasNextPage: boolean;
  isFetching: boolean;
  nearEnd?: boolean;
  onLoadMore: () => void;
  onRetry?: (() => void) | undefined;
  progressKey: string | number;
}

export function useScrollPagination({
  enabled = true,
  failed,
  hasNextPage,
  isFetching,
  nearEnd = false,
  onLoadMore,
  onRetry,
  progressKey,
}: ScrollPaginationOptions) {
  const [target, setTarget] = useState<HTMLElement | null>(null);
  const busy = useRef(false);
  const lastWheelAt = useRef(-Infinity);
  const touchY = useRef<number | null>(null);
  const touchRetried = useRef(false);

  const loadMore = useCallback(() => {
    if (busy.current || isFetching || (!hasNextPage && !failed)) return;
    const action = failed ? onRetry : onLoadMore;
    if (!action) return;
    busy.current = true;
    action();
  }, [failed, hasNextPage, isFetching, onLoadMore, onRetry]);

  useEffect(() => {
    busy.current = isFetching;
  }, [enabled, failed, isFetching, progressKey]);

  useEffect(() => {
    if (!target || !enabled) return;
    if (!failed && (nearEnd || isNearPagination(target))) loadMore();
  }, [enabled, failed, isFetching, loadMore, nearEnd, progressKey, target]);

  useEffect(() => {
    if (!target || !enabled) return;
    const root = paginationScrollRoot(target);
    const events = root ?? window;
    const tryLoad = (retry: boolean) => {
      if ((!failed || retry) && (nearEnd || isNearPagination(target))) loadMore();
    };
    const observer = typeof IntersectionObserver === "undefined" ? null : new IntersectionObserver(
      entries => {
        if (entries.some(entry => entry.isIntersecting)) tryLoad(false);
      },
      { root, rootMargin: `0px 0px ${prefetchDistance}px 0px` },
    );
    observer?.observe(target);
    const onScroll = () => tryLoad(false);
    const ownsEvent = (event: Event) =>
      event.target instanceof Element ? paginationScrollRoot(event.target) === root : root === null;
    const onWheel = (event: Event) => {
      if (!ownsEvent(event)) return;
      const wheel = event as WheelEvent;
      if (wheel.deltaY < 0) { lastWheelAt.current = -Infinity; return; }
      const now = performance.now();
      const freshGesture = now - lastWheelAt.current > wheelGestureIdleMs;
      lastWheelAt.current = now;
      if (wheel.deltaY > 0) tryLoad(freshGesture);
    };
    const onTouchStart = (event: Event) => {
      if (!ownsEvent(event)) return;
      touchY.current = (event as TouchEvent).touches[0]?.clientY ?? null;
      touchRetried.current = false;
    };
    const onTouchMove = (event: Event) => {
      if (!ownsEvent(event)) return;
      const y = (event as TouchEvent).touches[0]?.clientY;
      if (y === undefined || touchY.current === null) return;
      if (y < touchY.current && !touchRetried.current) {
        tryLoad(true);
        touchRetried.current = true;
      }
      touchY.current = y;
    };
    const onKeyDown = (event: Event) => {
      if (!ownsEvent(event)) return;
      if (event.target instanceof Element && event.target.closest("input, textarea, select, button, [contenteditable=true], [role=combobox]")) return;
      const key = event as KeyboardEvent;
      if (!key.repeat && !(key.key === " " && key.shiftKey) && ["ArrowDown", "PageDown", "End", " "].includes(key.key)) tryLoad(true);
    };
    events.addEventListener("scroll", onScroll, { passive: true });
    events.addEventListener("wheel", onWheel, { passive: true });
    events.addEventListener("touchstart", onTouchStart, { passive: true });
    events.addEventListener("touchmove", onTouchMove, { passive: true });
    events.addEventListener("keydown", onKeyDown);
    return () => {
      observer?.disconnect();
      events.removeEventListener("scroll", onScroll);
      events.removeEventListener("wheel", onWheel);
      events.removeEventListener("touchstart", onTouchStart);
      events.removeEventListener("touchmove", onTouchMove);
      events.removeEventListener("keydown", onKeyDown);
    };
  }, [enabled, failed, loadMore, nearEnd, target]);

  return { loadMore, ref: setTarget };
}

function paginationScrollRoot(target: Element): HTMLElement | null {
  for (let parent: Element | null = target; parent && parent !== document.body; parent = parent.parentElement) {
    if (parent instanceof HTMLElement && /(auto|scroll)/.test(getComputedStyle(parent).overflowY)) return parent;
  }
  return null;
}

function isNearPagination(target: HTMLElement): boolean {
  const bounds = target.getBoundingClientRect();
  if (bounds.width === 0 || bounds.height === 0) return false;
  const root = paginationScrollRoot(target)?.getBoundingClientRect();
  const top = Math.max(root?.top ?? 0, 0);
  const bottom = Math.min(root?.bottom ?? window.innerHeight, window.innerHeight);
  const left = Math.max(root?.left ?? 0, 0);
  const right = Math.min(root?.right ?? window.innerWidth, window.innerWidth);
  return bottom > top && bounds.top <= bottom + prefetchDistance && bounds.left < right && bounds.right > left;
}
