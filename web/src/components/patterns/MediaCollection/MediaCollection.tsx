import {
  Eye,
  FileImage,
  FilmSlate,
  HourglassMedium,
  Play,
  WarningCircle,
} from "@phosphor-icons/react";
import { useWindowVirtualizer } from "@tanstack/react-virtual";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type CSSProperties,
} from "react";

import { Button, InlineStatus } from "../../ui";
import styles from "./MediaCollection.module.css";

export type MediaCollectionLayout = "grid" | "masonry";

export interface MediaCollectionItem {
  height: number | null;
  id: string;
  kind: "image" | "animated" | "video";
  modifiedLabel: string;
  name: string;
  sourceHref?: string;
  sourceLabel?: string;
  storyboard?: MediaCollectionStoryboard;
  thumbnailStatus: "pending" | "ready" | "failed" | "unavailable";
  thumbnailUrl: string | null;
  width: number | null;
}

export interface MediaCollectionStoryboard {
  cellHeight: number;
  cellWidth: number;
  columns: number;
  frameCount: 4 | 10;
  rows: number;
  url: string;
}

export interface MediaCollectionLabels {
  activatePreview: string;
  animated: string;
  failedThumbnail: string;
  image: string;
  loadMore: string;
  loadMoreFailed: string;
  loadingMore: string;
  pendingThumbnail: string;
  previewing: string;
  retryLoadMore: string;
  unavailableThumbnail: string;
  video: string;
}

export interface MediaCollectionHandle {
  restoreItem: (id: string) => void;
}

export const mediaCollectionCapacityBudget = {
  focusRestoreFrames: 12,
  primaryTierItems: 100_000,
} as const;

export const storyboardPlaybackTiming = {
  frameMs: 500,
  hoverIntentMs: 300,
} as const;

interface StoryboardPlayback {
  frame: number;
  itemId: string;
}

export function shouldLoadNextMediaPage({
  columns,
  hasNextPage,
  isFetchingNextPage,
  itemCount,
  lastVirtualIndex,
  paginationError,
}: {
  columns: number;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  itemCount: number;
  lastVirtualIndex: number;
  paginationError: boolean;
}): boolean {
  return (
    hasNextPage &&
    !isFetchingNextPage &&
    !paginationError &&
    lastVirtualIndex >= itemCount - columns * 2
  );
}

