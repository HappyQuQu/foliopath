import { describe, expect, it } from "vitest";

import type { Asset } from "../api/catalog";
import {
  mediaAvailability,
  mediaDerivedStatePending,
  mediaPosterUrl,
  mediaStoryboard,
} from "./availability";

const asset: Asset = {
  directoryId: "dir",
  durationMs: null,
  height: 800,
  id: "asset",
  kind: "image",
  libraryId: "library",
  libraryName: "Library",
  mimeType: "image/jpeg",
  modifiedAt: "2026-07-28T00:00:00Z",
  name: "photo.jpg",
  playbackStatus: "not_applicable",
  probeStatus: "ready",
  relativePath: "photo.jpg",
  sizeBytes: 1024,
  sourceAvailability: "available",
  storyboard: {
    cellHeight: null,
    cellWidth: null,
    columns: null,
    errorCode: null,
    frameCount: null,
    rows: null,
    status: "not_applicable",
    url: null,
  },
  thumbnail: { errorCode: null, status: "ready", url: "/thumbnail" },
  width: 1200,
};

describe("media availability policy", () => {
  it("prioritizes source availability over derived media status", () => {
    expect(
      mediaAvailability({
        ...asset,
        probeStatus: "failed",
        sourceAvailability: "offline",
      }),
    ).toBe("offline");
    expect(mediaAvailability({ ...asset, sourceAvailability: "missing" })).toBe(
      "missing",
    );
    expect(
      mediaAvailability({ ...asset, sourceAvailability: "unreadable" }),
    ).toBe("unreadable");
  });

  it("maps corrupt and unsupported media without preempting browser video playback", () => {
    expect(mediaAvailability({ ...asset, probeStatus: "failed" })).toBe(
      "invalid",
    );
    expect(mediaAvailability({ ...asset, probeStatus: "unsupported" })).toBe(
      "unsupported",
    );
    expect(
      mediaAvailability({
        ...asset,
        kind: "video",
        playbackStatus: "unsupported_codec",
      }),
    ).toBeUndefined();
    expect(
      mediaAvailability({
        ...asset,
        kind: "animated",
        mimeType: "image/gif",
      }),
    ).toBeUndefined();
  });

  it("uses only a ready thumbnail as the native video poster", () => {
    expect(mediaPosterUrl(asset)).toBe("/thumbnail");
    expect(
      mediaPosterUrl({
        ...asset,
        thumbnail: { errorCode: null, status: "pending", url: null },
      }),
    ).toBeUndefined();
  });

  it("maps only a complete ready storyboard and polls pending derivations", () => {
    const ready: Asset = {
      ...asset,
      kind: "video" as const,
      storyboard: {
        cellHeight: 180,
        cellWidth: 320,
        columns: 5,
        errorCode: null,
        frameCount: 10,
        rows: 2,
        status: "ready" as const,
        url: "/storyboard",
      },
    };
    expect(mediaStoryboard(ready)).toEqual({
      cellHeight: 180,
      cellWidth: 320,
      columns: 5,
      frameCount: 10,
      rows: 2,
      url: "/storyboard",
    });
    expect(
      mediaStoryboard({
        ...ready,
        storyboard: { ...ready.storyboard, columns: null },
      }),
    ).toBeUndefined();
    const invalidFrameCount = {
      ...ready,
      storyboard: { ...ready.storyboard, frameCount: 9 },
    } as unknown as Asset;
    expect(mediaStoryboard(invalidFrameCount)).toBeUndefined();
    expect(
      mediaDerivedStatePending({
        ...asset,
        storyboard: { ...asset.storyboard, status: "pending" },
      }),
    ).toBe(true);
  });
});
