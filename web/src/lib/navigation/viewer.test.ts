import { expect, it } from "vitest";

import type { Asset } from "../api/catalog";
import {
  createViewerLocationState,
  readViewerLocationState,
  readViewerReturnState,
  safeViewerReturnPath,
} from "./viewer";

const assets = [
  { id: "first", libraryId: "lib_family" },
  { id: "second", libraryId: "lib_archive" },
] as Asset[];

it("stores only the stable sequence identity needed by the viewer", () => {
  expect(createViewerLocationState(assets, "/search?q=kyoto")).toEqual({
    returnTo: "/search?q=kyoto",
    sequence: [
      { id: "first", libraryId: "lib_family" },
      { id: "second", libraryId: "lib_archive" },
    ],
  });
});

it("accepts valid viewer and focus state and rejects malformed input", () => {
  expect(
    readViewerLocationState({
      returnTo: "/search",
      sequence: [{ id: "first", libraryId: "lib_family" }],
    }),
  ).toBeDefined();
  expect(readViewerLocationState({ returnTo: "/search", sequence: [{}] })).toBeUndefined();
  expect(
    readViewerReturnState({ restoreFocusAssetId: "first" }),
  ).toEqual({ restoreFocusAssetId: "first" });
  expect(readViewerReturnState({ restoreFocusAssetId: 1 })).toBeUndefined();
});

it("allows only same-origin browse and search returns", () => {
  expect(
    safeViewerReturnPath(
      "/libraries/lib_family/browse/dir_kyoto?recursive=1",
      "lib_family",
    ),
  ).toBe("/libraries/lib_family/browse/dir_kyoto?recursive=1");
  expect(safeViewerReturnPath("/search?q=kyoto", "lib_family")).toBe(
    "/search?q=kyoto",
  );
  expect(
    safeViewerReturnPath(
      "/libraries/lib_other/search?q=kyoto&scope=all",
      "lib_family",
    ),
  ).toBe("/libraries/lib_other/search?q=kyoto&scope=all");
  expect(
    safeViewerReturnPath("//evil.example/search", "lib_family"),
  ).toBe("/libraries/lib_family/browse");
  expect(
    safeViewerReturnPath(
      "/libraries/lib_other/browse?from=elsewhere",
      "lib_family",
    ),
  ).toBe("/libraries/lib_family/browse");
  expect(
    safeViewerReturnPath(
      "/libraries/lib_family/browse-not-a-route",
      "lib_family",
    ),
  ).toBe("/libraries/lib_family/browse");
});
