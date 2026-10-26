import { renderHook } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { aiKeys, useAIModelsQuery, useAIOperationQueries, useFaceSettingsQueries, useSemanticSettingsQueries, useVideoSemanticStatusQuery, useApplyFaceReviewMutation, useUndoFaceReviewMutation } from "./queries";

const mocks = vi.hoisted(() => ({
  queries: vi.fn((options: unknown) => options),
  client: { invalidateQueries: vi.fn() },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: unknown) => options,
  useQueries: mocks.queries,
  useMutation: (options: unknown) => options,
  useQueryClient: () => mocks.client,
}));

beforeEach(() => {
  mocks.queries.mockReset().mockImplementation((options: unknown) => options);
  mocks.client.invalidateQueries.mockReset();
});

it("refreshes model availability without requiring a page reload", () => {
  expect(useAIModelsQuery()).toMatchObject({ queryKey: ["ai", "models"], refetchInterval: 10_000 });
});

it("refreshes idle, missing and active image status at bounded intervals", () => {
  const options = useSemanticSettingsQueries(["lib_1"]) as unknown as {
    queries: Array<{ queryKey: string[]; refetchInterval: (query: { state: { data?: { state: string } } }) => number }>;
  };
  expect(options.queries[0]!.queryKey).toEqual(["ai", "semantic", "lib_1"]);
  const interval = options.queries[0]!.refetchInterval;
  for (const state of ["ready", "disabled", "awaiting_model", "degraded"]) {
    expect(interval({ state: { data: { state } } })).toBe(10_000);
  }
  expect(interval({ state: {} })).toBe(10_000);
  expect(interval({ state: { data: { state: "building" } } })).toBe(1_500);
  expect(interval({ state: { data: { state: "clearing" } } })).toBe(1_500);
});

it("scopes video polling to the selected library", () => {
  expect(useVideoSemanticStatusQuery("lib_2")).toMatchObject({ queryKey: ["ai", "video-status", "lib_2"], refetchInterval: 10_000 });
});

it("keeps face progress fresh for each library while slowing down idle and unavailable states", () => {
  const options = useFaceSettingsQueries(["lib_1", "lib_2"]) as unknown as {
    queries: Array<{ queryKey: string[]; refetchInterval: (query: { state: { data?: { state: string } } }) => number }>;
  };
  expect(options.queries.map(({ queryKey }) => queryKey)).toEqual([["ai", "faces", "lib_1"], ["ai", "faces", "lib_2"]]);
  for (const { refetchInterval } of options.queries) {
    for (const state of ["building", "clearing"]) {
      expect(refetchInterval({ state: { data: { state } } })).toBe(1_500);
    }
    for (const state of ["ready", "disabled", "awaiting_model", "degraded"]) {
      expect(refetchInterval({ state: { data: { state } } })).toBe(10_000);
    }
    expect(refetchInterval({ state: {} })).toBe(10_000);
  }
});

it.each(["succeeded", "failed", "cancelled"])("refreshes suggestion results when an operation becomes %s, once per revision", (state) => {
  const active = { id: "aio_tags", revision: 2, state: "running" };
  mocks.queries.mockReturnValue([{ data: active }]);
  const { rerender } = renderHook(() => useAIOperationQueries([active.id]));
  expect(mocks.client.invalidateQueries).not.toHaveBeenCalled();

  mocks.queries.mockReturnValue([{ data: { ...active, revision: 3, state } }]);
  rerender();
  expect(mocks.client.invalidateQueries).toHaveBeenCalledWith({ queryKey: aiKeys.suggestionsRoot });
  expect(mocks.client.invalidateQueries.mock.calls.map(([options]) => options.queryKey)).toEqual([
    aiKeys.models, aiKeys.downloads, ["ai", "semantic"], ["ai", "video-status"], ["ai", "faces"], aiKeys.suggestionsRoot,
  ]);
  mocks.client.invalidateQueries.mockClear();
  rerender();
  expect(mocks.client.invalidateQueries).not.toHaveBeenCalled();

  mocks.queries.mockReturnValue([{ data: { ...active, revision: 4, state } }]);
  rerender();
  expect(mocks.client.invalidateQueries).toHaveBeenCalledWith({ queryKey: aiKeys.suggestionsRoot });
});

it("does not invalidate result queries when operation records are unavailable", () => {
  mocks.queries.mockReturnValue([{ data: undefined, isError: true }]);
  renderHook(() => useAIOperationQueries(["aio_unavailable"]));
  expect(mocks.client.invalidateQueries).not.toHaveBeenCalled();
});

it.each([useApplyFaceReviewMutation, useUndoFaceReviewMutation])("refreshes active and inactive review caches on success and uncertain failure", async (hook) => {
  const options = hook() as unknown as { onSuccess: () => Promise<unknown>; onError: () => Promise<unknown> };
  await options.onSuccess(); await options.onError();
  expect(mocks.client.invalidateQueries.mock.calls).toEqual([[{ queryKey: aiKeys.all }], [{ queryKey: aiKeys.all }]]);
});
