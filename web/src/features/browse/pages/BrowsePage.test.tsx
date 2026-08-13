import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useParams,
} from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { ApiError } from "../../../lib/api/errors";
import {
  getDirectory,
  listAssets,
  listDirectories,
} from "../../../lib/api/catalog";
import { getLibrary, listLibraries } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { BrowsePage } from "./BrowsePage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../lib/api/catalog")>();
  return {
    ...actual,
    getDirectory: vi.fn(),
    listAssets: vi.fn(),
    listDirectories: vi.fn(),
  };
});

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../lib/api/libraries")>();
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

const notApplicableStoryboard = {
  cellHeight: null,
  cellWidth: null,
  columns: null,
  errorCode: null,
  frameCount: null,
  rows: null,
  status: "not_applicable" as const,
  url: null,
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
        automaticDiscoveryErrorCode: null,
        automaticDiscoveryStatus: "active",
        contentRevision: 2,
        directoryCount: 3,
        displayPath: "/library/family",
        id: "lib_family",
        lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
        lastAutomaticDiscoveryAt: "2026-07-28T00:01:00Z",
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
      automaticDiscoveryErrorCode: null,
      automaticDiscoveryStatus: "active",
      contentRevision: 2,
      directoryCount: 3,
      displayPath: "/library/family",
      id: "lib_family",
      lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
      lastAutomaticDiscoveryAt: "2026-07-28T00:01:00Z",
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
    counts: recursive
      ? { all: 8, images: 6, videos: 2 }
      : { all: 2, images: 2, videos: 0 },
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
            storyboard: {
              cellHeight: 180,
              cellWidth: 320,
              columns: 5,
              errorCode: null,
              frameCount: 10,
              rows: 2,
              status: "ready",
              url: "/api/v1/assets/ast_clip/thumbnail?variant=storyboard",
            },
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
            storyboard: notApplicableStoryboard,
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

  expect(
    await screen.findByRole("heading", { name: "旅行", level: 1 }),
  ).toBeVisible();
  expect(
    screen.getByRole("navigation", { name: "目录位置" }),
  ).toHaveTextContent("家庭影像旅行");
  expect(screen.getByRole("link", { name: /^日本4 项$/ })).toHaveAttribute(
    "href",
    "/libraries/lib_family/browse/dir_japan",
  );
  expect(screen.getByRole("link", { name: "家庭影像" })).toHaveAttribute(
    "href",
    "/libraries/lib_family/browse",
  );

  const collapse = await screen.findByRole("button", { name: "收起目录 旅行" });
  const tree = screen.getByRole("navigation", { name: "媒体库目录" });
  await user.click(collapse);
  expect(
    within(tree).queryByRole("link", { name: "日本" }),
  ).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "展开目录 旅行" }));
  expect(await within(tree).findByRole("link", { name: "日本" })).toBeVisible();
});

it("opens the library menu and restores focus when it closes with Escape", async () => {
  const user = userEvent.setup();

  renderBrowse();

  const trigger = await screen.findByRole("button", {
    name: "媒体库：家庭影像",
  });
  await user.click(trigger);

  const menu = screen.getByRole("listbox", { name: "媒体库" });
  expect(menu).toBeVisible();
  expect(
    within(menu).getByRole("option", { name: /家庭影像/ }),
  ).toHaveAttribute("aria-selected", "true");

  await user.keyboard("{Escape}");
  expect(
    screen.queryByRole("listbox", { name: "媒体库" }),
  ).not.toBeInTheDocument();
  expect(trigger).toHaveFocus();
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
    JSON.parse(window.localStorage.getItem("foliopath.preferences.v1") ?? "{}"),
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

it("separates quick-access all media from the library root directory", async () => {
  const user = userEvent.setup();

  renderRootBrowse();

  const tree = await screen.findByRole("navigation", {
    name: "媒体库目录",
  });
  const quickAccess = screen.getByRole("navigation", { name: "快速访问" });
  const allMedia = within(quickAccess).getByRole("link", { name: "全部媒体" });
  const libraryRoot = await within(tree).findByRole("link", {
    name: "家庭影像",
  });

  expect(allMedia).toHaveAttribute(
    "href",
    "/libraries/lib_family/browse?recursive=1&view=all",
  );
  expect(allMedia).not.toHaveAttribute("aria-current");
  expect(libraryRoot).toHaveAttribute("aria-current", "page");

  await user.click(allMedia);

  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse?recursive=1&view=all",
  );
  expect(allMedia).toHaveAttribute("aria-current", "page");
  expect(libraryRoot).not.toHaveAttribute("aria-current");
});

it("keeps the library root current when recursion is enabled there", async () => {
  const user = userEvent.setup();

  renderRootBrowse();

  const tree = await screen.findByRole("navigation", {
    name: "媒体库目录",
  });
  const quickAccess = screen.getByRole("navigation", { name: "快速访问" });
  const allMedia = within(quickAccess).getByRole("link", { name: "全部媒体" });
  const libraryRoot = await within(tree).findByRole("link", {
    name: "家庭影像",
  });

  await user.click(screen.getByRole("button", { name: "包含子目录" }));

  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse?recursive=1",
  );
  expect(libraryRoot).toHaveAttribute("aria-current", "page");
  expect(allMedia).not.toHaveAttribute("aria-current");
});

