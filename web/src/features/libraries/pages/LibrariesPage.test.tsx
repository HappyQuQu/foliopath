import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  listMediaFailures,
  repairLibraryMediaProcessing,
} from "../../../lib/api/diagnostics";
import { listLibraries } from "../../../lib/api/libraries";
import { getLibraryMediaProcessingProgress } from "../../../lib/api/media-processing";
import { getScan, listLibraryScans } from "../../../lib/api/scans";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { LibrariesPage } from "./LibrariesPage";

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/libraries")>();
  return {
    ...actual,
    listLibraries: vi.fn(),
  };
});

vi.mock("../../../lib/api/media-processing", async (importOriginal) => {
  const actual = await importOriginal<
    typeof import("../../../lib/api/media-processing")
  >();
  return {
    ...actual,
    getLibraryMediaProcessingProgress: vi.fn(),
  };
});

vi.mock("../../../lib/api/diagnostics", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/diagnostics")>();
  return {
    ...actual,
    listMediaFailures: vi.fn(),
    repairLibraryMediaProcessing: vi.fn(),
  };
});

vi.mock("../../../lib/api/scans", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/scans")>();
  return {
    ...actual,
    getScan: vi.fn(),
    listLibraryScans: vi.fn(),
  };
});

const session: AuthenticatedSession = {
  administrator: {
    displayName: "家庭管理员",
    id: "adm_test",
    username: "admin",
  },
  csrfToken: "csrf-token-that-is-long-enough-for-the-contract",
  expiresAt: "2026-08-04T00:00:00Z",
};

beforeEach(() => {
  vi.mocked(listLibraries).mockReset();
  vi.mocked(listLibraries).mockResolvedValue({
    items: [
      {
        assetCount: 9_983,
        directoryCount: 13,
        displayPath: "/library/temp",
        id: "lib_1",
        lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
        latestScanId: "scan_1",
        name: "temp",
        status: "ready",
      },
    ],
    nextCursor: null,
  });
  vi.mocked(getScan).mockResolvedValue({
    canCancel: false,
    cancelRequestedAt: null,
    createdAt: "2026-07-28T00:00:00Z",
    discoveredAssets: 9_983,
    discoveredDirectories: 13,
    errorCode: null,
    errorCount: 2,
    finishedAt: "2026-07-28T00:02:00Z",
    id: "scan_1",
    issues: [],
    issuesTruncated: false,
    libraryId: "lib_1",
    phase: "completed",
    processedAssets: 9_983,
    progressRatio: 1,
    skippedDirectories: 0,
    skippedFiles: 0,
    startedAt: "2026-07-28T00:00:00Z",
    status: "succeeded",
    trigger: "manual",
  });
  vi.mocked(listLibraryScans).mockResolvedValue({
    items: [awaitedScan()],
    nextCursor: null,
  });
  vi.mocked(listMediaFailures).mockResolvedValue({
    items: [],
    nextCursor: null,
    revision: null,
  });
  vi.mocked(repairLibraryMediaProcessing).mockReset();
  vi.mocked(repairLibraryMediaProcessing).mockResolvedValue({
    permanentFailures: 0,
    remainingEligible: 0,
    requeued: 2,
  });
  vi.mocked(getLibraryMediaProcessingProgress).mockResolvedValue({
    active: false,
    thumbnails: {
      failed: 2,
      processed: 9_983,
      queued: 0,
      running: 0,
      succeeded: 9_981,
      total: 9_983,
    },
    videoPreviews: {
      failed: 0,
      processed: 120,
      queued: 0,
      running: 0,
      succeeded: 120,
      total: 120,
    },
    videoPreviewsPendingEligibility: 0,
  });
});

