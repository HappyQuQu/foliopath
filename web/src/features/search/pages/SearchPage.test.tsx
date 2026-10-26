import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import {
  searchAssets,
  searchFusedAssets,
  searchLibraryAssets,
  searchSemanticAssets,
  searchSemanticVideos,
  type Asset,
} from "../../../lib/api/catalog";
import { getLibrary, listLibraries } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { SearchPage } from "./SearchPage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/catalog")>();
  return {
    ...actual,
    searchAssets: vi.fn(),
    searchFusedAssets: vi.fn(),
    searchLibraryAssets: vi.fn(),
    searchSemanticAssets: vi.fn(),
    searchSemanticVideos: vi.fn(),
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
  thumbnail: { errorCode: null, status: "pending", url: null },
  width: 1200,
};

const videoResult: Asset = {
  ...result,
  durationMs: 42_000,
  height: 1080,
  id: "ast_video",
  kind: "video",
  mimeType: "video/mp4",
  name: "京都散步.mp4",
  playbackStatus: "playable",
  relativePath: "旅行/日本/京都散步.mp4",
  width: 1920,
};

beforeEach(() => {
  vi.mocked(listLibraries).mockResolvedValue({ items: [], nextCursor: null });
  vi.mocked(getLibrary).mockReset();
  vi.mocked(searchAssets).mockReset();
  vi.mocked(searchFusedAssets).mockReset();
  vi.mocked(searchFusedAssets).mockResolvedValue({ items: [result], nextCursor: null, searchSemanticState: "ready", searchMatches: { [result.id]: {sources: ["filename", "semantic"], semanticScore: .32} } });
  vi.mocked(searchLibraryAssets).mockReset();
  vi.mocked(searchSemanticAssets).mockReset();
  vi.mocked(searchSemanticVideos).mockReset();
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
  vi.mocked(searchLibraryAssets).mockResolvedValue({
    items: [result],
    nextCursor: null,
  });
  vi.mocked(searchAssets).mockResolvedValue({
    items: [result],
    nextCursor: null,
  });
  vi.mocked(searchSemanticAssets).mockResolvedValue({
    items: [result],
    nextCursor: null,
    semanticCoverage: {
      complete: false,
      completed: 8,
      eligible: 12,
      excludedLibraries: [],
      failed: 1,
      stale: 3,
    },
  });
  vi.mocked(searchSemanticVideos).mockResolvedValue({
    items: [videoResult],
    nextCursor: null,
    semanticCoverage: {
      complete: true,
      completed: 1,
      eligible: 1,
      excludedLibraries: [],
      failed: 0,
      stale: 0,
    },
    semanticVideoMatches: {
      ast_video: { ordinal: 3, planSize: 10, timestampMs: 42_000 },
    },
  });
});

it("restores a library search and exposes the source library and directory", async () => {
  renderSearch("/libraries/lib_family/search?q=京都&mode=filename");

  expect(
    await screen.findByRole("heading", { name: "“京都”的搜索结果" }),
  ).toBeVisible();
  expect(screen.getByRole("searchbox", { name: "搜索文件名或路径" })).toHaveValue(
    "京都",
  );
  expect(await screen.findByText("京都夜景.jpg")).toBeVisible();
  expect(
    screen.getByRole("link", { name: "家庭影像 · 旅行/日本" }),
  ).toHaveAttribute("href", "/libraries/lib_family/browse/dir_japan");
  expect(screen.queryByRole("link", { name: "返回浏览" })).not.toBeInTheDocument();
  await userEvent.setup().click(
    screen.getByRole("button", { name: "家庭管理员的账户菜单" }),
  );
  expect(screen.getByRole("menuitem", { name: "管理中心" })).toHaveAttribute(
    "href",
    "/settings/general?libraryId=lib_family",
  );
  expect(searchLibraryAssets).toHaveBeenCalledWith(
    expect.objectContaining({
      libraryId: "lib_family",
      order: "desc",
      q: "京都",
      sort: "modifiedAt",
    }),
  );
});

it("refreshes the current search from an explicit button", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?q=京都&mode=filename");

  expect(await screen.findByText("京都夜景.jpg")).toBeVisible();
  const calls = vi.mocked(searchLibraryAssets).mock.calls.length;

  await user.click(
    screen.getByRole("button", { name: "刷新搜索结果" }),
  );

  await waitFor(() =>
    expect(searchLibraryAssets).toHaveBeenCalledTimes(calls + 1),
  );
});

