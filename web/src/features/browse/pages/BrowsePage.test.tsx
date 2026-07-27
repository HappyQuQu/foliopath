import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  getDirectory,
  listAssets,
  listDirectories,
} from "../../../lib/api/catalog";
import { getLibrary, listLibraries } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { BrowsePage } from "./BrowsePage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/catalog")>();
  return {
    ...actual,
    getDirectory: vi.fn(),
    listAssets: vi.fn(),
    listDirectories: vi.fn(),
  };
});

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/libraries")>();
  return {
    ...actual,
    getLibrary: vi.fn(),
    listLibraries: vi.fn(),
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
  vi.mocked(getLibrary).mockReset();
  vi.mocked(getDirectory).mockReset();
  vi.mocked(listAssets).mockReset();
  vi.mocked(listDirectories).mockReset();

  vi.mocked(listLibraries).mockResolvedValue({
    items: [
      {
        assetCount: 8,
        directoryCount: 3,
        displayPath: "/library/family",
        id: "lib_family",
        lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
        latestScanId: "scan_test",
        name: "家庭影像",
        status: "ready",
      },
    ],
    nextCursor: null,
  });
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
  vi.mocked(getDirectory).mockResolvedValue({
    breadcrumbs: [
      { id: "dir_root", name: "家庭影像", relativePath: "" },
      { id: "dir_travel", name: "旅行", relativePath: "旅行" },
    ],
    directAssetCount: 2,
    hasChildren: true,
    id: "dir_travel",
    libraryId: "lib_family",
    name: "旅行",
    parentId: "dir_root",
    recursiveAssetCount: 8,
    relativePath: "旅行",
  });
  vi.mocked(listAssets).mockImplementation(async ({ recursive }) => ({
    items: recursive
      ? [
          {
            directoryId: "dir_japan",
            durationMs: 18_000,
            height: 1080,
            id: "ast_clip",
            kind: "video",
            libraryId: "lib_family",
            libraryName: "家庭影像",
            mimeType: "video/mp4",
            modifiedAt: "2026-07-28T00:00:00Z",
            name: "clip.mp4",
            playbackStatus: "playable",
            probeStatus: "ready",
            relativePath: "旅行/日本/clip.mp4",
            sizeBytes: 1024,
            sourceAvailability: "available",
            thumbnail: { errorCode: null, status: "pending", url: null },
            width: 1920,
          },
        ]
      : [
          {
            directoryId: "dir_travel",
            durationMs: null,
            height: 800,
            id: "ast_photo",
            kind: "image",
            libraryId: "lib_family",
            libraryName: "家庭影像",
            mimeType: "image/jpeg",
            modifiedAt: "2026-07-27T00:00:00Z",
            name: "photo.jpg",
            playbackStatus: "not_applicable",
            probeStatus: "ready",
            relativePath: "旅行/photo.jpg",
            sizeBytes: 512,
            sourceAvailability: "available",
            thumbnail: { errorCode: null, status: "pending", url: null },
            width: 1200,
          },
        ],
    nextCursor: null,
  }));
  vi.mocked(listDirectories).mockImplementation(async ({ parentId }) => ({
    items:
      parentId === "dir_travel"
        ? [
            {
              directAssetCount: 1,
              hasChildren: false,
              id: "dir_japan",
              libraryId: "lib_family",
              name: "日本",
              parentId: "dir_travel",
              recursiveAssetCount: 4,
              relativePath: "旅行/日本",
            },
          ]
        : [
            {
              directAssetCount: 2,
              hasChildren: true,
              id: "dir_travel",
              libraryId: "lib_family",
              name: "旅行",
              parentId: "dir_root",
              recursiveAssetCount: 8,
              relativePath: "旅行",
            },
          ],
    nextCursor: null,
  }));
});

