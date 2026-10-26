import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AIReviewPage } from "./AIReviewPage";
import { ApiError } from "../../../lib/api/errors";
import { getAssetCuration } from "../../../lib/api/curation";

const mocks = vi.hoisted(() => ({ review: vi.fn(), create: vi.fn(), clusters: vi.fn(), cluster: vi.fn(), people: vi.fn(), personAssets: vi.fn(), libraryStatus: "ready" }));
const suggestion = {
  asset: { id: "asset-1", libraryId: "lib-1", name: "海边.jpg", kind: "image" as const, thumbnail: { status: "ready" as const, url: "/thumb.jpg" } },
  confidence: 0.93, generationId: "gen-1", id: "suggestion-1", libraryId: "lib-1",
  reviewedAt: null, revision: 2, status: "pending" as const,
  tag: { name: "海滩", tagId: "tag-1" }, vocabularyRevision: 1,
};
const tagPage = vi.hoisted(() => ({ nextCursor: null as string | null, second: false, sameAsset: false, empty: false, complete: false, query: vi.fn() }));

vi.mock("../../libraries", () => ({ useLibrariesQuery: () => ({ data: { pages: [{ items: [{ id: "lib-1", name: "照片", status: mocks.libraryStatus }] }] }, isPending: false, refetch: vi.fn() }) }));
vi.mock("../../../lib/api/curation", () => ({ getAssetCuration: vi.fn().mockResolvedValue({ assetId: "asset-1", favorite: false, favoritedAt: null, revision: 7, tags: [] }) }));
vi.mock("../queries", () => ({
  useAITagSuggestionsQuery: (...args: unknown[]) => { tagPage.query(...args); return { data: { items: tagPage.empty ? [] : [suggestion, ...(tagPage.second ? [{ ...suggestion, id: "suggestion-2", asset: { ...suggestion.asset, id: "asset-2", name: "第二张.jpg" } }] : []), ...(tagPage.sameAsset ? [{ ...suggestion, id: "suggestion-blue", tag: { name: "蓝色", tagId: "tag-blue" } }] : [])], nextCursor: tagPage.nextCursor, coverage: { complete: tagPage.complete, completed: 8, eligible: 10, failed: 1, degraded: 1, stale: 1, revision: 1 } }, isError: false, isPending: false, refetch: vi.fn() }; },
  useReviewAITagSuggestionsMutation: () => ({ isPending: false, mutateAsync: mocks.review }),
  usePeopleQuery: mocks.people,
  usePersonQuery: () => ({ isPending: true }), usePersonAssetsQuery: mocks.personAssets,
  useFaceClustersQuery: mocks.clusters, useFaceClusterQuery: mocks.cluster,
  useCreatePersonMutation: () => ({ isPending: false, mutateAsync: mocks.create }),
  useRenamePersonMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useApplyFaceReviewMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

beforeEach(() => { mocks.people.mockReset().mockReturnValue({ data: { items: [{ id: "person-1", name: "青岚", assetCount: 4, confirmedFaceCount: 5 }], nextCursor: null }, isError: false, isPending: false, refetch: vi.fn() }); mocks.personAssets.mockReset().mockReturnValue({ isPending: true }); mocks.libraryStatus = "ready"; mocks.review.mockReset(); mocks.review.mockResolvedValue({ items: [] }); mocks.clusters.mockReturnValue({ isPending: true }); mocks.cluster.mockReturnValue({ isPending: true }); window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}'); });
beforeEach(() => { tagPage.nextCursor = null; tagPage.second = false; tagPage.sameAsset = false; tagPage.empty = false; tagPage.complete = false; tagPage.query.mockClear(); vi.mocked(getAssetCuration).mockReset().mockResolvedValue({ assetId: "asset-1", favorite: false, favoritedAt: null, revision: 7, tags: [] }); });

it("keeps the legacy review deep link but removes it from normal navigation", () => {
  renderPage();
  expect(screen.getByRole("heading", { name: "标签建议", level: 1 })).toBeVisible();
  expect(screen.queryByRole("link", { name: "标签建议" })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: /^标签$/ })).toHaveAttribute("href", "/tags?libraryId=lib-1");
  expect(screen.getByRole("link", { name: "分析设置" })).toHaveAttribute("href", "/settings/ai?library=lib-1");
  expect(screen.queryByRole("button", { name: "待整理" })).not.toBeInTheDocument();
  expect(screen.queryByText("关于本地处理与人物归类")).not.toBeInTheDocument();
});

it("groups several suggestions for the same photo and shows the page selection count", async () => {
  tagPage.sameAsset = true;
  const user = userEvent.setup(); renderPage();
  expect(screen.getAllByRole("img", { name: "海边.jpg" })).toHaveLength(1);
  expect(screen.getByText("本页 2 条建议 · 1 张照片")).toBeVisible();
  await user.click(screen.getByRole("checkbox", { name: "选择本页全部建议" }));
  expect(screen.getByText("已选 2 条")).toBeVisible();
  expect(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" })).toBeChecked();
  expect(screen.getByRole("checkbox", { name: "选择 海边.jpg · 蓝色" })).toBeChecked();
});

it("recovers failed photo tag state without discarding the review selection", async () => {
  vi.mocked(getAssetCuration).mockRejectedValueOnce(new Error("network unavailable"));
  const user = userEvent.setup(); renderPage();
  expect(await screen.findByText(/部分照片的标签状态未能载入/)).toBeVisible();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  expect(screen.getByRole("button", { name: "采纳所选" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "忽略所选" })).toBeDisabled();
  expect(mocks.review).not.toHaveBeenCalled();
  await user.click(screen.getByRole("button", { name: "重新尝试" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "采纳所选" })).toBeEnabled());
  expect(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" })).toBeChecked();
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ items: [{ curationRevision: 7, suggestion }] }));
});

