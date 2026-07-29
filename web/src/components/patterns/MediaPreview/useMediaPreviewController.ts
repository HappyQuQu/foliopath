import { useEffect, useRef, useState } from "react";

import {
  readPreviewWidthPreference,
  writePreviewWidthPreference,
} from "../../../lib/storage/preferences";
import type { MediaCollectionHandle } from "../MediaCollection/MediaCollection";

interface PreviewCandidate {
  id: string;
}

export function useMediaPreviewController<T extends PreviewCandidate>({
  items,
  resetKey,
}: {
  items: T[];
  resetKey: string;
}) {
  const [previewItem, setPreviewItem] = useState<T>();
  const [selectedItemId, setSelectedItemId] = useState<string>();
  const [pinned, setPinned] = useState(false);
  const [width, setWidthState] = useState(readPreviewWidthPreference);
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window === "undefined" ? 1280 : window.innerWidth,
  );
  const collectionRef = useRef<MediaCollectionHandle>(null);
  const maxWidth =
    viewportWidth <= 1024
      ? 620
      : Math.max(360, Math.min(620, Math.floor(viewportWidth * 0.48)));
  const previewIndex = previewItem
    ? items.findIndex((item) => item.id === previewItem.id)
    : -1;

  useEffect(() => {
    setPreviewItem(undefined);
    setSelectedItemId(undefined);
    setPinned(false);
  }, [resetKey]);

  useEffect(() => {
    const updateViewportWidth = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", updateViewportWidth);
    return () => window.removeEventListener("resize", updateViewportWidth);
  }, []);

  useEffect(() => {
    setWidthState((currentWidth) => Math.min(currentWidth, maxWidth));
  }, [maxWidth]);

  useEffect(() => {
    setPreviewItem((currentItem) => {
      if (!currentItem) return undefined;
      const currentResult = items.find((item) => item.id === currentItem.id);
      return currentResult ?? (pinned ? currentItem : undefined);
    });
    if (
      !pinned &&
      selectedItemId &&
      !items.some((item) => item.id === selectedItemId)
    ) {
      setSelectedItemId(undefined);
    }
  }, [items, pinned, selectedItemId]);

  function activate(itemId: string, activation: "single" | "double") {
    const item = items.find((candidate) => candidate.id === itemId);
    if (!item) return;
    setSelectedItemId(itemId);
    if (!pinned || activation === "double") {
      setPreviewItem(item);
    }
  }

  function updatePinned(nextPinned: boolean) {
    setPinned(nextPinned);
    if (nextPinned || !selectedItemId) return;
    const selectedItem = items.find((item) => item.id === selectedItemId);
    if (selectedItem) setPreviewItem(selectedItem);
  }

  function moveTo(item: T | undefined) {
    if (!item) return;
    setPreviewItem(item);
    setSelectedItemId(item.id);
  }

  function close() {
    const restoreItemId = previewItem?.id;
    setPreviewItem(undefined);
    setPinned(false);
    if (!restoreItemId) return;
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        collectionRef.current?.restoreItem(restoreItemId);
      });
    });
  }

  function setWidth(nextWidth: number) {
    setWidthState(nextWidth);
    writePreviewWidthPreference(nextWidth);
  }

  return {
    activate,
    close,
    collectionRef,
    maxWidth,
    moveTo,
    pinned,
    previewIndex,
    previewItem,
    selectedItemId,
    setWidth,
    updatePinned,
    width,
  };
}