it("moves scope and filters into the URL and selects the global endpoint", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?q=京都&mode=filename");

  await screen.findByText("京都夜景.jpg");
  await user.click(screen.getByRole("radio", { name: "全部媒体库" }));
  await user.selectOptions(screen.getByLabelText("媒体类型"), "video");

  await waitFor(() =>
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/libraries/lib_family/search?q=%E4%BA%AC%E9%83%BD&mode=filename&scope=all&kind=video",
    ),
  );
  expect(searchAssets).toHaveBeenCalledWith(
    expect.objectContaining({
      kinds: ["video"],
      q: "京都",
    }),
  );
});

it("switches to semantic mode through the canonical URL and fixed relevance query", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?q=夜景&kind=animated&date=30d&mode=filename");

  await screen.findByText("京都夜景.jpg");
  await user.click(screen.getByRole("radio", { name: "画面内容" }));

  await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_family/search?q=%E5%A4%9C%E6%99%AF&mode=semantic",
  ));
  expect(searchSemanticAssets).toHaveBeenCalledWith({
    libraryId: "lib_family",
    q: "夜景",
  });
  expect(screen.getByLabelText("媒体类型")).toHaveValue("image");
  expect(await screen.findByText("部分照片尚未处理，搜索结果可能不完整。")).toBeVisible();
  expect(screen.queryByLabelText("日期")).not.toBeInTheDocument();
  expect(screen.queryByLabelText("排序")).not.toBeInTheDocument();
  expect(screen.getByRole("searchbox", { name: "描述画面内容" })).toHaveValue("夜景");
  expect(screen.getByText("画面语义检索")).toBeVisible();
  expect(screen.queryByText("文件名或路径检索")).not.toBeInTheDocument();
});

it("explains empty semantic search without requesting filename results or inventing OCR", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?mode=semantic&scope=directory&directoryId=dir_japan&recursive=1");
  const input = screen.getByRole("searchbox", { name: "描述画面内容" });
  expect(input).toHaveAttribute("placeholder", "描述画面，例如：汽车、海边日落、蓝色头发");
  expect(screen.getByText("描述你想找的画面，例如海边日落。")).toBeVisible();
  expect(searchSemanticAssets).not.toHaveBeenCalled();
  expect(searchLibraryAssets).not.toHaveBeenCalled();
  await user.type(input, "车");
  await user.keyboard("{Enter}");
  expect(await screen.findByText("画面语义检索")).toBeVisible();
  expect(searchSemanticAssets).toHaveBeenCalledWith({ libraryId: "lib_family", directoryId: "dir_japan", recursive: true, q: "车" });
  expect(searchLibraryAssets).not.toHaveBeenCalled();
  expect(screen.queryByText(/按画面相似度排序，不代表/)).not.toBeInTheDocument();
  await user.click(screen.getByRole("radio", { name: "文件名" }));
  expect(await screen.findByText("文件名或路径检索")).toBeVisible();
  expect(screen.getByRole("searchbox", { name: "搜索文件名或路径" })).toHaveValue("车");
  expect(screen.queryByText("画面语义检索")).not.toBeInTheDocument();
  expect(searchLibraryAssets).toHaveBeenCalledWith(expect.objectContaining({ libraryId: "lib_family", directoryId: "dir_japan", recursive: true, q: "车" }));
});

it("shows the best matching storyboard frame for semantic video results", async () => {
  renderSearch("/libraries/lib_family/search?q=散步&mode=semantic&kind=video");

  expect(await screen.findByText("京都散步.mp4")).toBeVisible();
  expect(screen.getByText("画面语义检索 · 视频 0:42")).toBeVisible();
  expect(searchSemanticVideos).toHaveBeenCalledWith({
    libraryId: "lib_family",
    q: "散步",
  });
});

it("preserves an empty query state and clears only active filters", async () => {
  const user = userEvent.setup();
  vi.mocked(searchLibraryAssets).mockResolvedValue({
    items: [],
    nextCursor: null,
  });
  renderSearch("/libraries/lib_family/search?q=不存在&kind=video&mode=filename");

  expect(await screen.findByText("没有搜索结果")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "清除筛选" }));

  await waitFor(() =>
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/libraries/lib_family/search?q=%E4%B8%8D%E5%AD%98%E5%9C%A8",
    ),
  );
  expect(screen.getByRole("searchbox")).toHaveValue("不存在");
});

