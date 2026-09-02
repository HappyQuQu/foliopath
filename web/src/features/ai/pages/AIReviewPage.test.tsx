import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AIReviewPage } from "./AIReviewPage";

const mocks = vi.hoisted(() => ({ review: vi.fn() }));
const suggestion = {
  asset: { id: "asset-1", libraryId: "lib-1", name: "海边.jpg", kind: "image" as const, thumbnail: { status: "ready" as const, url: "/thumb.jpg" } },
  confidence: 0.93, generationId: "gen-1", id: "suggestion-1", libraryId: "lib-1",
  reviewedAt: null, revision: 2, status: "pending" as const,
  tag: { name: "海滩", tagId: "tag-1" }, vocabularyRevision: 1,
};

vi.mock("../../libraries", () => ({ useLibrariesQuery: () => ({ data: { pages: [{ items: [{ id: "lib-1", name: "照片", status: "ready" }] }] }, isPending: false }) }));
vi.mock("../../../lib/api/curation", () => ({ getAssetCuration: vi.fn().mockResolvedValue({ assetId: "asset-1", favorite: false, favoritedAt: null, revision: 7, tags: [] }) }));
vi.mock("../queries", () => ({
  useAITagSuggestionsQuery: () => ({ data: { items: [suggestion], nextCursor: null, coverage: { complete: false, completed: 8, eligible: 10, failed: 1, degraded: 1, stale: 1, revision: 1 } }, isError: false, isPending: false, refetch: vi.fn() }),
  useReviewAITagSuggestionsMutation: () => ({ isPending: false, mutateAsync: mocks.review }),
  usePeopleQuery: () => ({ data: { items: [], nextCursor: null }, isError: false, isPending: false, refetch: vi.fn() }),
  usePersonQuery: () => ({ isPending: true }), usePersonAssetsQuery: () => ({ isPending: true }),
  useFaceClustersQuery: () => ({ isPending: true }), useFaceClusterQuery: () => ({ isPending: true }),
  useCreatePersonMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useRenamePersonMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useApplyFaceReviewMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

beforeEach(() => { mocks.review.mockReset(); mocks.review.mockResolvedValue({ items: [] }); window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}'); });

it("separates AI suggestions from manual tags and submits exact revisions", async () => {
  const user = userEvent.setup(); renderPage();
  expect(screen.getByText("AI 建议（尚非人工标签）")).toBeVisible();
  expect(screen.getByText("分析覆盖 8/10；失败 1，降级 1")).toBeVisible();
  await user.click(screen.getByRole("checkbox", { name: "选择 海边.jpg" }));
  await user.click(screen.getByRole("button", { name: "接受建议" }));
  expect(mocks.review).toHaveBeenCalledWith(expect.objectContaining({ action: "accept", items: [{ curationRevision: 7, suggestion }] }));
});

it("provides explicit privacy language in Chinese", () => {
  renderPage();
  expect(screen.getByText(/不会识别现实身份、不会联网查人，也不会训练或发布模型/)).toBeVisible();
});

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><MemoryRouter initialEntries={["/intelligence?libraryId=lib-1"]}><ThemeProvider><LocaleProvider><ToastProvider><AIReviewPage session={{ administrator: { id: "admin-1", displayName: "管理员", username: "admin" }, csrfToken: "csrf-token-that-is-long-enough", expiresAt: "2026-09-03T00:00:00Z" }} /></ToastProvider></LocaleProvider></ThemeProvider></MemoryRouter></QueryClientProvider>);
}
