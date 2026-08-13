import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "./client";
import { listMediaFailures } from "./diagnostics";

vi.mock("./client", () => ({
  apiClient: {
    GET: vi.fn(),
  },
}));

describe("diagnostics adapter", () => {
  beforeEach(() => {
    vi.mocked(apiClient.GET).mockReset();
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: { items: [], nextCursor: null },
      error: undefined,
      response: new Response(),
    } as never);
  });

  it("forwards the selected processing-result filters", async () => {
    await listMediaFailures({
      cursor: "mjob_8",
      errorCode: "media_processing_failed",
      libraryId: "lib_family",
      limit: 25,
      variant: "storyboard",
    });

    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/diagnostics/media-failures",
      {
        params: {
          query: {
            cursor: "mjob_8",
            errorCode: "media_processing_failed",
            libraryId: "lib_family",
            limit: 25,
            variant: "storyboard",
          },
        },
      },
    );
  });
});
