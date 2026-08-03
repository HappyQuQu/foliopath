import { describe, expect, it } from "vitest";

import type { MediaProcessingProgress } from "../../lib/api/media-processing";
import {
  mediaProcessingIsActive,
  mediaProcessingRefreshInterval,
} from "./media-processing-queries";

const terminal: MediaProcessingProgress = {
  active: false,
  thumbnails: {
    total: 2,
    processed: 2,
    queued: 0,
    running: 0,
    succeeded: 2,
    failed: 0,
  },
  videoPreviews: {
    total: 1,
    processed: 1,
    queued: 0,
    running: 0,
    succeeded: 1,
    failed: 0,
  },
  videoPreviewsPendingEligibility: 0,
};

describe("mediaProcessingRefreshInterval", () => {
  it("keeps polling while scanning or derived work is active", () => {
    expect(mediaProcessingRefreshInterval(terminal, true)).toBe(1_500);
    expect(
      mediaProcessingRefreshInterval({ ...terminal, active: true }, false),
    ).toBe(1_500);
  });

  it("stops after both scanning and derived work are terminal", () => {
    expect(mediaProcessingRefreshInterval(terminal, false)).toBe(false);
  });

  it("treats queued work as active even if a stale active flag is false", () => {
    const queued = {
      ...terminal,
      thumbnails: { ...terminal.thumbnails, processed: 1, queued: 1 },
    };
    expect(mediaProcessingIsActive(queued)).toBe(true);
    expect(mediaProcessingRefreshInterval(queued, false)).toBe(1_500);
  });
});
