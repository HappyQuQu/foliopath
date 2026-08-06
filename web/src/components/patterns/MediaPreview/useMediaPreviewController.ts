import { useEffect, useRef, useState } from "react";

import {
  readPreviewAutoplayPreference,
  readPreviewMutedPreference,
  readPreviewPinnedPreference,
  readPreviewWidthPreference,
  writePreviewPinnedPreference,
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
  const [autoPlayVideo] = useState(readPreviewAutoplayPreference);
  const [muteVideo] = useState(readPreviewMutedPreference);
  const [pinned, setPinned] = useState(readPreviewPinnedPreference);
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
    setPinned(readPreviewPinnedPreference());
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
      const isOpeningPreview = !previewItem;
      setPreviewItem(item);
      if (isOpeningPreview) restoreAfterLayout(itemId);
    }
  }

  function updatePinned(nextPinned: boolean) {
    setPinned(nextPinned);
    writePreviewPinnedPreference(nextPinned);
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
    writePreviewPinnedPreference(false);
    if (!restoreItemId) return;
    restoreAfterLayout(restoreItemId);
  }

  function restoreAfterLayout(itemId: string) {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        collectionRef.current?.restoreItem(itemId);
      });
    });
  }

  function setWidth(nextWidth: number) {
    setWidthState(nextWidth);
    writePreviewWidthPreference(nextWidth);
  }

  return {
    activate,
    autoPlayVideo,
    close,
    collectionRef,
    maxWidth,
    moveTo,
    muteVideo,
    pinned,
    previewIndex,
    previewItem,
    selectedItemId,
    setWidth,
    updatePinned,
    width,
  };
}
