import {
  ArrowsOut,
  CaretLeft,
  CaretRight,
  FileImage,
  FilmSlate,
  PushPin,
  PushPinSlash,
  X,
} from "@phosphor-icons/react";
import {
  useEffect,
  useState,
  type KeyboardEvent,
  type PointerEvent,
} from "react";

import { Button, IconButton } from "../../ui";
import styles from "./MediaPreview.module.css";

export interface MediaPreviewItem {
  contentUrl: string;
  details: Array<{ label: string; value: string }>;
  id: string;
  kind: "image" | "animated" | "video";
  name: string;
}

export interface MediaPreviewLabels {
  close: string;
  followingDescription: string;
  followingTitle: string;
  imageFailed: string;
  next: string;
  openViewer: string;
  pin: string;
  pinnedDescription: string;
  pinnedTitle: string;
  position: string;
  previous: string;
  preview: string;
  resize: string;
  unpin: string;
  videoFailed: string;
}

const widthStep = 24;

export function MediaPreview({
  canGoNext,
  canGoPrevious,
  item,
  labels,
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
  const [resizing, setResizing] = useState(false);

  useEffect(() => setLoadFailed(false), [item.id]);

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
        <div>
          <span>{labels.preview}</span>
          <strong title={item.name}>{item.name}</strong>
        </div>
        <div className={styles.actions}>
          <IconButton
            label={pinned ? labels.unpin : labels.pin}
            onClick={() => onPinnedChange(!pinned)}
            pressed={pinned}
          >
            {pinned ? (
              <PushPinSlash aria-hidden="true" size={19} weight="fill" />
            ) : (
              <PushPin aria-hidden="true" size={19} />
            )}
          </IconButton>
          <IconButton label={labels.close} onClick={onClose}>
            <X aria-hidden="true" size={19} />
          </IconButton>
        </div>
      </header>

      <div className={styles.stage}>
        {loadFailed ? (
          <div className={styles.failed} role="status">
            {item.kind === "video" ? (
              <FilmSlate aria-hidden="true" size={38} />
            ) : (
              <FileImage aria-hidden="true" size={38} />
            )}
            <span>{failedLabel}</span>
          </div>
        ) : item.kind === "video" ? (
          <video
            aria-label={item.name}
            controls
            key={item.id}
            onError={() => setLoadFailed(true)}
            playsInline
            preload="metadata"
            src={item.contentUrl}
          />
        ) : (
          <img
            alt={item.name}
            onError={() => setLoadFailed(true)}
            src={item.contentUrl}
          />
        )}
      </div>

      <nav className={styles.navigation} aria-label={labels.preview}>
        <IconButton
          disabled={!canGoPrevious}
          label={labels.previous}
          onClick={onPrevious}
        >
          <CaretLeft aria-hidden="true" size={19} />
        </IconButton>
        <span aria-live="polite">{labels.position}</span>
        <IconButton
          disabled={!canGoNext}
          label={labels.next}
          onClick={onNext}
        >
          <CaretRight aria-hidden="true" size={19} />
        </IconButton>
      </nav>

      <Button
        className={styles.openViewer}
        onClick={onOpenViewer}
        size="small"
        variant="secondary"
      >
        <ArrowsOut aria-hidden="true" size={18} />
        {labels.openViewer}
      </Button>

      <dl className={styles.details}>
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