it("switches between all media, pictures, and videos through URL-bound queries", async () => {
  const user = userEvent.setup();

  renderBrowse();

  let typeFilter = screen.getByRole("radiogroup", { name: "媒体类型" });
  expect(
    await within(typeFilter).findByRole("radio", { name: "全部 2" }),
  ).toHaveAttribute("aria-checked", "true");

  const images = await within(typeFilter).findByRole("radio", {
    name: "图片 2",
  });
  await user.click(images);
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse/dir_travel?kind=image",
  );
  expect(vi.mocked(listAssets)).toHaveBeenLastCalledWith(
    expect.objectContaining({ kinds: ["image", "animated"] }),
  );

  typeFilter = screen.getByRole("radiogroup", { name: "媒体类型" });
  const videos = await within(typeFilter).findByRole("radio", {
    name: "视频 0",
  });
  await user.click(videos);
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse/dir_travel?kind=video",
  );
  expect(vi.mocked(listAssets)).toHaveBeenLastCalledWith(
    expect.objectContaining({ kinds: ["video"] }),
  );

  const callsAfterVideo = vi.mocked(listAssets).mock.calls.length;
  typeFilter = screen.getByRole("radiogroup", { name: "媒体类型" });
  const all = await within(typeFilter).findByRole("radio", {
    name: "全部 2",
  });
  await user.click(all);
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse/dir_travel",
  );
  expect(listAssets).toHaveBeenCalledTimes(callsAfterVideo);
});

it("filters subdirectories and media within the current directory through URL-backed queries", async () => {
  const user = userEvent.setup();

  renderBrowse();

  const filter = screen.getByRole("searchbox", {
    name: "筛选当前目录",
  });
  await user.type(filter, "日本");

  await waitFor(() => {
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/libraries/lib_family/browse/dir_travel?q=%E6%97%A5%E6%9C%AC",
    );
  });
  expect(screen.getByRole("combobox", { name: "排序" })).toHaveValue(
    "modifiedAt:desc",
  );
  expect(vi.mocked(listDirectories)).toHaveBeenCalledWith(
    expect.objectContaining({ parentId: "dir_travel", q: "日本" }),
  );
  expect(vi.mocked(listAssets)).toHaveBeenLastCalledWith(
    expect.objectContaining({
      directoryId: "dir_travel",
      order: "desc",
      q: "日本",
      recursive: false,
      sort: "modifiedAt",
    }),
  );

  await user.clear(filter);
  await waitFor(() => {
    expect(screen.getByRole("combobox", { name: "排序" })).toHaveValue(
      "name:asc",
    );
  });
  expect(screen.getByTestId("location")).toHaveTextContent(
    /^\/libraries\/lib_family\/browse\/dir_travel$/,
  );

  await user.click(screen.getByRole("button", { name: "包含子目录" }));
  await user.type(filter, "日本");

  await waitFor(() => {
    expect(vi.mocked(listAssets)).toHaveBeenLastCalledWith(
      expect.objectContaining({
        directoryId: "dir_travel",
        q: "日本",
        recursive: true,
      }),
    );
  });
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/browse/dir_travel?recursive=1&q=%E6%97%A5%E6%9C%AC",
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
      automaticDiscoveryErrorCode: "source_unavailable",
      automaticDiscoveryStatus: "degraded",
      contentRevision: 3,
      directoryCount: 0,
      displayPath: "/library/family",
      id: "lib_family",
      lastSuccessfulScanAt: null,
      lastAutomaticDiscoveryAt: null,
      latestScanId: "scan_offline",
      name: "家庭影像",
      status: "offline",
    },
  });
  vi.mocked(listAssets).mockResolvedValue({ items: [], nextCursor: null });

  renderBrowse();

  expect(await screen.findByText("媒体库当前离线")).toBeVisible();
  expect(screen.getByText(/这不表示原目录为空/)).toBeVisible();
  expect(screen.queryByText("当前目录没有媒体")).not.toBeInTheDocument();
});

it("recovers a first-page media error through the shared retry action", async () => {
  const user = userEvent.setup();
  vi.mocked(listAssets)
    .mockRejectedValueOnce(new Error("network unavailable"))
    .mockResolvedValueOnce({ items: [], nextCursor: null });

  renderBrowse();

  expect(
    await screen.findByText("暂时无法读取媒体，请重新尝试。"),
  ).toBeVisible();
  await user.click(screen.getByRole("button", { name: "重新尝试" }));
  expect(await screen.findByText("当前目录没有媒体")).toBeVisible();
  expect(listAssets).toHaveBeenCalledTimes(2);
});

