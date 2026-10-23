import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, it, vi } from "vitest";

import { addManualAssetTag, createTag, getAssetCuration, listTags, replaceAssetTags } from "../../lib/api/curation";
import { curationKeys, useAddManualAssetTagMutation, useCreateAndAssignTagMutation, useLibraryTagCloudQuery, useReplaceAssetTagsMutation, useTagsQuery } from "./queries";

vi.mock("../../lib/api/curation", async importOriginal => ({
  ...await importOriginal<typeof import("../../lib/api/curation")>(), listTags: vi.fn(), replaceAssetTags: vi.fn(), addManualAssetTag: vi.fn(), createTag: vi.fn(), getAssetCuration: vi.fn(),
}));

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return { client, wrapper: ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider> };
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(listTags).mockReset();
  vi.mocked(listTags).mockResolvedValue({ items: [], nextCursor: null });
  vi.mocked(replaceAssetTags).mockResolvedValue({ assetId: "asset", favorite: false, favoritedAt: null, revision: 2, tags: [] });
});

it("keeps global selectors unfiltered and isolates library/search/cursor caches", async () => {
  const { wrapper, client } = setup();
  renderHook(() => { useTagsQuery(); useLibraryTagCloudQuery("lib_one", "sea", "page2"); useLibraryTagCloudQuery("lib_two"); }, { wrapper });
  await waitFor(() => expect(listTags).toHaveBeenCalledTimes(3));
  expect(listTags).toHaveBeenCalledWith(undefined);
  expect(listTags).toHaveBeenCalledWith("page2", { libraryId: "lib_one", kinds: ["image", "animated"], q: "sea" });
  expect(client.getQueryCache().find({ queryKey: curationKeys.tags, exact: true })).toBeDefined();
  expect(client.getQueryCache().find({ queryKey: curationKeys.tagCloud("lib_two"), exact: true })).toBeDefined();
});

it("does not make a global request when no cloud library is available", async () => {
  const { wrapper } = setup();
  renderHook(() => useLibraryTagCloudQuery(""), { wrapper });
  expect(listTags).not.toHaveBeenCalled();
});

it("refreshes current photo counts, global selectors and tag results after single-image tag edits", async () => {
  const { wrapper, client } = setup();
  const { result } = renderHook(() => {
    useLibraryTagCloudQuery("lib_one"); useTagsQuery(); return useReplaceAssetTagsMutation();
  }, { wrapper });
  client.setQueryData([...curationKeys.tagAssets("tag_removed"), { libraryId: "lib_one" }], { pages: [] });
  await waitFor(() => expect(listTags).toHaveBeenCalledTimes(2));
  await act(async () => { await result.current.mutateAsync({ assetId: "asset", csrfToken: "token", revision: 1, tagIds: [] }); });
  await waitFor(() => expect(listTags).toHaveBeenCalledTimes(4));
  expect(client.getQueryState([...curationKeys.tagAssets("tag_removed"), { libraryId: "lib_one" }])?.isInvalidated).toBe(true);
});

it("explicitly protects an existing AI tag when its name is manually entered", async () => {
  const { wrapper } = setup();
  const tag = { id: "tag_flower", name: "flower", assetCount: 1, createdAt: "now", updatedAt: "now" };
  const state = { assetId: "asset", favorite: false, favoritedAt: null, revision: 8, tags: [{ ...tag, source: "ai" as const }] };
  vi.mocked(createTag).mockResolvedValue(tag);
  vi.mocked(getAssetCuration).mockResolvedValue(state);
  vi.mocked(addManualAssetTag).mockResolvedValue({ ...state, revision: 9, tags: [{ ...tag, source: "manual" }] });
  const { result } = renderHook(() => useCreateAndAssignTagMutation(), { wrapper });
  await act(async () => { await result.current.mutateAsync({ assetId: "asset", name: "flower", csrfToken: "token" }); });
  expect(addManualAssetTag).toHaveBeenCalledWith({ assetId: "asset", tagId: "tag_flower", csrfToken: "token", revision: 8 });
});

it("refreshes a conflicted manual edit without replaying it over new AI results", async () => {
  const { wrapper, client } = setup();
  client.setQueryData(curationKeys.asset("asset"), { revision: 1 });
  vi.mocked(replaceAssetTags).mockRejectedValueOnce(new Error("precondition_failed"));
  const { result } = renderHook(() => useReplaceAssetTagsMutation(), { wrapper });
  await act(async () => { await expect(result.current.mutateAsync({ assetId: "asset", csrfToken: "token", revision: 1, tagIds: [] })).rejects.toThrow(); });
  expect(client.getQueryState(curationKeys.asset("asset"))?.isInvalidated).toBe(true);
});

it("refreshes the photo and globally created tag after a create-then-assign CAS conflict without retrying", async () => {
  const { wrapper, client } = setup();
  client.setQueryData(curationKeys.asset("asset"), { revision: 1 });
  client.setQueryData(curationKeys.tags, { pages: [] });
  vi.mocked(createTag).mockResolvedValue({ id: "tag_new", name: "new", assetCount: 0, createdAt: "now", updatedAt: "now" });
  vi.mocked(getAssetCuration).mockResolvedValue({ assetId: "asset", favorite: false, favoritedAt: null, revision: 8, tags: [] });
  vi.mocked(addManualAssetTag).mockRejectedValueOnce(new Error("precondition_failed"));
  const { result } = renderHook(() => useCreateAndAssignTagMutation(), { wrapper });
  await act(async () => { await expect(result.current.mutateAsync({ assetId: "asset", name: "new", csrfToken: "csrf" })).rejects.toThrow(); });
  expect(addManualAssetTag).toHaveBeenCalledOnce();
  expect(client.getQueryState(curationKeys.asset("asset"))?.isInvalidated).toBe(true);
  expect(client.getQueryState(curationKeys.tags)?.isInvalidated).toBe(true);
});

it("promotes one AI tag using CAS while preserving the rest of the photo tags and refreshing tag results", async () => {
  const { wrapper, client } = setup();
  const metadata = { assetCount: 1, createdAt: "now", updatedAt: "now" };
  const state = { assetId: "asset", favorite: false, favoritedAt: null, revision: 9, tags: [{ ...metadata, id: "tag_1", name: "tree", source: "manual" as const }, { ...metadata, id: "tag_2", name: "sky", source: "ai" as const }] };
  vi.mocked(addManualAssetTag).mockResolvedValue(state);
  client.setQueryData(curationKeys.tags, { pages: [] });
  client.setQueryData(curationKeys.tagAssets("tag_1"), { pages: [] });
  const { result } = renderHook(() => useAddManualAssetTagMutation(), { wrapper });
  const input = { assetId: "asset", csrfToken: "csrf", revision: 8, tagId: "tag_1" };
  await act(async () => { await result.current.mutateAsync(input); });
  expect(addManualAssetTag).toHaveBeenCalledWith(input, expect.anything());
  expect(replaceAssetTags).not.toHaveBeenCalled();
  expect(client.getQueryData(curationKeys.asset("asset"))).toEqual(state);
  expect(client.getQueryState(curationKeys.tags)?.isInvalidated).toBe(true);
  expect(client.getQueryState(curationKeys.tagAssets("tag_1"))?.isInvalidated).toBe(true);
});
