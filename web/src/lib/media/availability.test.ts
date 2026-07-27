import { describe, expect, it } from "vitest";

import type { Asset } from "../api/catalog";
import { mediaAvailability, mediaPosterUrl } from "./availability";

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
    expect(mediaAvailability({ ...asset, sourceAvailability: "unreadable" })).toBe(
      "unreadable",
    );
  });

  it("maps corrupt, unsupported, and codec states without blocking ready GIFs", () => {
    expect(mediaAvailability({ ...asset, probeStatus: "failed" })).toBe("invalid");
    expect(mediaAvailability({ ...asset, probeStatus: "unsupported" })).toBe(
      "unsupported",
    );
    expect(
      mediaAvailability({
        ...asset,
        kind: "video",
        playbackStatus: "unsupported_codec",
      }),
    ).toBe("unsupportedCodec");
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
});
