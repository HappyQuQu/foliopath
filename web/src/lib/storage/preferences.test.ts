import { expect, it } from "vitest";

import {
  readAcknowledgedMediaFailureRevision,
  readClearedMediaFailureRevision,
  readMediaLayoutPreference,
  readMediaTileSizePreference,
  writeMediaTileSizePreference,
  readMediaSortPreference,
  readPreviewAutoplayPreference,
  readPreviewMutedPreference,
  readPreviewWidthPreference,
  readSidebarWidthPreference,
  writeMediaLayoutPreference,
  writeMediaSortPreference,
  writePreviewAutoplayPreference,
  writePreviewMutedPreference,
  writeAcknowledgedMediaFailureRevision,
  writeClearedMediaFailureRevision,
  clearClearedMediaFailureRevision,
  writePreviewWidthPreference,
  writeSidebarWidthPreference,
} from "./preferences";

it("defaults an absent or invalid media layout preference to grid", () => {
  expect(readMediaLayoutPreference()).toBe("grid");

  window.localStorage.setItem(
    "foliopath.preferences.v1",
    JSON.stringify({ mediaLayout: "columns" }),
  );

  expect(readMediaLayoutPreference()).toBe("grid");
});

it("remembers the selected media layout without replacing other preferences", () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    JSON.stringify({ locale: "en", theme: "dark" }),
  );

  writeMediaLayoutPreference("masonry");

  expect(readMediaLayoutPreference()).toBe("masonry");
  expect(JSON.parse(window.localStorage.getItem("foliopath.preferences.v1") ?? "{}"))
    .toEqual({
      locale: "en",
      mediaLayout: "masonry",
      theme: "dark",
    });
});

it("defaults sorting to the current view and remembers an explicit default", () => {
  expect(readMediaSortPreference()).toBe("contextual");

  writeMediaSortPreference("size:desc");

  expect(readMediaSortPreference()).toBe("size:desc");
});

it("remembers seamless masonry without replacing other preferences", () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    JSON.stringify({ locale: "en", mediaSort: "name:asc" }),
  );
  writeMediaLayoutPreference("seamless");
  expect(readMediaLayoutPreference()).toBe("seamless");
  expect(JSON.parse(window.localStorage.getItem("foliopath.preferences.v1") ?? "{}"))
    .toMatchObject({ locale: "en", mediaSort: "name:asc", mediaLayout: "seamless" });
});

it("reads and writes remembered panel widths", () => {
  writeSidebarWidthPreference(320);
  writePreviewWidthPreference(520);

  expect(readSidebarWidthPreference()).toBe(320);
  expect(readPreviewWidthPreference()).toBe(520);
});

it("defaults video preview autoplay on and remembers when it is disabled", () => {
  expect(readPreviewAutoplayPreference()).toBe(true);

  writePreviewAutoplayPreference(false);

  expect(readPreviewAutoplayPreference()).toBe(false);
});

it("defaults video previews to muted and remembers an audible preference", () => {
  expect(readPreviewMutedPreference()).toBe(true);

  writePreviewMutedPreference(false);

  expect(readPreviewMutedPreference()).toBe(false);
});

it("stores only a valid acknowledged media failure watermark", () => {
  writeAcknowledgedMediaFailureRevision("mfailrev_1000_42");
  expect(readAcknowledgedMediaFailureRevision()).toBe("mfailrev_1000_42");

  writeAcknowledgedMediaFailureRevision("../../42");
  expect(readAcknowledgedMediaFailureRevision()).toBe("mfailrev_1000_42");

  writeClearedMediaFailureRevision("mfailrev_2000_43");
  expect(readClearedMediaFailureRevision()).toBe("mfailrev_2000_43");
  writeClearedMediaFailureRevision("not-a-revision");
  expect(readClearedMediaFailureRevision()).toBe("mfailrev_2000_43");
  clearClearedMediaFailureRevision();
  expect(readClearedMediaFailureRevision()).toBeUndefined();
});

it("remembers thumbnail size and rejects corrupt values without losing other preferences", () => {
  window.localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ mediaLayout: "seamless", mediaTileSize: "huge" }));
  expect(readMediaTileSizePreference()).toBe(210);
  writeMediaTileSizePreference(350);
  expect(readMediaTileSizePreference()).toBe(350);
  expect(readMediaLayoutPreference()).toBe("seamless");
  for (const invalid of [NaN, Infinity, 0, 481]) writeMediaTileSizePreference(invalid);
  expect(readMediaTileSizePreference()).toBe(350);
});