it("moves scan history and processing results into library tabs", async () => {
  const user = userEvent.setup();
  renderLibraries();

  expect(await screen.findByRole("tab", { name: "媒体库" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await user.click(screen.getByRole("tab", { name: "扫描记录" }));
  expect(await screen.findByText(/手动操作/)).toBeVisible();
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/settings/libraries?view=scans",
  );

  await user.click(screen.getByRole("tab", { name: "处理结果" }));
  expect(await screen.findByText("没有失败结果")).toBeVisible();
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/settings/libraries?view=results",
  );
});

it("expands status details below the current row without leaving the library list", async () => {
  const user = userEvent.setup();
  renderLibraries();

  await user.click(await screen.findByRole("button", { name: "查看状态" }));

  expect(
    screen.getByRole("region", { name: "“temp”状态详情" }),
  ).toBeVisible();
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/settings/libraries",
  );
  expect(await screen.findByText("最近一次完整扫描")).toBeVisible();
  expect(screen.getByText("缩略图与视频封面")).toBeVisible();
  expect(screen.getByRole("button", { name: "补齐缺失" })).toBeVisible();
  expect(screen.getByRole("button", { name: "全部重建" })).toBeVisible();
  expect(screen.getByRole("button", { name: "重新扫描" })).toBeVisible();
  expect(
    screen.queryByRole("button", { name: "打开完整状态页" }),
  ).not.toBeInTheDocument();

  await user.click(screen.getByRole("button", { name: "收起状态" }));
  expect(
    screen.queryByRole("region", { name: "“temp”状态详情" }),
  ).not.toBeInTheDocument();
});

it("distinguishes missing media from a full derived-media rebuild", async () => {
  const user = userEvent.setup();
  renderLibraries();

  await user.click(await screen.findByRole("button", { name: "查看状态" }));
  await user.click(screen.getByRole("button", { name: "补齐缺失" }));
  expect(repairLibraryMediaProcessing).toHaveBeenLastCalledWith(
    {
      csrfToken: session.csrfToken,
      libraryId: "lib_1",
      mode: "missing",
    },
    expect.anything(),
  );

  await waitFor(() =>
    expect(screen.getByRole("button", { name: "全部重建" })).toBeEnabled(),
  );
  await user.click(screen.getByRole("button", { name: "全部重建" }));
  const confirmation = screen.getByRole("dialog", { name: "确认全部重建" });
  await user.click(within(confirmation).getByRole("button", { name: "全部重建" }));
  expect(repairLibraryMediaProcessing).toHaveBeenLastCalledWith(
    {
      csrfToken: session.csrfToken,
      libraryId: "lib_1",
      mode: "all",
    },
    expect.anything(),
  );
});

it("uses the library action as the only browse navigation after indexing", async () => {
  const user = userEvent.setup();
  renderLibraries();

  expect(await screen.findByRole("button", { name: "浏览" })).toBeVisible();
  expect(screen.queryByRole("link", { name: "浏览" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "返回浏览" })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "FolioPath" })).toHaveAttribute(
    "href",
    "/",
  );

  await user.click(screen.getByRole("button", { name: "浏览" }));
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_1/browse",
  );
});

function renderLibraries() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  return render(
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <MemoryRouter initialEntries={["/settings/libraries"]}>
            <LibrariesPage session={session} />
            <LocationProbe />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname}{location.search}</output>;
}

function awaitedScan() {
  return {
    canCancel: false,
    cancelRequestedAt: null,
    createdAt: "2026-07-28T00:00:00Z",
    discoveredAssets: 9_983,
    discoveredDirectories: 13,
    errorCode: null,
    errorCount: 2,
    finishedAt: "2026-07-28T00:02:00Z",
    id: "scan_1",
    issues: [],
    issuesTruncated: false,
    libraryId: "lib_1",
    phase: "completed" as const,
    processedAssets: 9_983,
    progressRatio: 1,
    skippedDirectories: 0,
    skippedFiles: 0,
    startedAt: "2026-07-28T00:00:00Z",
    status: "succeeded" as const,
    trigger: "manual" as const,
  };
}
