import { QueryClient } from "@tanstack/react-query";
import { expect, it } from "vitest";

import type { AssetPage } from "../../lib/api/catalog";
import {
  catalogKeys,
  pendingThumbnailRefreshInterval,
  pendingThumbnailRefreshMs,
  pendingThumbnailRefreshPageBudget,
  refreshCatalogScope,
} from "./queries";

function page(status: "pending" | "ready" | "failed" | "unavailable"): AssetPage {
  return {
    items: [
      {
        directoryId: "dir_test",
        durationMs: null,
        height: 800,
        id: "ast_test",
        kind: "image",
        libraryId: "lib_test",
        libraryName: "Test",
        mimeType: "image/jpeg",
        modifiedAt: "2026-07-28T00:00:00Z",
        name: "test.jpg",
        playbackStatus: "not_applicable",
        probeStatus: "ready",
        relativePath: "test.jpg",
        sizeBytes: 512,
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
        thumbnail: {
          errorCode: status === "failed" ? "thumbnail_failed" : null,
          status,
          url:
            status === "ready"
              ? "/api/v1/assets/ast_test/thumbnail?variant=grid"
              : null,
        },
        width: 1200,
      },
    ],
    nextCursor: null,
  };
}

it("polls only while at least one indexed thumbnail remains pending", () => {
  expect(pendingThumbnailRefreshInterval([page("pending")])).toBe(
    pendingThumbnailRefreshMs,
  );
  expect(pendingThumbnailRefreshInterval([page("ready"), page("failed")])).toBe(
    false,
  );
  expect(pendingThumbnailRefreshInterval(undefined)).toBe(false);
  const storyboardPending = page("ready");
  storyboardPending.items[0]!.storyboard.status = "pending";
  expect(pendingThumbnailRefreshInterval([storyboardPending])).toBe(
    pendingThumbnailRefreshMs,
  );
});

it("stops periodic page refetches before a large collection can create a request storm", () => {
  const boundedPages = Array.from(
    { length: pendingThumbnailRefreshPageBudget },
    () => page("pending"),
  );
  expect(pendingThumbnailRefreshInterval(boundedPages)).toBe(
    pendingThumbnailRefreshMs,
  );
  expect(
    pendingThumbnailRefreshInterval([...boundedPages, page("pending")]),
  ).toBe(false);
});

it("drops loaded cursor pages before refreshing the current scope", async () => {
  const queryClient = new QueryClient();
  const queryKey = catalogKeys.assets(
    "lib_test",
    "dir_test",
    false,
    undefined,
    "name",
    "asc",
  );
  queryClient.setQueryData(queryKey, {
    pageParams: [undefined, "cursor-2"],
    pages: [page("ready"), page("ready")],
  });

  await refreshCatalogScope(queryClient, {
    directoryId: "dir_test",
    libraryId: "lib_test",
    order: "asc",
    recursive: false,
    sort: "name",
  });

  expect(
    queryClient.getQueryData<{ pages: AssetPage[] }>(queryKey)?.pages,
  ).toHaveLength(1);
});