it("reuses the pinned non-modal preview state machine", async () => {
  const user = userEvent.setup();
  vi.mocked(searchLibraryAssets).mockResolvedValue({
    items: [result, videoResult],
    nextCursor: null,
  });
  renderSearch("/libraries/lib_family/search?q=京都&mode=filename");

  const firstTrigger = await screen.findByRole("button", {
    name: "预览 京都夜景.jpg",
  });
  await user.click(firstTrigger);
  const imagePreview = screen.getByRole("complementary", {
    name: "预览: 京都夜景.jpg",
  });
  await user.click(
    within(imagePreview).getByRole("button", { name: "固定预览" }),
  );

  const videoTrigger = screen.getByRole("button", {
    name: "选择 京都散步.mp4，双击切换固定预览",
  });
  await user.click(videoTrigger);
  expect(imagePreview).toBeVisible();
  await user.dblClick(videoTrigger);
  expect(
    screen.getByRole("complementary", { name: "预览: 京都散步.mp4" }),
  ).toBeVisible();

  await user.keyboard("{Escape}");
  await waitFor(() => expect(videoTrigger).toHaveFocus());
});

it("keeps a pinned preview while filters replace the result set", async () => {
  const user = userEvent.setup();
  vi.mocked(searchLibraryAssets).mockImplementation(async (input) => ({
    items: input.kinds?.includes("video") ? [] : [result],
    nextCursor: null,
  }));
  renderSearch("/libraries/lib_family/search?q=京都&mode=filename");

  await user.click(
    await screen.findByRole("button", { name: "预览 京都夜景.jpg" }),
  );
  const preview = screen.getByRole("complementary", {
    name: "预览: 京都夜景.jpg",
  });
  await user.click(within(preview).getByRole("button", { name: "固定预览" }));
  await user.selectOptions(screen.getByLabelText("媒体类型"), "video");

  expect(await screen.findByText("没有搜索结果")).toBeVisible();
  expect(preview).toBeVisible();
  expect(preview).toHaveTextContent("固定预览不在当前结果中");
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

it("owns search filters only in the contextual sidebar and retains URL query context", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?q=京都&scope=directory&directoryId=dir_japan&recursive=1&mode=filename");
  const sidebar = screen.getByRole("complementary", { name: "搜索工具" });
  expect(within(sidebar).getByRole("radio", { name: "当前目录" })).toBeChecked();
  expect(within(screen.getByRole("main")).queryByRole("combobox")).not.toBeInTheDocument();
  expect(screen.queryByRole("navigation", { name: "目录导航" })).not.toBeInTheDocument();
  await user.selectOptions(within(sidebar).getByRole("combobox", { name: "媒体类型" }), "video");
  await waitFor(() => expect(searchLibraryAssets).toHaveBeenLastCalledWith(expect.objectContaining({ directoryId: "dir_japan", q: "京都", recursive: true, kinds: ["video"] })));
});


it("defaults to the fused endpoint and shows actual sources with preview evidence", async () => {
  const user = userEvent.setup();
  renderSearch("/libraries/lib_family/search?q=京都");
  expect(await screen.findByText("京都夜景.jpg")).toBeVisible();
  expect(searchFusedAssets).toHaveBeenCalledWith(expect.objectContaining({libraryId: "lib_family", q: "京都"}));
  expect(searchLibraryAssets).not.toHaveBeenCalled();
  expect(searchSemanticAssets).not.toHaveBeenCalled();
  const tools = screen.getByRole("complementary", {name: "搜索工具"});
  expect(within(tools).getByRole("radio", {name: "全部来源"})).toBeChecked();
  expect(within(tools).getByRole("radio", {name: "图片文字"})).toBeEnabled();
  expect(screen.queryByLabelText("排序")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", {name: "预览 京都夜景.jpg"}));
  const preview = screen.getByRole("complementary", {name: "预览: 京都夜景.jpg"});
  expect(within(preview).getByRole("heading", {name: "为什么匹配"})).toBeVisible();
  expect(within(preview).getByText("文件名")).toBeVisible();
  expect(within(preview).getByText("画面内容")).toBeVisible();
  expect(within(preview).queryByText("库内路径")).not.toBeInTheDocument();
});

it("keeps literal results when semantic service is unavailable without inventing visual matches", async () => {
  vi.mocked(searchFusedAssets).mockResolvedValue({items: [result], nextCursor: null, searchSemanticState: "unavailable", searchMatches: {[result.id]: {sources: ["path"], semanticScore: null}}});
  renderSearch("/libraries/lib_family/search?q=日本");
  expect(await screen.findByText("京都夜景.jpg")).toBeVisible();
  expect(screen.getByText("画面搜索暂不可用，仍可按文件名和图片文字查找。")).toBeVisible();
  expect(screen.getByText("库内路径", {selector: "span"})).toBeVisible();
  expect(screen.queryByText("画面内容", {selector: "span[data-tone]"})).not.toBeInTheDocument();
});
