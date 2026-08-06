import { act, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";

import { useMediaPreviewController } from "./useMediaPreviewController";

const items = [{ id: "deep-asset" }, { id: "next-asset" }];

afterEach(() => {
  vi.unstubAllGlobals();
  window.localStorage.clear();
});

it("restores the activated virtual card after the preview opens and reflows the grid", () => {
  const frames: FrameRequestCallback[] = [];
  vi.stubGlobal(
    "requestAnimationFrame",
    vi.fn((callback: FrameRequestCallback) => {
      frames.push(callback);
      return frames.length;
    }),
  );
  const restoreItem = vi.fn();
  const { result } = renderHook(() =>
    useMediaPreviewController({
      items,
      resetKey: "library:directory",
    }),
  );
  result.current.collectionRef.current = { restoreItem };

  act(() => result.current.activate("deep-asset", "single"));
  expect(result.current.previewItem?.id).toBe("deep-asset");
  expect(restoreItem).not.toHaveBeenCalled();

  act(() => frames.shift()?.(0));
  expect(restoreItem).not.toHaveBeenCalled();
  act(() => frames.shift()?.(16));
  expect(restoreItem).toHaveBeenCalledWith("deep-asset");

  act(() => result.current.activate("next-asset", "single"));
  expect(frames).toHaveLength(0);
});

it("reads independent video autoplay and mute preferences from the shared owner", () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    JSON.stringify({ previewAutoplay: false, previewMuted: false }),
  );

  const { result } = renderHook(() =>
    useMediaPreviewController({
      items,
      resetKey: "library:directory",
    }),
  );

  expect(result.current.autoPlayVideo).toBe(false);
  expect(result.current.muteVideo).toBe(false);
});