it("refreshes the current directory from the toolbar", async () => {
  const user = userEvent.setup();

  renderBrowse();

  expect(await screen.findByText("photo.jpg")).toBeVisible();
  const assetsCalls = vi.mocked(listAssets).mock.calls.length;
  const directoryCalls = vi.mocked(getDirectory).mock.calls.length;
  const libraryCalls = vi.mocked(getLibrary).mock.calls.length;

  await user.click(screen.getByRole("button", { name: "刷新当前目录" }));

  await waitFor(() => {
    expect(listAssets).toHaveBeenCalledTimes(assetsCalls + 1);
    expect(getDirectory).toHaveBeenCalledTimes(directoryCalls + 1);
    expect(getLibrary).toHaveBeenCalledTimes(libraryCalls + 1);
  });
});

it("refetches a cached directory when directory navigation returns to it", async () => {
  const user = userEvent.setup();
  vi.mocked(getDirectory).mockImplementation(async (directoryId) =>
    directoryId === "dir_japan"
      ? {
          breadcrumbs: [
            { id: "dir_root", name: "家庭影像", relativePath: "" },
            { id: "dir_travel", name: "旅行", relativePath: "旅行" },
            {
              id: "dir_japan",
              name: "日本",
              relativePath: "旅行/日本",
            },
          ],
          directAssetCount: 1,
          hasChildren: false,
          id: "dir_japan",
          libraryId: "lib_family",
          name: "日本",
          parentId: "dir_travel",
          recursiveAssetCount: 4,
          relativePath: "旅行/日本",
        }
      : {
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
        },
  );

  renderRouteBoundBrowse();

  await screen.findByRole("heading", { name: "旅行", level: 1 });
  const initialTravelCalls = vi
    .mocked(listAssets)
    .mock.calls.filter(([input]) => input.directoryId === "dir_travel").length;

  await user.click(screen.getByRole("link", { name: /^日本4 项$/ }));
  await screen.findByRole("heading", { name: "日本", level: 1 });
  await user.click(
    within(screen.getByRole("navigation", { name: "目录位置" })).getByRole(
      "link",
      { name: "旅行" },
    ),
  );

  await waitFor(() => {
    const travelCalls = vi
      .mocked(listAssets)
      .mock.calls.filter(
        ([input]) => input.directoryId === "dir_travel",
      ).length;
    expect(travelCalls).toBeGreaterThan(initialTravelCalls);
  });
});

it("refreshes an expired cursor before retrying the next media page", async () => {
  const user = userEvent.setup();
  let firstPageCalls = 0;
  vi.mocked(listAssets).mockImplementation(async ({ cursor }) => {
    if (!cursor) {
      firstPageCalls += 1;
      return {
        items: [
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
            storyboard: notApplicableStoryboard,
            thumbnail: { errorCode: null, status: "ready", url: "/photo.webp" },
            width: 1200,
          },
        ],
        nextCursor: firstPageCalls === 1 ? "expired-cursor" : "fresh-cursor",
      };
    }
    if (cursor === "expired-cursor") {
      throw new ApiError({
        code: "invalid_cursor",
        message: "The pagination cursor is invalid.",
        requestId: "req_cursor",
        status: 400,
      });
    }
    return {
      items: [
        {
          directoryId: "dir_travel",
          durationMs: null,
          height: 900,
          id: "ast_second",
          kind: "image",
          libraryId: "lib_family",
          libraryName: "家庭影像",
          mimeType: "image/jpeg",
          modifiedAt: "2026-07-26T00:00:00Z",
          name: "second.jpg",
          playbackStatus: "not_applicable",
          probeStatus: "ready",
          relativePath: "旅行/second.jpg",
          sizeBytes: 768,
          sourceAvailability: "available",
          storyboard: notApplicableStoryboard,
          thumbnail: { errorCode: null, status: "ready", url: "/second.webp" },
          width: 1200,
        },
      ],
      nextCursor: null,
    };
  });

  renderBrowse();

  expect(await screen.findByText("photo.jpg")).toBeVisible();
  expect(
    await screen.findByText("更多媒体未能载入，已显示的项目仍然保留。"),
  ).toBeVisible();
  await user.click(screen.getByRole("button", { name: "重试载入更多" }));

  expect(await screen.findByText("second.jpg")).toBeVisible();
  expect(listAssets).toHaveBeenCalledTimes(4);
  expect(vi.mocked(listAssets).mock.calls[3]?.[0]).toEqual(
    expect.objectContaining({ cursor: "fresh-cursor" }),
  );
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

function renderRootBrowse() {
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
          <MemoryRouter initialEntries={["/libraries/lib_family/browse"]}>
            <BrowsePage libraryId="lib_family" session={session} />
            <LocationProbe />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

function renderRouteBoundBrowse() {
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
            initialEntries={["/libraries/lib_family/browse/dir_travel"]}
          >
            <Routes>
              <Route
                element={<RouteBoundBrowse />}
                path="/libraries/:libraryId/browse/:directoryId?"
              />
            </Routes>
            <LocationProbe />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

function RouteBoundBrowse() {
  const { directoryId, libraryId = "" } = useParams<{
    directoryId?: string;
    libraryId: string;
  }>();
  return (
    <BrowsePage
      directoryId={directoryId}
      libraryId={libraryId}
      session={session}
    />
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
