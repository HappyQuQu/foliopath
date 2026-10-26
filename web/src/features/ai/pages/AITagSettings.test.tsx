import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { ApiError } from "../../../lib/api/errors";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { AITagSettings, type AITagSettingsProps } from "./AITagSettings";

const mocks = vi.hoisted(() => ({ vocabulary: vi.fn(), tags: vi.fn(), suggestions: vi.fn(), save: vi.fn(), job: vi.fn(), refresh: vi.fn(), tagRefresh: vi.fn(), next: vi.fn(), remember: vi.fn(), coverageRefresh: vi.fn() }));
vi.mock("../tagQueries", () => ({
  useAITagVocabularyQuery: mocks.vocabulary,
  useReplaceAITagVocabularyMutation: () => ({ mutateAsync: mocks.save, isPending: false }),
  useRequestAITagSuggestionJobMutation: () => ({ mutateAsync: mocks.job, isPending: false }),
  useAITagJobCoverageRefresh: mocks.coverageRefresh,
}));
vi.mock("../queries", () => ({ useAITagSuggestionsQuery: mocks.suggestions }));
vi.mock("../../curation", () => ({ useTagsQuery: mocks.tags }));

const vocabulary = {
  revision: 1,
  entries: [{ tagId: "tag_1", name: "旅行" }],
  publishedAt: "2026-10-22T00:00:00Z",
  etag: '"ai-tag-vocabulary-r1"',
};
const library = {
  id: "lib_family", name: "家庭照片", status: "ready" as const, assetCount: 12, directoryCount: 3, displayPath: "家庭照片", contentRevision: 2,
  automaticDiscoveryErrorCode: null, automaticDiscoveryStatus: "active" as const, lastAutomaticDiscoveryAt: null, lastSuccessfulScanAt: null, latestScanId: null,
};
const props: AITagSettingsProps = {
  session: { administrator: { displayName: "管理员", id: "admin_test", username: "test" }, csrfToken: "csrf", expiresAt: "2099-01-01T00:00:00Z" },
  library, mode: "vocabulary", semanticEnabled: true, semanticGenerationReady: true, serviceReady: true, onOperation: mocks.remember,
};
const operation = { id: "aio_tags", kind: "tag_suggestion_missing", state: "queued", revision: 1 };

beforeEach(() => {
  Object.values(mocks).forEach((mock) => mock.mockReset());
  mocks.vocabulary.mockReturnValue({ data: vocabulary, isPending: false, isError: false, refetch: mocks.refresh });
  mocks.tags.mockReturnValue({ data: { pages: [{ items: [{ id: "tag_1", name: "旅行" }, { id: "tag_2", name: "风景" }], nextCursor: null }] }, isPending: false, isError: false, refetch: mocks.tagRefresh, fetchNextPage: mocks.next, hasNextPage: false, isFetchingNextPage: false });
  mocks.suggestions.mockReturnValue({ data: { items: [], nextCursor: null, coverage: { completed: 6, eligible: 12, failed: 1 } }, isPending: false, isError: false, refetch: vi.fn() });
  mocks.save.mockResolvedValue({ ...vocabulary, etag: '"ai-tag-vocabulary-r2"', revision: 2, entries: [...vocabulary.entries, { tagId: "tag_2", name: "风景" }] });
  mocks.job.mockResolvedValue(operation);
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
});

it("publishes the shared vocabulary explicitly without starting a generation job", async () => {
  renderSettings();
  expect(screen.getByText("所有媒体库共用")).toBeVisible();
  expect(screen.getByRole("button", { name: "保存候选标签" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "候选标签：风景" }));
  await userEvent.click(screen.getByRole("button", { name: "保存候选标签" }));
  expect(mocks.save).toHaveBeenCalledWith({ csrfToken: "csrf", etag: vocabulary.etag, tagIds: ["tag_1", "tag_2"] });
  expect(mocks.job).not.toHaveBeenCalled();
  expect(mocks.remember).not.toHaveBeenCalled();
  expect(screen.queryByText("有未保存的更改")).not.toBeInTheDocument();
});

