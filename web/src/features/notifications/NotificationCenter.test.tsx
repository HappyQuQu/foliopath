import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { useMediaFailuresQuery } from "../diagnostics";
import {
  useLibrariesMediaProcessingProgressQueries,
  useLibrariesQuery,
} from "../libraries";
import { useReleaseInformationQuery } from "../release-info";
import { NotificationCenter } from "./NotificationCenter";

vi.mock("../diagnostics", () => ({ useMediaFailuresQuery: vi.fn() }));
vi.mock("../libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../libraries")>();
  return {
    ...actual,
    useLibrariesMediaProcessingProgressQueries: vi.fn(),
    useLibrariesQuery: vi.fn(),
  };
});
vi.mock("../release-info", () => ({ useReleaseInformationQuery: vi.fn() }));

beforeEach(() => {
  window.localStorage.clear();
  vi.mocked(useLibrariesQuery).mockReturnValue({
    data: {
      pageParams: [undefined],
      pages: [{
        items: [{
          assetCount: 1,
          automaticDiscoveryErrorCode: null,
          automaticDiscoveryStatus: "active",
          contentRevision: 1,
          directoryCount: 1,
          displayPath: "/library/photos",
          id: "lib_1",
          lastAutomaticDiscoveryAt: null,
          lastSuccessfulScanAt: "2026-08-13T02:03:04Z",
          latestScanId: "scan_1",
          name: "照片",
          status: "ready",
        }],
        nextCursor: null,
      }],
    },
  } as never);
  vi.mocked(useLibrariesMediaProcessingProgressQueries).mockReturnValue([]);
  vi.mocked(useMediaFailuresQuery).mockReturnValue({
    data: {
      pageParams: [undefined],
      pages: [{
        items: [{
          assetId: "asset_1",
          attempts: 1,
          errorCode: "invalid_media",
          finishedAt: "2026-08-13T03:04:05Z",
          id: "mjob_1",
          libraryId: "lib_1",
          libraryName: "照片",
          relativePath: "broken.jpg",
          variant: "grid",
        }],
        nextCursor: null,
        revision: "mfailrev_1786590245000_1",
      }],
    },
  } as never);
  vi.mocked(useReleaseInformationQuery).mockReturnValue({
    data: {
      checkedAt: "2026-08-13T04:05:06Z",
      currentVersion: "1.0.0",
      latestVersion: "1.1.0",
      releases: [],
      updateAvailable: true,
    },
  } as never);
});

it("shows concrete times for completed, failed, and update notifications", async () => {
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <NotificationCenter />
    </MemoryRouter>,
  );

  await user.click(screen.getByRole("button", { name: "消息中心" }));

  const formatter = new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "short",
    timeStyle: "short",
  });
  for (const value of [
    "2026-08-13T02:03:04Z",
    "2026-08-13T03:04:05Z",
    "2026-08-13T04:05:06Z",
  ]) {
    expect(screen.getByText(formatter.format(new Date(value)))).toBeVisible();
  }
});