export const MediaCollection = forwardRef<MediaCollectionHandle, {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  items: MediaCollectionItem[];
  labels: MediaCollectionLabels;
  layout: MediaCollectionLayout;
  onItemActivate?: (
    id: string,
    activation: "single" | "double",
    trigger: HTMLButtonElement,
  ) => void;
  onLoadMore: () => void;
  onRetryLoadMore?: () => void;
  paginationError?: boolean;
  previewItemId?: string;
  selectedItemId?: string;
}>(function MediaCollection(
  {
    hasNextPage,
    isFetchingNextPage,
    items,
    labels,
    layout,
    onItemActivate,
    onLoadMore,
    onRetryLoadMore,
    paginationError = false,
    previewItemId,
    selectedItemId,
  },
  ref,
) {
  const hostRef = useRef<HTMLDivElement>(null);
  const focusRestoreFrameRef = useRef<number | undefined>(undefined);
  const storyboard = useStoryboardPlayback();
  const [geometry, setGeometry] = useState({ scrollMargin: 0, width: 960 });
  const columns = columnCount(geometry.width);
  const gap = 12;
  const columnWidth =
    (geometry.width - gap * Math.max(0, columns - 1)) / columns;
  const getItemKey = useCallback(
    (index: number) => items[index]?.id ?? `media-${index}`,
    [items],
  );
  const estimateSize = useCallback(
    (index: number) =>
      estimatedCardHeight(items[index], layout, columnWidth),
    [columnWidth, items, layout],
  );
  const virtualizer = useWindowVirtualizer({
    count: items.length,
    estimateSize,
    gap,
    getItemKey,
    initialRect: {
      height: typeof window === "undefined" ? 768 : window.innerHeight,
      width: typeof window === "undefined" ? 1024 : window.innerWidth,
    },
    laneAssignmentMode: "estimate",
    lanes: columns,
    overscan: columns * 2,
    scrollMargin: geometry.scrollMargin,
  });
  const virtualItems = [...virtualizer.getVirtualItems()].sort(
    (a, b) => a.index - b.index,
  );
  const lastVirtualIndex = virtualItems.at(-1)?.index ?? -1;

  useImperativeHandle(
    ref,
    () => ({
      restoreItem(id) {
        const index = items.findIndex((item) => item.id === id);
        if (index < 0) return;

        if (focusRestoreFrameRef.current !== undefined) {
          cancelAnimationFrame(focusRestoreFrameRef.current);
        }
        virtualizer.scrollToIndex(index, { align: "center" });
        const focusTrigger = (remainingFrames: number) => {
          const trigger = Array.from(
            hostRef.current?.querySelectorAll<HTMLButtonElement>(
              "[data-media-id]",
            ) ?? [],
          ).find((candidate) => candidate.dataset.mediaId === id);
          if (trigger) {
            trigger.focus({ preventScroll: true });
            focusRestoreFrameRef.current = undefined;
            return;
          }
          if (remainingFrames <= 0) {
            focusRestoreFrameRef.current = undefined;
            return;
          }
          focusRestoreFrameRef.current = requestAnimationFrame(() =>
            focusTrigger(remainingFrames - 1),
          );
        };
        focusRestoreFrameRef.current = requestAnimationFrame(() =>
          focusTrigger(mediaCollectionCapacityBudget.focusRestoreFrames),
        );
      },
    }),
    [items, virtualizer],
  );

  useEffect(
    () => () => {
      if (focusRestoreFrameRef.current !== undefined) {
        cancelAnimationFrame(focusRestoreFrameRef.current);
      }
    },
    [],
  );

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    const update = () => {
      const bounds = host.getBoundingClientRect();
      setGeometry({
        scrollMargin: bounds.top + window.scrollY,
        width: Math.max(bounds.width, 280),
      });
    };
    update();
    const observer =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(update);
    observer?.observe(host);
    window.addEventListener("resize", update);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", update);
    };
  }, []);

  useEffect(() => {
    virtualizer.measure();
  }, [columns, layout, virtualizer]);

  useEffect(() => {
    if (
      shouldLoadNextMediaPage({
        columns,
        hasNextPage,
        isFetchingNextPage,
        itemCount: items.length,
        lastVirtualIndex,
        paginationError,
      })
    ) {
      onLoadMore();
    }
  }, [
    columns,
    hasNextPage,
    isFetchingNextPage,
    items.length,
    lastVirtualIndex,
    onLoadMore,
    paginationError,
  ]);

  return (
    <div ref={hostRef} className={styles.host}>
      <ul
        className={styles.collection}
        data-layout={layout}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {virtualItems.map((virtualItem) => {
          const item = items[virtualItem.index];
          if (!item) return null;
          return (
            <li
              aria-posinset={virtualItem.index + 1}
              aria-setsize={items.length}
              className={styles.virtualItem}
              data-index={virtualItem.index}
              key={item.id}
              ref={virtualizer.measureElement}
              style={
                {
                  "--collection-left": `calc(${
                    (virtualItem.lane * 100) / columns
                  }% + ${(virtualItem.lane * gap) / columns}px)`,
                  "--collection-width": `calc(${100 / columns}% - ${
                    (gap * (columns - 1)) / columns
                  }px)`,
                  transform: `translateY(${
                    virtualItem.start - geometry.scrollMargin
                  }px)`,
                } as CSSProperties
              }
            >
              <MediaCard
                item={item}
                labels={labels}
                layout={layout}
                onStoryboardEnter={storyboard.start}
                onStoryboardLeave={storyboard.stop}
                {...(onItemActivate ? { onActivate: onItemActivate } : {})}
                previewing={item.id === previewItemId}
                selected={item.id === selectedItemId}
                {...(storyboard.playback?.itemId === item.id
                  ? { storyboardFrame: storyboard.playback.frame }
                  : {})}
              />
            </li>
          );
        })}
      </ul>
      {paginationError && (
        <div className={styles.paginationError}>
          <InlineStatus tone="danger">{labels.loadMoreFailed}</InlineStatus>
          {onRetryLoadMore && (
            <Button
              loading={isFetchingNextPage}
              onClick={onRetryLoadMore}
              size="small"
              variant="secondary"
            >
              {isFetchingNextPage ? labels.loadingMore : labels.retryLoadMore}
            </Button>
          )}
        </div>
      )}
      {hasNextPage && !paginationError && (
        <div className={styles.pagination} role="status" aria-live="polite">
          <Button
            loading={isFetchingNextPage}
            onClick={onLoadMore}
            variant="secondary"
          >
            {isFetchingNextPage ? labels.loadingMore : labels.loadMore}
          </Button>
        </div>
      )}
    </div>
  );
});