it("preserves unsaved choices when a newer shared vocabulary arrives", async () => {
  const view = renderSettings();
  await userEvent.click(screen.getByRole("button", { name: "候选标签：风景" }));
  mocks.vocabulary.mockReturnValue({ data: { ...vocabulary, revision: 2, entries: [], etag: '"ai-tag-vocabulary-r2"' }, isPending: false, isError: false, refetch: mocks.refresh });
  view.rerender(wrapper(props));
  expect(screen.getByRole("button", { name: "候选标签：风景" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("button", { name: "保存候选标签" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "载入最新候选列表" }));
  expect(screen.getByRole("button", { name: "候选标签：风景" })).toHaveAttribute("aria-pressed", "false");
  expect(mocks.save).not.toHaveBeenCalled();
});

it("retains the draft and refreshes on a server revision conflict", async () => {
  mocks.save.mockRejectedValue(new ApiError({ code: "precondition_failed", status: 412, message: "changed", requestId: undefined }));
  renderSettings();
  await userEvent.click(screen.getByRole("button", { name: "候选标签：风景" }));
  await userEvent.click(screen.getByRole("button", { name: "保存候选标签" }));
  expect(mocks.refresh).toHaveBeenCalledOnce();
  expect(screen.getByRole("button", { name: "候选标签：风景" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByText("有未保存的更改")).toBeVisible();
  expect(screen.queryByText("候选标签已保存，适用于所有媒体库。")).not.toBeInTheDocument();
});

it("preserves unsaved choices when a background vocabulary refresh fails", async () => {
  const view = renderSettings();
  await userEvent.click(screen.getByRole("button", { name: "候选标签：风景" }));
  mocks.vocabulary.mockReturnValue({ data: vocabulary, isPending: false, isError: true, refetch: mocks.refresh });
  view.rerender(wrapper(props));
  expect(screen.getByText("无法读取候选标签，请重试。")).toBeVisible();
  expect(screen.getByRole("button", { name: "候选标签：风景" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("button", { name: "保存候选标签" })).toBeDisabled();
  mocks.vocabulary.mockReturnValue({ data: vocabulary, isPending: false, isError: false, refetch: mocks.refresh });
  view.rerender(wrapper(props));
  expect(screen.getByRole("button", { name: "保存候选标签" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "候选标签：风景" })).toHaveAttribute("aria-pressed", "true");
});

it("shows one bounded candidate page and keeps selections across pages", async () => {
  mocks.tags.mockReturnValue({ data: { pages: [{ items: [{ id: "tag_1", name: "旅行" }], nextCursor: "next" }, { items: [{ id: "tag_2", name: "风景" }], nextCursor: null }] }, isPending: false, isError: false, hasNextPage: false, isFetchingNextPage: false });
  renderSettings();
  expect(screen.queryByRole("button", { name: "候选标签：风景" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "下一页" }));
  expect(screen.queryByRole("button", { name: "候选标签：旅行" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "候选标签：风景" }));
  await userEvent.click(screen.getByRole("button", { name: "保存候选标签" }));
  expect(mocks.save).toHaveBeenCalledWith(expect.objectContaining({ tagIds: ["tag_1", "tag_2"] }));
});

it("requests missing suggestions for the selected library and remembers the returned operation", async () => {
  renderSettings({ mode: "jobs" });
  await userEvent.click(screen.getByRole("button", { name: "生成缺失建议" }));
  expect(mocks.job).toHaveBeenCalledWith(expect.objectContaining({ libraryId: library.id, csrfToken: "csrf", mode: "missing", idempotencyKey: expect.any(String) }));
  expect(mocks.remember).toHaveBeenCalledWith(operation);
  expect(mocks.coverageRefresh).toHaveBeenLastCalledWith(library.id, operation.id);
  expect(mocks.save).not.toHaveBeenCalled();
});

it("requires confirmation before a full regeneration and closes the dialog on library change", async () => {
  const view = renderSettings({ mode: "jobs" });
  await userEvent.click(screen.getByRole("button", { name: "重新生成全部建议" }));
  expect(mocks.job).not.toHaveBeenCalled();
  expect(within(screen.getByRole("dialog")).getByText(/家庭照片/)).toBeVisible();
  view.rerender(wrapper({ ...props, mode: "jobs", library: { ...library, id: "lib_other", name: "其他照片" } }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(mocks.coverageRefresh).toHaveBeenLastCalledWith("lib_other", null);
  await userEvent.click(screen.getByRole("button", { name: "重新生成全部建议" }));
  await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "确认重新生成" }));
  expect(mocks.job).toHaveBeenCalledWith(expect.objectContaining({ libraryId: "lib_other", mode: "all" }));
});

it.each([
  { semanticEnabled: false },
  { semanticGenerationReady: false },
  { serviceReady: false },
  { library: { ...library, status: "offline" as const } },
  { disabled: true },
])("does not launch jobs when a required condition is missing (%j)", (override) => {
  renderSettings({ mode: "jobs", ...override });
  expect(screen.getByRole("button", { name: "生成缺失建议" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "重新生成全部建议" })).toBeDisabled();
  expect(mocks.job).not.toHaveBeenCalled();
});

it("does not offer generation with an empty controlled vocabulary", () => {
  mocks.vocabulary.mockReturnValue({ data: { ...vocabulary, entries: [] }, isPending: false, isError: false });
  renderSettings({ mode: "jobs" });
  expect(screen.getByText("尚未选择候选标签。先保存候选列表，再生成建议。")).toBeVisible();
  expect(screen.getByRole("button", { name: "生成缺失建议" })).toBeDisabled();
});

it("explains why tags cannot run until this library has an image generation", () => {
  renderSettings({ mode: "jobs", semanticGenerationReady: false });
  expect(screen.getByText("请先为当前媒体库建立智能搜索索引，再生成标签建议。")).toBeVisible();
  expect(screen.getByRole("button", { name: "生成缺失建议" })).toBeDisabled();
});

function wrapper(value: AITagSettingsProps) {
  return <LocaleProvider><ToastProvider><MemoryRouter><AITagSettings {...value} /></MemoryRouter></ToastProvider></LocaleProvider>;
}

function renderSettings(override: Partial<AITagSettingsProps> = {}) {
  return render(wrapper({ ...props, ...override }));
}
