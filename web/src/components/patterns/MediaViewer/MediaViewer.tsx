import {
  ArrowsIn,
  CaretLeft,
  CaretRight,
  CornersOut,
  Info,
  MagnifyingGlassMinus,
  MagnifyingGlassPlus,
  X,
} from "@phosphor-icons/react";
import {
  useEffect,
  useRef,
  useState,
  type PointerEvent,
  type WheelEvent,
} from "react";

import { Button, IconButton } from "../../ui";
import {
  MediaAvailabilityState,
  type MediaAvailabilityPresentation,
} from "../MediaAvailabilityState/MediaAvailabilityState";
import styles from "./MediaViewer.module.css";

export interface MediaViewerItem {
  contentUrl: string;
  details: Array<{ label: string; value: string }>;
  id: string;
  kind: "image" | "animated" | "video";
  name: string;
  posterUrl?: string | undefined;
}

export interface MediaViewerLabels {
  close: string;
  exitFullscreen: string;
  fit: string;
  fullscreen: string;
  imageFailed: string;
  info: string;
  information: string;
  loadFailedDescription: string;
  next: string;
  originalSize: string;
  previous: string;
  retry: string;
  shortcutHint: string;
  videoFailed: string;
  zoomIn: string;
  zoomOut: string;
}

const minimumScale = 0.25;
const maximumScale = 4;
const scaleStep = 0.25;

