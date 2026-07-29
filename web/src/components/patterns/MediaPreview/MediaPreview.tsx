import {
  ArrowsOut,
  CaretLeft,
  CaretRight,
  PushPin,
  X,
} from "@phosphor-icons/react";
import {
  useEffect,
  useState,
  type KeyboardEvent,
  type PointerEvent,
} from "react";

import { Button, IconButton } from "../../ui";
import {
  MediaAvailabilityState,
  type MediaAvailabilityPresentation,
} from "../MediaAvailabilityState/MediaAvailabilityState";
import styles from "./MediaPreview.module.css";

export interface MediaPreviewItem {
  contentUrl: string;
  details: Array<{ label: string; value: string }>;
  id: string;
  kind: "image" | "animated" | "video";
  name: string;
  posterUrl?: string | undefined;
}

export interface MediaPreviewLabels {
  close: string;
  followingDescription: string;
  followingTitle: string;
  imageFailed: string;
  loadFailedDescription: string;
  next: string;
  openViewer: string;
  pin: string;
  pinnedDescription: string;
  pinnedTitle: string;
  position: string;
  previous: string;
  preview: string;
  resize: string;
  retry: string;
  unpin: string;
  videoFailed: string;
}

const widthStep = 24;

export function MediaPreview({
  canGoNext,
  canGoPrevious,
  item,
  labels,
  availability,
  maxWidth = 620,
  minWidth = 360,
  onClose,
  onNext,
  onOpenViewer,
  onPinnedChange,
  onPrevious,
  onWidthChange,
  pinned,
  width,
}: {
  canGoNext: boolean;
  canGoPrevious: boolean;
  item: MediaPreviewItem;
  labels: MediaPreviewLabels;
  availability?: MediaAvailabilityPresentation | undefined;
  maxWidth?: number;
  minWidth?: number;
  onClose: () => void;
  onNext: () => void;
  onOpenViewer: () => void;
  onPinnedChange: (pinned: boolean) => void;
  onPrevious: () => void;
  onWidthChange: (width: number) => void;
  pinned: boolean;
  width: number;
}) {
  const [loadFailed, setLoadFailed] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [resizing, setResizing] = useState(false);

  useEffect(() => {
    setLoadAttempt(0);
    setLoadFailed(false);
  }, [item.id]);

  useEffect(() => {
    const closeOnEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      onClose();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onClose]);

  function resize(nextWidth: number) {
    onWidthChange(Math.min(maxWidth, Math.max(minWidth, nextWidth)));
  }

  function startResize(event: PointerEvent<HTMLDivElement>) {
    event.preventDefault();
    const startX = event.clientX;
    const startWidth = width;
    setResizing(true);
    event.currentTarget.setPointerCapture(event.pointerId);

    const handleMove = (moveEvent: globalThis.PointerEvent) => {
      resize(startWidth + startX - moveEvent.clientX);
    };
    const handleEnd = () => {
      setResizing(false);
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleEnd);
      window.removeEventListener("pointercancel", handleEnd);
    };
    window.addEventListener("pointermove", handleMove);
    window.addEventListener("pointerup", handleEnd);
    window.addEventListener("pointercancel", handleEnd);
  }

  function resizeWithKeyboard(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      resize(width + widthStep);
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      resize(width - widthStep);
    } else if (event.key === "Home") {
      event.preventDefault();
      resize(minWidth);
    } else if (event.key === "End") {
      event.preventDefault();
      resize(maxWidth);
    }
  }

  const failedLabel =
    item.kind === "video" ? labels.videoFailed : labels.imageFailed;
  const visibleAvailability =
    availability ??
    (loadFailed
      ? {
          actionLabel: labels.retry,
          description: labels.loadFailedDescription,
          kind: "loadFailed" as const,
          onAction: () => {
            setLoadFailed(false);
            setLoadAttempt((current) => current + 1);
          },
          title: failedLabel,
        }
      : undefined);

  return (
    <aside
      aria-label={`${labels.preview}: ${item.name}`}
      className={styles.preview}
      data-resizing={resizing || undefined}
      style={{ width }}
    >
      <div
        aria-label={labels.resize}
        aria-orientation="vertical"
        aria-valuemax={maxWidth}
        aria-valuemin={minWidth}
        aria-valuenow={Math.round(width)}
        className={styles.resizer}
        onKeyDown={resizeWithKeyboard}
        onPointerDown={startResize}
        role="separator"
        tabIndex={0}
      >
        <span />
      </div>
      <header className={styles.header}>
        <strong>{labels.preview}</strong>
        <div className={styles.actions}>
          <Button
            aria-label={pinned ? labels.unpin : labels.pin}
            aria-pressed={pinned}
            className={styles.pinButton}
            data-pinned={pinned || undefined}
            onClick={() => onPinnedChange(!pinned)}
            size="small"
            variant="secondary"
          >
            <PushPin aria-hidden="true" size={15} weight={pinned ? "fill" : "regular"} />
            {labels.pin}
          </Button>
          <IconButton label={labels.close} onClick={onClose}>
            <X aria-hidden="true" size={19} />
          </IconButton>
        </div>
      </header>

      <div className={styles.stage}>
        {visibleAvailability ? (
          <MediaAvailabilityState compact state={visibleAvailability} />
        ) : item.kind === "video" ? (
          <video
            aria-label={item.name}
            controls
            key={`${item.id}:${loadAttempt}`}
            onError={() => setLoadFailed(true)}
            playsInline
            poster={item.posterUrl}
            preload="metadata"
            src={item.contentUrl}
          />
        ) : (
          <img
            alt={item.name}
            key={`${item.id}:${loadAttempt}`}
            onError={() => setLoadFailed(true)}
            src={item.contentUrl}
          />
        )}
      </div>

      <div className={styles.navigation}>
        <nav className={styles.navGroup} aria-label={labels.preview}>
          <Button
            disabled={!canGoPrevious}
            onClick={onPrevious}
            size="small"
            variant="secondary"
          >
            <CaretLeft aria-hidden="true" size={15} />
            {labels.previous}
          </Button>
          <span aria-live="polite">{labels.position}</span>
          <Button
            disabled={!canGoNext}
            onClick={onNext}
            size="small"
            variant="secondary"
          >
            {labels.next}
            <CaretRight aria-hidden="true" size={15} />
          </Button>
        </nav>
        <Button
          className={styles.openViewer}
          onClick={onOpenViewer}
          size="small"
          variant="quiet"
        >
          <ArrowsOut aria-hidden="true" size={16} />
          {labels.openViewer}
        </Button>
      </div>

      <dl className={styles.details}>
        <div className={styles.fileIdentity}>
          <strong title={item.name}>{item.name}</strong>
          <span>{item.kind === "video" ? "VIDEO" : item.kind === "animated" ? "GIF" : "JPG"}</span>
        </div>
        {item.details.map((detail) => (
          <div key={detail.label}>
            <dt>{detail.label}</dt>
            <dd title={detail.value}>{detail.value}</dd>
          </div>
        ))}
      </dl>
      <div className={styles.pinStatus} role="status">
        <PushPin aria-hidden="true" size={18} weight={pinned ? "fill" : "regular"} />
        <div>
          <strong>{pinned ? labels.pinnedTitle : labels.followingTitle}</strong>
          <span>
            {pinned ? labels.pinnedDescription : labels.followingDescription}
          </span>
        </div>
      </div>
    </aside>
  );
}