export function MediaCollectionSkeleton({
  label,
}: {
  label: string;
}) {
  return (
    <div className={styles.skeletonCollection} role="status" aria-label={label}>
      <span className={styles.visuallyHidden}>{label}</span>
      {Array.from({ length: 12 }, (_, index) => (
        <div aria-hidden="true" className={styles.skeletonCard} key={index}>
          <span className={styles.skeletonThumbnail} />
          <span className={styles.skeletonLine} />
          <span className={styles.skeletonLineShort} />
        </div>
      ))}
    </div>
  );
}

function MediaCard({
  item,
  labels,
  layout,
  onActivate,
  onStoryboardEnter,
  onStoryboardLeave,
  previewing,
  selected,
  storyboardFrame,
}: {
  item: MediaCollectionItem;
  labels: MediaCollectionLabels;
  layout: MediaCollectionLayout;
  onActivate?: (
    id: string,
    activation: "single" | "double",
    trigger: HTMLButtonElement,
  ) => void;
  onStoryboardEnter: (item: MediaCollectionItem) => void;
  onStoryboardLeave: (itemId?: string) => void;
  previewing: boolean;
  selected: boolean;
  storyboardFrame?: number;
}) {
  const kindLabel =
    item.kind === "video"
      ? labels.video
      : item.kind === "animated"
        ? labels.animated
        : labels.image;
  const thumbnailLabel =
    item.thumbnailStatus === "failed"
      ? labels.failedThumbnail
      : item.thumbnailStatus === "unavailable"
        ? labels.unavailableThumbnail
        : labels.pendingThumbnail;
  const aspectRatio =
    layout === "masonry" ? safeAspectRatio(item) : 4 / 3;
  useEffect(
    () => () => {
      onStoryboardLeave(item.id);
    },
    [item.id, onStoryboardLeave],
  );

  return (
    <article
      aria-label={`${item.name} · ${kindLabel}`}
      className={styles.card}
      data-previewing={previewing || undefined}
      data-selected={selected || undefined}
      data-storyboard-playing={
        storyboardFrame !== undefined ? true : undefined
      }
      onPointerEnter={() => onStoryboardEnter(item)}
      onPointerLeave={() => onStoryboardLeave(item.id)}
    >
      {onActivate && (
        <button
          aria-label={labels.activatePreview.replace("{name}", item.name)}
          aria-pressed={selected}
          className={styles.previewTrigger}
          data-media-id={item.id}
          onClick={(event) =>
            onActivate(item.id, "single", event.currentTarget)
          }
          onDoubleClick={(event) =>
            onActivate(item.id, "double", event.currentTarget)
          }
          type="button"
        />
      )}
      <div className={styles.thumbnail} style={{ aspectRatio }}>
        {item.thumbnailStatus === "ready" && item.thumbnailUrl ? (
          <img
            alt=""
            decoding="async"
            loading="lazy"
            src={item.thumbnailUrl}
          />
        ) : (
          <div
            className={styles.thumbnailPlaceholder}
            data-thumbnail-status={item.thumbnailStatus}
          >
            {item.thumbnailStatus === "pending" ? (
              <HourglassMedium aria-hidden="true" size={28} />
            ) : item.thumbnailStatus === "failed" ? (
              <WarningCircle aria-hidden="true" size={28} weight="fill" />
            ) : item.kind === "video" ? (
              <FilmSlate aria-hidden="true" size={28} />
            ) : (
              <FileImage aria-hidden="true" size={28} />
            )}
            <span>{thumbnailLabel}</span>
          </div>
        )}
        {item.storyboard && storyboardFrame !== undefined && (
          <img
            alt=""
            aria-hidden="true"
            className={styles.storyboardSprite}
            data-cover-axis={
              item.storyboard.cellWidth / item.storyboard.cellHeight >=
              aspectRatio
                ? "height"
                : "width"
            }
            decoding="async"
            draggable={false}
            src={item.storyboard.url}
            style={
              {
                "--storyboard-columns": item.storyboard.columns,
                "--storyboard-rows": item.storyboard.rows,
                "--storyboard-x": `${
                  ((storyboardFrame % item.storyboard.columns) + 0.5) /
                  item.storyboard.columns *
                  100
                }%`,
                "--storyboard-y": `${
                  (Math.floor(storyboardFrame / item.storyboard.columns) +
                    0.5) /
                  item.storyboard.rows *
                  100
                }%`,
              } as CSSProperties
            }
          />
        )}
        {item.kind === "video" && (
          <span className={styles.videoBadge} aria-hidden="true">
            <Play size={12} weight="fill" />
          </span>
        )}
        {previewing && (
          <span className={styles.previewBadge} title={labels.previewing}>
            <Eye aria-hidden="true" size={14} weight="fill" />
            <span className={styles.visuallyHidden}>{labels.previewing}</span>
          </span>
        )}
      </div>
      <div className={styles.identity}>
        <strong title={item.name}>{item.name}</strong>
        {item.sourceHref && item.sourceLabel ? (
          <a href={item.sourceHref}>{item.sourceLabel}</a>
        ) : (
          <span>{item.modifiedLabel}</span>
        )}
      </div>
    </article>
  );
}