it("accepts multiple tags on one photo using refreshed revisions after its own first acceptance", async () => {
  tagPage.sameAsset = true;
  mocks.review.mockImplementation(async (request) => {
    const item = request.items[0];
    vi.mocked(getAssetCuration).mockResolvedValue({ assetId: "asset-1", favorite: false, favoritedAt: null, revision: 8, tags: [] });
    return { items: [{ suggestionId: item.suggestion.id, outcome: "accepted", revision: 3 }] };
  });
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择本页全部建议" }));
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  await waitFor(() => expect(mocks.review).toHaveBeenCalledTimes(2));
  expect(mocks.review.mock.calls[0]?.[0].items).toEqual([{ suggestion, curationRevision: 7 }]);
  expect(mocks.review.mock.calls[1]?.[0].items).toEqual([expect.objectContaining({ suggestion: expect.objectContaining({ id: "suggestion-blue" }), curationRevision: 8 })]);
  expect(screen.getByText("已选 0 条")).toBeVisible();
});

it("does not rebase subsequent tags on a photo whose first tag conflicts", async () => {
  tagPage.sameAsset = true;
  mocks.review.mockResolvedValue({ items: [{ suggestionId: suggestion.id, outcome: "conflict", revision: 2 }] });
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择本页全部建议" }));
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  expect(mocks.review).toHaveBeenCalledTimes(1);
  expect(screen.getByText("已选 2 条")).toBeVisible();
  expect(screen.getByText(/已保存 0 项，2 项未确认保存/)).toBeVisible();
});

it.each([["pending", "暂无待处理的标签建议"], ["accepted", "还没有已采纳的建议"], ["dismissed", "还没有已忽略的建议"]])("uses a truthful empty state for %s suggestions", (status, title) => {
  tagPage.empty = true; tagPage.complete = true;
  renderPage(`/intelligence?view=tags&libraryId=lib-1&status=${status}`);
  expect(screen.getByText(title)).toBeVisible();
  expect(screen.queryByText(/建议已处理完/)).not.toBeInTheDocument();
});

it("explains an incomplete empty list once without irrelevant selection instructions", () => {
  tagPage.empty = true;
  renderPage();
  expect(screen.getAllByText(/标签分析尚未全部完成/)).toHaveLength(1);
  expect(screen.queryByText(/勾选适合照片的标签后批量采纳/)).not.toBeInTheDocument();
});

it("uses real library availability instead of interpreting offline as no faces", () => {
  mocks.libraryStatus = "offline";
  renderPage("/intelligence?view=clusters&libraryId=lib-1");
  expect(screen.getByText(/媒体库暂不可用，已有内容保留/)).toBeVisible();
  expect(screen.queryByText(/人脸分析尚未就绪/)).not.toBeInTheDocument();
});

