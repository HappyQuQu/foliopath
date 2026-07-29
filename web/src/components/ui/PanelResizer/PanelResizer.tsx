import {
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
} from "react";

import styles from "./PanelResizer.module.css";

const widthStep = 24;

export function PanelResizer({
  ariaLabel,
  className,
  growDirection,
  max,
  min,
  onChange,
  value,
}: {
  ariaLabel: string;
  className?: string | undefined;
  growDirection: "left" | "right";
  max: number;
  min: number;
  onChange: (width: number) => void;
  value: number;
}) {
  const dragStart = useRef<{ pointerX: number; width: number } | undefined>(
    undefined,
  );
  const removeDragListeners = useRef<(() => void) | undefined>(undefined);
  const [resizing, setResizing] = useState(false);

  function clamp(nextWidth: number) {
    onChange(Math.min(max, Math.max(min, nextWidth)));
  }

  function finishResize() {
    dragStart.current = undefined;
    setResizing(false);
    document.body.style.removeProperty("cursor");
    document.body.style.removeProperty("user-select");
  }

  useEffect(
    () => () => {
      removeDragListeners.current?.();
      document.body.style.removeProperty("cursor");
      document.body.style.removeProperty("user-select");
    },
    [],
  );

  function handleMouseDown(event: MouseEvent<HTMLDivElement>) {
    if (event.button !== 0) return;
    event.preventDefault();
    dragStart.current = { pointerX: event.clientX, width: value };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    setResizing(true);

    const handleMove = (moveEvent: globalThis.MouseEvent) => {
      if (!dragStart.current) return;
      const pointerDelta = moveEvent.clientX - dragStart.current.pointerX;
      const widthDelta =
        growDirection === "right" ? pointerDelta : -pointerDelta;
      clamp(dragStart.current.width + widthDelta);
    };
    const handleEnd = () => {
      removeDragListeners.current?.();
      removeDragListeners.current = undefined;
      finishResize();
    };
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleEnd);
    removeDragListeners.current = () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleEnd);
    };
  }

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      clamp(value + (growDirection === "left" ? widthStep : -widthStep));
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      clamp(value + (growDirection === "right" ? widthStep : -widthStep));
    } else if (event.key === "Home") {
      event.preventDefault();
      clamp(min);
    } else if (event.key === "End") {
      event.preventDefault();
      clamp(max);
    }
  }

  return (
    <div
      aria-label={ariaLabel}
      aria-orientation="vertical"
      aria-valuemax={max}
      aria-valuemin={min}
      aria-valuenow={Math.round(value)}
      className={[
        styles.resizer,
        styles[growDirection],
        className,
      ]
        .filter(Boolean)
        .join(" ")}
      data-resizing={resizing || undefined}
      onKeyDown={handleKeyDown}
      onMouseDown={handleMouseDown}
      role="separator"
      tabIndex={0}
    >
      <span className={styles.handle} />
    </div>
  );
}