function useStoryboardPlayback(): {
  playback: StoryboardPlayback | undefined;
  start: (item: MediaCollectionItem) => void;
  stop: (itemId?: string) => void;
} {
  const [playback, setPlayback] = useState<StoryboardPlayback>();
  const pendingItemRef = useRef<string | undefined>(undefined);
  const intentTimerRef = useRef<number | undefined>(undefined);
  const frameTimerRef = useRef<number | undefined>(undefined);
  const decodeImageRef = useRef<HTMLImageElement | undefined>(undefined);
  const generationRef = useRef(0);
  const playbackRef = useRef<StoryboardPlayback | undefined>(undefined);

  const stop = useCallback((itemId?: string) => {
    if (
      itemId &&
      pendingItemRef.current !== itemId &&
      playbackRef.current?.itemId !== itemId
    ) {
      return;
    }
    generationRef.current += 1;
    pendingItemRef.current = undefined;
    if (intentTimerRef.current !== undefined) {
      window.clearTimeout(intentTimerRef.current);
      intentTimerRef.current = undefined;
    }
    if (frameTimerRef.current !== undefined) {
      window.clearInterval(frameTimerRef.current);
      frameTimerRef.current = undefined;
    }
    if (decodeImageRef.current) {
      decodeImageRef.current.src = "";
      decodeImageRef.current = undefined;
    }
    playbackRef.current = undefined;
    setPlayback(undefined);
  }, []);

  const start = useCallback(
    (item: MediaCollectionItem) => {
      stop();
      if (!item.storyboard || !storyboardMotionAllowed()) return;
      const story = item.storyboard;
      const generation = generationRef.current;
      pendingItemRef.current = item.id;
      intentTimerRef.current = window.setTimeout(() => {
        intentTimerRef.current = undefined;
        const image = new Image();
        decodeImageRef.current = image;
        image.decoding = "async";
        image.src = story.url;
        void decodeImage(image)
          .then(() => {
            if (
              generation !== generationRef.current ||
              pendingItemRef.current !== item.id ||
              document.visibilityState === "hidden"
            ) {
              return;
            }
            pendingItemRef.current = undefined;
            decodeImageRef.current = undefined;
            const initial = { frame: 0, itemId: item.id };
            playbackRef.current = initial;
            setPlayback(initial);
            frameTimerRef.current = window.setInterval(() => {
              const current = playbackRef.current;
              if (!current || current.itemId !== item.id) return;
              const next = {
                frame: (current.frame + 1) % story.frameCount,
                itemId: item.id,
              };
              playbackRef.current = next;
              setPlayback(next);
            }, storyboardPlaybackTiming.frameMs);
          })
          .catch(() => {
            if (generation === generationRef.current) stop(item.id);
          });
      }, storyboardPlaybackTiming.hoverIntentMs);
    },
    [stop],
  );

  useEffect(() => {
    const handleVisibility = () => {
      if (document.visibilityState === "hidden") stop();
    };
    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
      stop();
    };
  }, [stop]);

  return { playback, start, stop };
}

function storyboardMotionAllowed(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false;
  }
  return (
    window.matchMedia("(hover: hover) and (pointer: fine)").matches &&
    !window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}

function decodeImage(image: HTMLImageElement): Promise<void> {
  if (typeof image.decode === "function") {
    return image.decode();
  }
  return new Promise((resolve, reject) => {
    image.addEventListener("load", () => resolve(), { once: true });
    image.addEventListener("error", () => reject(new Error("decode failed")), {
      once: true,
    });
  });
}

function columnCount(width: number): number {
  return Math.max(1, Math.min(6, Math.floor((width + 12) / 210)));
}

function estimatedCardHeight(
  item: MediaCollectionItem | undefined,
  layout: MediaCollectionLayout,
  columnWidth: number,
): number {
  const ratio = layout === "masonry" && item ? safeAspectRatio(item) : 4 / 3;
  return columnWidth / ratio + 54;
}

function safeAspectRatio(item: MediaCollectionItem): number {
  if (
    item.width === null ||
    item.height === null ||
    item.width <= 0 ||
    item.height <= 0
  ) {
    return 4 / 3;
  }
  return Math.min(2.4, Math.max(0.6, item.width / item.height));
}