it("marks the actual face region in group photos before individual selection", () => {
  mocks.cluster.mockReturnValue({ isPending: false, isError: false, data: {
    cluster: { id: "cluster-1", kind: "core", memberCount: 1 }, nextCursor: null,
    items: [{ faceId: "face-1", assetId: "asset-1", kind: "core", revision: 1, region: { xPercent: 10, yPercent: 20, widthPercent: 30, heightPercent: 40 } }],
  } });
  renderPage("/intelligence?view=clusters&libraryId=lib-1&cluster=cluster-1");
  const region = screen.getByRole("checkbox").closest("label")?.querySelector('span[aria-hidden="true"]');
  expect(region).toHaveStyle({ left: "10%", top: "20%", width: "30%", height: "40%" });
  expect(screen.getByRole("button", { name: "新建人物" })).toBeDisabled();
});

it.each([
  ["face_not_ready", /人脸分析尚未就绪/],
  ["library_offline", /媒体库暂不可用，已有内容保留/],
  ["face_cursor_stale", /状态已被其他操作更新/],
  ["network_error", /无法载入审核数据/],
] as const)("distinguishes %s from empty face results", (code, message) => {
  mocks.clusters.mockReturnValue({ isPending: false, isError: true, error: new ApiError({ code, message: "test", requestId: undefined, status: 409 }), refetch: vi.fn() });
  renderPage("/intelligence?view=clusters&libraryId=lib-1");
  expect(screen.getByText(message)).toBeVisible();
});

it("separates AI suggestions from manual tags and submits exact revisions", async () => {
  const user = userEvent.setup(); renderPage();
  expect(screen.getByText(/采纳后才会加入你的标签/)).toBeVisible();
  expect(screen.getByText("分析覆盖 8/10；失败 1，降级 1")).toBeVisible();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ action: "accept", items: [{ curationRevision: 7, suggestion }] }));
});

it("keeps local processing details available on demand", async () => {
  const user = userEvent.setup();
  renderPage("/intelligence?view=clusters&libraryId=lib-1");
  await user.click(screen.getByText("关于本地处理与人物归类"));
  expect(screen.getByText(/人物分析在本地进行/)).toBeVisible();
});

it.each([["采纳所选", "accepted"], ["忽略所选", "dismissed"]])("restores focus after %s removes the review controls", async (label, outcome) => {
  mocks.review.mockResolvedValue({ items: [{ suggestionId: suggestion.id, outcome, revision: 3 }] });
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  await user.click(screen.getByRole("button", { name: label }));
  expect(screen.queryByRole("button", { name: label })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "待处理" })).toHaveFocus();
});

it("does not steal focus moved elsewhere while a tag review is pending", async () => {
  let finish!: (value: unknown) => void;
  mocks.review.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  const search = screen.getByRole("searchbox");
  await user.click(search);
  await act(async () => { finish({ items: [{ suggestionId: suggestion.id, outcome: "accepted", revision: 3 }] }); });
  expect(search).toHaveFocus();
});

it("does not report saved when the tag review returns an HTTP-success item conflict", async () => {
  mocks.review.mockResolvedValue({ items: [{ suggestionId: suggestion.id, outcome: "conflict", revision: suggestion.revision }] });
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  expect(mocks.review).toHaveBeenCalled();
  expect(screen.queryByText("审核已保存。")).not.toBeInTheDocument();
  expect(screen.getByText(/已保存 0 项，1 项未确认保存/)).toBeVisible();
  expect(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" })).toBeChecked();
});

it("retains only conflicted selections in a mixed response", async () => {
  tagPage.second = true;
  mocks.review.mockResolvedValue({ items: [{ suggestionId: suggestion.id, outcome: "accepted", revision: 3 }, { suggestionId: "suggestion-2", outcome: "conflict", revision: 2 }] });
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  await user.click(screen.getByRole("checkbox", { name: "选择 第二张.jpg · 海滩" }));
  await user.click(screen.getByRole("button", { name: "采纳所选" }));
  expect(screen.getByText(/已保存 1 项，1 项未确认保存/)).toBeVisible();
  expect(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" })).not.toBeChecked();
  expect(screen.getByRole("checkbox", { name: "选择 第二张.jpg · 海滩" })).toBeChecked();
});

it("uses the URL cursor and clears it when status changes", async () => {
  tagPage.nextCursor = "opaque-next-page";
  const user = userEvent.setup(); renderPage();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" }));
  await user.click(screen.getByRole("button", { name: "下一页" }));
  expect(tagPage.query).toHaveBeenLastCalledWith("lib-1", "pending", "opaque-next-page");
  expect(screen.getByRole("checkbox", { name: "选择 海边.jpg · 海滩" })).not.toBeChecked();
  await user.click(screen.getByRole("button", { name: "已采纳" }));
  expect(tagPage.query).toHaveBeenLastCalledWith("lib-1", "accepted", "");
});

it("makes the next tag suggestion page reachable when the backend returns a cursor", () => {
  tagPage.nextCursor = "opaque-next-page";
  renderPage();
  expect(screen.getByRole("button", { name: "下一页" })).toBeEnabled();
});

it("allows creating the first person without enabling automatic group assignment", async () => {
  mocks.create.mockResolvedValue({ id: "person-created" });
  const user = userEvent.setup(); renderPage("/intelligence?view=people");
  expect(screen.getByRole("button", { name: "新建人物" })).toBeDisabled();
  await user.type(screen.getByLabelText("管理员填写的名称"), "人工人物");
  await user.click(screen.getByRole("button", { name: "新建人物" }));
  expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ name: "人工人物" }));
  expect(mocks.create.mock.calls[0]?.[0]).not.toHaveProperty("sourceCluster");
  await waitFor(() => expect(screen.getByRole("heading", { name: "人物", level: 1 })).toHaveFocus());
});