it("restores a deep directory, exposes breadcrumbs, and lazily expands its tree path", async () => {
  const user = userEvent.setup();

  renderBrowse();

  expect(await screen.findByRole("heading", { name: "旅行", level: 1 })).toBeVisible();
  expect(screen.getByRole("navigation", { name: "目录位置" })).toHaveTextContent(
    "家庭影像旅行",
  );
  expect(
    screen.getByRole("link", { name: /^日本4 项$/ }),
  ).toHaveAttribute("href", "/libraries/lib_family/browse/dir_japan");
  expect(
    screen.getByRole("link", { name: "家庭影像" }),
  ).toHaveAttribute("href", "/libraries/lib_family/browse");

  const collapse = await screen.findByRole("button", { name: "收起目录 旅行" });
  const tree = screen.getByRole("navigation", { name: "媒体库目录" });
  await user.click(collapse);
  expect(within(tree).queryByRole("link", { name: "日本" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "展开目录 旅行" }));
  expect(await within(tree).findByRole("link", { name: "日本" })).toBeVisible();
});

it("restores recursive scope, changes its default sort, and closes recursion from a source link", async () => {
  const user = userEvent.setup();

  renderBrowse("?recursive=1");

  const gridLayout = screen.getByRole("button", { name: "自适应网格" });
  const masonryLayout = screen.getByRole("button", { name: "瀑布流" });
  expect(gridLayout).toHaveAttribute("aria-pressed", "true");
  await user.click(masonryLayout);
  expect(masonryLayout).toHaveAttribute("aria-pressed", "true");
  expect(
    JSON.parse(
      window.localStorage.getItem("foliopath.preferences.v1") ?? "{}",
    ),
  ).toMatchObject({ mediaLayout: "masonry" });

  const recursiveToggle = await screen.findByRole("button", {
    name: "包含子目录",
  });
  expect(recursiveToggle).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("combobox", { name: "排序" })).toHaveValue(
    "modifiedAt:desc",
  );
  expect(await screen.findByText("clip.mp4")).toBeVisible();
  expect(screen.getByRole("link", { name: "来源：旅行/日本" })).toHaveAttribute(
    "href",
    "/libraries/lib_family/browse/dir_japan",
  );
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse/dir_travel?recursive=1",
  );

  await user.click(recursiveToggle);

  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse/dir_travel",
  );
  expect(await screen.findByText("photo.jpg")).toBeVisible();
  expect(vi.mocked(listAssets)).toHaveBeenLastCalledWith(
    expect.objectContaining({
      directoryId: "dir_travel",
      order: "asc",
      recursive: false,
      sort: "name",
    }),
  );
});

it("uses a stable gallery skeleton while the first media page is loading", async () => {
  vi.mocked(listAssets).mockReturnValue(new Promise(() => undefined));

  renderBrowse();

  expect(
    await screen.findByRole("status", { name: "正在载入媒体…" }),
  ).toBeVisible();
  expect(screen.queryByText("当前目录没有媒体")).not.toBeInTheDocument();
});

it("keeps an offline library distinct from an empty reliable index", async () => {
  vi.mocked(getLibrary).mockResolvedValue({
    etag: '"library-offline"',
    library: {
      assetCount: 0,
      directoryCount: 0,
      displayPath: "/library/family",
      id: "lib_family",
      lastSuccessfulScanAt: null,
      latestScanId: "scan_offline",
      name: "家庭影像",
      status: "offline",
    },
  });
  vi.mocked(listAssets).mockResolvedValue({ items: [], nextCursor: null });

  renderBrowse();

  expect(await screen.findByText("媒体库当前离线")).toBeVisible();
  expect(
    screen.getByText(/这不表示原目录为空/),
  ).toBeVisible();
  expect(screen.queryByText("当前目录没有媒体")).not.toBeInTheDocument();
});

it("recovers a first-page media error through the shared retry action", async () => {
  const user = userEvent.setup();
  vi.mocked(listAssets)
    .mockRejectedValueOnce(new Error("network unavailable"))
    .mockResolvedValueOnce({ items: [], nextCursor: null });

  renderBrowse();

  expect(await screen.findByText("暂时无法读取媒体，请重新尝试。")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "重新尝试" }));
  expect(await screen.findByText("当前目录没有媒体")).toBeVisible();
  expect(listAssets).toHaveBeenCalledTimes(2);
});

function renderBrowse(search = "") {
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
          <MemoryRouter
            initialEntries={[
              `/libraries/lib_family/browse/dir_travel${search}`,
            ]}
          >
            <BrowsePage
              directoryId="dir_travel"
              libraryId="lib_family"
              session={session}
            />
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
