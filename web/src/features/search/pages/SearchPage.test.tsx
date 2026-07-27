import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  searchAssets,
  searchLibraryAssets,
  type Asset,
} from "../../../lib/api/catalog";
import { getLibrary } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { SearchPage } from "./SearchPage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/catalog")>();
  return {
    ...actual,
    searchAssets: vi.fn(),
    searchLibraryAssets: vi.fn(),
  };
});

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/libraries")>();
  return {
    ...actual,
    getLibrary: vi.fn(),
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

const result: Asset = {
  directoryId: "dir_japan",
  durationMs: null,
  height: 800,
  id: "ast_kyoto",
  kind: "image",
  libraryId: "lib_family",
  libraryName: "家庭影像",
  mimeType: "image/jpeg",
  modifiedAt: "2026-07-27T00:00:00Z",
  name: "京都夜景.jpg",
  playbackStatus: "not_applicable",
  probeStatus: "ready",
  relativePath: "旅行/日本/京都夜景.jpg",
  sizeBytes: 512,
  sourceAvailability: "available",
  thumbnail: { errorCode: null, status: "pending", url: null },
  width: 1200,
};

beforeEach(() => {
  vi.mocked(getLibrary).mockReset();
  vi.mocked(searchAssets).mockReset();
  vi.mocked(searchLibraryAssets).mockReset();
  vi.mocked(getLibrary).mockResolvedValue({
    etag: '"library-v1"',
    library: {
      assetCount: 8,
      directoryCount: 3,
      displayPath: "/library/family",
      id: "lib_family",
      lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
      latestScanId: "scan_test",
      name: "家庭影像",
      status: "ready",
    },
  });
  vi.mocked(searchLibraryAssets).mockResolvedValue({
    items: [result],
    nextCursor: null,
  });
  vi.mocked(searchAssets).mockResolvedValue({
    items: [result],
    nextCursor: null,
  });
});

it("restores a library search and exposes the source library and directory", async () => {
  renderSearch("/libraries/lib_family/search?q=京都");

  expect(
    await screen.findByRole("heading", { name: "搜索媒体" }),
  ).toBeVisible();
  expect(screen.getByRole("searchbox", { name: "搜索文件名或路径" })).toHaveValue(
    "京都",
  );
  expect(await screen.findByText("京都夜景.jpg")).toBeVisible();
  expect(
    screen.getByRole("link", { name: "家庭影像 · 旅行/日本" }),
  ).toHaveAttribute("href", "/libraries/lib_family/browse/dir_japan");
  expect(searchLibraryAssets).toHaveBeenCalledWith(
    expect.objectContaining({
      libraryId: "lib_family",
      order: "desc",
      q: "京都",
      sort: "modifiedAt",
    }),
  );
});

it("moves scope and filters into the URL and selects the global endpoint", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?q=京都");

  await screen.findByText("京都夜景.jpg");
  await user.selectOptions(screen.getByLabelText("范围"), "all");
  await user.selectOptions(screen.getByLabelText("媒体类型"), "video");

  await waitFor(() =>
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/libraries/lib_family/search?q=%E4%BA%AC%E9%83%BD&scope=all&kind=video",
    ),
  );
  expect(searchAssets).toHaveBeenCalledWith(
    expect.objectContaining({
      kinds: ["video"],
      q: "京都",
    }),
  );
});

it("preserves an empty query state and clears only active filters", async () => {
  const user = userEvent.setup();
  vi.mocked(searchLibraryAssets).mockResolvedValue({
    items: [],
    nextCursor: null,
  });
  renderSearch("/libraries/lib_family/search?q=不存在&kind=video");

  expect(await screen.findByText("没有搜索结果")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "清除筛选" }));

  await waitFor(() =>
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/libraries/lib_family/search?q=%E4%B8%8D%E5%AD%98%E5%9C%A8",
    ),
  );
  expect(screen.getByRole("searchbox")).toHaveValue("不存在");
});

function renderSearch(initialEntry: string) {
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
          <MemoryRouter initialEntries={[initialEntry]}>
            <SearchPage libraryId="lib_family" session={session} />
            <LocationProbe />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

function LocationProbe() {
  const location = useLocation();
  return (
    <output data-testid="location">
      {location.pathname}
      {location.search}
    </output>
  );
}