export function MediaViewer({
  canGoNext,
  canGoPrevious,
  item,
  labels,
  availability,
  onClose,
  onNext,
  onPrevious,
  position,
}: {
  canGoNext: boolean;
  canGoPrevious: boolean;
  item: MediaViewerItem;
  labels: MediaViewerLabels;
  availability?: MediaAvailabilityPresentation | undefined;
  onClose: () => void;
  onNext: () => void;
  onPrevious: () => void;
  position: string;
}) {
  const rootRef = useRef<HTMLElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const dragRef = useRef({ originX: 0, originY: 0, pointerX: 0, pointerY: 0 });
  const [fit, setFit] = useState(true);
  const [infoOpen, setInfoOpen] = useState(true);
  const [fullscreen, setFullscreen] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [scale, setScale] = useState(1);
  const [translation, setTranslation] = useState({ x: 0, y: 0 });
  const [dragging, setDragging] = useState(false);
  const imageLike = item.kind !== "video";

  useEffect(() => {
    closeRef.current?.focus();
    if (window.matchMedia?.("(max-width: 48rem)").matches) {
      setInfoOpen(false);
    }
  }, []);

  useEffect(() => {
    setFit(true);
    setLoadFailed(false);
    setLoadAttempt(0);
    setScale(1);
    setTranslation({ x: 0, y: 0 });
  }, [item.id]);

  useEffect(() => {
    const handleFullscreenChange = () =>
      setFullscreen(document.fullscreenElement === rootRef.current);
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () =>
      document.removeEventListener("fullscreenchange", handleFullscreenChange);
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target;
      const isControl =
        target instanceof HTMLElement &&
        Boolean(target.closest("button, a, input, select, textarea, video"));

      if (event.key === "Escape") {
        event.preventDefault();
        if (document.fullscreenElement) {
          void document.exitFullscreen();
        } else {
          onClose();
        }
      } else if (!isControl && event.key === "ArrowLeft" && canGoPrevious) {
        event.preventDefault();
        onPrevious();
      } else if (!isControl && event.key === "ArrowRight" && canGoNext) {
        event.preventDefault();
        onNext();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [canGoNext, canGoPrevious, onClose, onNext, onPrevious]);

  function showFit() {
    setFit(true);
    setScale(1);
    setTranslation({ x: 0, y: 0 });
  }

  function showOriginalSize() {
    setFit(false);
    setScale(1);
    setTranslation({ x: 0, y: 0 });
  }

  function zoomBy(delta: number) {
    setFit(false);
    setScale((current) =>
      Math.min(maximumScale, Math.max(minimumScale, current + delta)),
    );
  }

  function zoomWithWheel(event: WheelEvent<HTMLDivElement>) {
    if (!imageLike || event.deltaY === 0) return;
    event.preventDefault();
    zoomBy(event.deltaY < 0 ? scaleStep : -scaleStep);
  }

  function startPan(event: PointerEvent<HTMLDivElement>) {
    if (!imageLike || fit || event.button !== 0) return;
    dragRef.current = {
      originX: translation.x,
      originY: translation.y,
      pointerX: event.clientX,
      pointerY: event.clientY,
    };
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
  }

  function movePan(event: PointerEvent<HTMLDivElement>) {
    if (!dragging) return;
    setTranslation({
      x: dragRef.current.originX + event.clientX - dragRef.current.pointerX,
      y: dragRef.current.originY + event.clientY - dragRef.current.pointerY,
    });
  }

  function endPan(event: PointerEvent<HTMLDivElement>) {
    if (!dragging) return;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setDragging(false);
  }

  async function toggleFullscreen() {
    if (document.fullscreenElement) {
      await document.exitFullscreen();
    } else {
      await rootRef.current?.requestFullscreen();
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
  const showImageControls = imageLike && !visibleAvailability;
  const hasDetails = item.details.length > 0;

  return (
    <main className={styles.viewer} ref={rootRef}>
      <header className={styles.header}>
        <Button
          className={styles.close}
          onClick={onClose}
          ref={closeRef}
          size="small"
          variant="quiet"
        >
          <X aria-hidden="true" size={18} />
          {labels.close}
        </Button>
        <strong title={item.name}>{item.name}</strong>
        <div className={styles.actions}>
          {showImageControls && (
            <>
              <IconButton label={labels.fit} onClick={showFit} pressed={fit}>
                <ArrowsIn aria-hidden="true" size={19} />
              </IconButton>
              <IconButton
                label={labels.originalSize}
                onClick={showOriginalSize}
                pressed={!fit && scale === 1}
              >
                <span aria-hidden="true">1:1</span>
              </IconButton>
              <IconButton
                disabled={scale <= minimumScale}
                label={labels.zoomOut}
                onClick={() => zoomBy(-scaleStep)}
              >
                <MagnifyingGlassMinus aria-hidden="true" size={19} />
              </IconButton>
              <IconButton
                disabled={scale >= maximumScale}
                label={labels.zoomIn}
                onClick={() => zoomBy(scaleStep)}
              >
                <MagnifyingGlassPlus aria-hidden="true" size={19} />
              </IconButton>
            </>
          )}
          {hasDetails && (
            <IconButton
              label={labels.info}
              onClick={() => setInfoOpen((current) => !current)}
              pressed={infoOpen}
            >
              <Info aria-hidden="true" size={19} />
            </IconButton>
          )}
          <IconButton
            label={fullscreen ? labels.exitFullscreen : labels.fullscreen}
            onClick={() => void toggleFullscreen()}
          >
            {fullscreen ? (
              <ArrowsIn aria-hidden="true" size={19} />
            ) : (
              <CornersOut aria-hidden="true" size={19} />
            )}
          </IconButton>
        </div>
      </header>

      <IconButton
        className={`${styles.arrow} ${styles.previous}`}
        disabled={!canGoPrevious}
        label={labels.previous}
        onClick={onPrevious}
      >
        <CaretLeft aria-hidden="true" size={25} />
      </IconButton>

      <div
        className={styles.stage}
        data-dragging={dragging || undefined}
        data-fit={fit || undefined}
        onPointerCancel={endPan}
        onPointerDown={startPan}
        onPointerMove={movePan}
        onPointerUp={endPan}
        onWheel={zoomWithWheel}
      >
        {visibleAvailability ? (
          <MediaAvailabilityState state={visibleAvailability} />
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
            draggable={false}
            key={`${item.id}:${loadAttempt}`}
            onError={() => setLoadFailed(true)}
            src={item.contentUrl}
            style={
              fit
                ? undefined
                : {
                    transform: `translate(${translation.x}px, ${translation.y}px) scale(${scale})`,
                  }
            }
          />
        )}
      </div>

      <IconButton
        className={`${styles.arrow} ${styles.next}`}
        disabled={!canGoNext}
        label={labels.next}
        onClick={onNext}
      >
        <CaretRight aria-hidden="true" size={25} />
      </IconButton>

      {infoOpen && hasDetails && (
        <aside aria-label={labels.information} className={styles.info}>
          <h2>{labels.information}</h2>
          <dl>
            {item.details.map((detail) => (
              <div key={detail.label}>
                <dt>{detail.label}</dt>
                <dd title={detail.value}>{detail.value}</dd>
              </div>
            ))}
          </dl>
        </aside>
      )}

      <footer className={styles.footer}>
        <span aria-live="polite">{position}</span>
        <span>{labels.shortcutHint}</span>
        {!fit && showImageControls && <span>{Math.round(scale * 100)}%</span>}
      </footer>
    </main>
  );
}