it("opens the people workspace by default and shows real preview thumbnails", () => {
  mocks.clusters.mockReturnValue({ isPending: false, isError: false, data: {
    coverage: { complete: true, completed: 10, eligible: 10, failed: 0, degraded: 0, stale: 0, revision: 1 },
    groupAssignmentAllowed: false, nextCursor: null,
    items: [{ id: "cluster-preview", kind: "core", memberCount: 4, previewAssetIds: ["asset-one", "asset-two"], revision: 1 }],
  } });
  renderPage("/intelligence?libraryId=lib-1");
  expect(screen.getByRole("link", { name: "人物" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  expect(screen.getByRole("button", { name: "待整理" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("button", { name: "待确认人物，4 张人脸" })).toBeVisible();
  expect(screen.getByRole("button", { name: "待确认人物，4 张人脸" }).querySelectorAll("img")).toHaveLength(2);
});

function renderPage(entry = "/intelligence?view=tags&libraryId=lib-1") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><MemoryRouter initialEntries={[entry]}><ThemeProvider><LocaleProvider><ToastProvider><AIReviewPage session={{ administrator: { id: "admin-1", displayName: "管理员", username: "admin" }, csrfToken: "csrf-token-that-is-long-enough", expiresAt: "2026-09-03T00:00:00Z" }} /></ToastProvider></LocaleProvider></ThemeProvider></MemoryRouter></QueryClientProvider>);
}

it("keeps named people global and separates sidebar paging from person asset paging", async () => {
  const user = userEvent.setup();
  renderPage("/intelligence?view=people&libraryId=lib-1&person=person-1&cursor=asset-page&peopleCursor=people-page");
  const sidebar = screen.getByRole("complementary", { name: "人物工具" });
  expect(within(sidebar).getByText("全部媒体库")).toBeVisible();
  expect(within(sidebar).queryByRole("button", { name: /^媒体库：/ })).not.toBeInTheDocument();
  expect(mocks.people).toHaveBeenCalledWith("", "people-page");
  expect(mocks.personAssets).toHaveBeenCalledWith("person-1", "asset-page");
  await user.type(within(sidebar).getByRole("searchbox", { name: "按管理员填写的名称搜索" }), "白露");
  expect(mocks.people).toHaveBeenCalledWith("白露", "");
  expect(mocks.personAssets).toHaveBeenLastCalledWith("person-1", "asset-page");
});

it("shows library selection for unnamed groups and keeps their subtype controls in the sidebar", () => {
  renderPage("/intelligence?view=clusters&libraryId=lib-1");
  const sidebar = screen.getByRole("complementary", { name: "人物工具" });
  expect(within(sidebar).getByRole("button", { name: /媒体库：照片/ })).toBeVisible();
  expect(within(sidebar).getByRole("button", { name: "待确认人物" })).toBeVisible();
  expect(within(screen.getByRole("main")).queryByRole("combobox", { name: "媒体库" })).not.toBeInTheDocument();
});
