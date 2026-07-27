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
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  forwardRef,
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
  thumbnailStatus: "pending" | "ready" | "failed" | "unavailable";
  thumbnailUrl: string | null;
  width: number | null;
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

        virtualizer.scrollToIndex(index, { align: "center" });
        const focusTrigger = () => {
          const trigger = Array.from(
            hostRef.current?.querySelectorAll<HTMLButtonElement>(
              "[data-media-id]",
            ) ?? [],
          ).find((candidate) => candidate.dataset.mediaId === id);
          trigger?.focus({ preventScroll: true });
          return Boolean(trigger);
        };
        requestAnimationFrame(() => {
          if (!focusTrigger()) requestAnimationFrame(focusTrigger);
        });
      },
    }),
    [items, virtualizer],
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
      hasNextPage &&
      !isFetchingNextPage &&
      lastVirtualIndex >= items.length - columns * 2
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
                {...(onItemActivate ? { onActivate: onItemActivate } : {})}
                previewing={item.id === previewItemId}
                selected={item.id === selectedItemId}
              />
            </li>
          );
        })}
      </ul>
      {paginationError && (
        <div className={styles.paginationError}>
          <InlineStatus tone="danger">{labels.loadMoreFailed}</InlineStatus>
          {onRetryLoadMore && (
            <Button onClick={onRetryLoadMore} size="small" variant="secondary">
              {labels.retryLoadMore}
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
  previewing,
  selected,
}: {
  item: MediaCollectionItem;
  labels: MediaCollectionLabels;
  layout: MediaCollectionLayout;
  onActivate?: (
    id: string,
    activation: "single" | "double",
    trigger: HTMLButtonElement,
  ) => void;
  previewing: boolean;
  selected: boolean;
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

  return (
    <article
      aria-label={`${item.name} · ${kindLabel}`}
      className={styles.card}
      data-previewing={previewing || undefined}
      data-selected={selected || undefined}
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

function columnCount(width: number): number {
  return Math.max(1, Math.min(6, Math.floor((width + 12) / 180)));
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
