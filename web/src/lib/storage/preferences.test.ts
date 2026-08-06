import { expect, it } from "vitest";

import {
  readAcknowledgedMediaFailureRevision,
  readMediaLayoutPreference,
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
});
