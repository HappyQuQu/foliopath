import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { getAutomaticTagSettings, requestAutomaticTagJob, updateAutomaticTagSettings, type AIOperationSnapshot, type AutomaticTagSnapshot } from "../../lib/api/ai";
import { ApiError } from "../../lib/api/errors";
import { curationKeys } from "../curation";
import { aiKeys } from "./queries";
import { useAutomaticTagSettings, useRequestAutomaticTags, useUpdateAutomaticTags } from "./automaticTagQueries";

vi.mock("../../lib/api/ai", async importOriginal => ({ ...await importOriginal<typeof import("../../lib/api/ai")>(),
  getAutomaticTagSettings: vi.fn(), requestAutomaticTagJob: vi.fn(), updateAutomaticTagSettings: vi.fn(),
}));

let client: QueryClient;
function snapshot(libraryId = "lib_1", revision = 2, coverageRevision = 3): AutomaticTagSnapshot {
  return { libraryId, revision, enabled: true, state: "ready", offline: false, model: { identity: "a".repeat(64), displayName: "JoyTag", protocol: "foliopath-tag-v1" },
    coverage: { eligible: 4, completed: 4, failed: 0, stale: 0, revision: coverageRevision }, activeOperationId: null, etag: `"autotag-library-${libraryId.slice(4)}-r${revision}"` };
}
function wrapper({ children }: { children: ReactNode }) { return <QueryClientProvider client={client}>{children}</QueryClientProvider>; }
beforeEach(() => {
  vi.clearAllMocks();
  client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } } });
  vi.mocked(getAutomaticTagSettings).mockImplementation(async id => snapshot(id));
});
afterEach(() => { cleanup(); client.clear(); vi.restoreAllMocks(); vi.useRealTimers(); });

it("does not fall back to a global or another library's request without a library", () => {
  renderHook(() => useAutomaticTagSettings(""), { wrapper });
  expect(getAutomaticTagSettings).not.toHaveBeenCalled();
});

it("invalidates curation after actual coverage changes, not initial load, unchanged polling or switching libraries", async () => {
  const invalidation = vi.spyOn(client, "invalidateQueries");
  const view = renderHook(({ id }) => useAutomaticTagSettings(id), { initialProps: { id: "lib_1" }, wrapper });
  await waitFor(() => expect(view.result.current.isSuccess).toBe(true));
  expect(invalidation).not.toHaveBeenCalled();
  await act(async () => { client.setQueryData(aiKeys.automaticTags("lib_1"), snapshot("lib_1", 2, 4)); });
  await waitFor(() => expect(invalidation).toHaveBeenCalledExactlyOnceWith({ queryKey: curationKeys.all }));
  await act(async () => { client.setQueryData(aiKeys.automaticTags("lib_1"), { ...snapshot("lib_1", 2, 4), state: "degraded" }); });
  view.rerender({ id: "lib_2" });
  await waitFor(() => expect(view.result.current.data?.libraryId).toBe("lib_2"));
  expect(invalidation).toHaveBeenCalledOnce();
});

it("uses bounded mounted-page polling and stops its timer when unmounted", async () => {
  vi.useFakeTimers();
  vi.mocked(getAutomaticTagSettings).mockResolvedValue({ ...snapshot(), state: "building", activeOperationId: "aio_running" });
  const view = renderHook(() => useAutomaticTagSettings("lib_1"), { wrapper });
  await act(() => vi.advanceTimersByTimeAsync(0));
  expect(getAutomaticTagSettings).toHaveBeenCalledTimes(1);
  await act(() => vi.advanceTimersByTimeAsync(1_999));
  expect(getAutomaticTagSettings).toHaveBeenCalledTimes(1);
  await act(() => vi.advanceTimersByTimeAsync(1));
  expect(getAutomaticTagSettings).toHaveBeenCalledTimes(2);
  view.unmount();
  await act(() => vi.advanceTimersByTimeAsync(20_000));
  expect(getAutomaticTagSettings).toHaveBeenCalledTimes(2);
});

it("preserves the exact CAS token, refreshes only the conflicted library and never replays a stale write", async () => {
  const before = snapshot(); client.setQueryData(aiKeys.automaticTags("lib_1"), before); client.setQueryData(aiKeys.automaticTags("lib_2"), snapshot("lib_2"));
  vi.mocked(updateAutomaticTagSettings).mockRejectedValueOnce(new ApiError({ code: "precondition_failed", status: 412, requestId: undefined, message: "Changed" }));
  const { result } = renderHook(() => useUpdateAutomaticTags(), { wrapper });
  const input = { libraryId: "lib_1", enabled: false, csrfToken: "csrf", etag: before.etag };
  await act(async () => { await expect(result.current.mutateAsync(input)).rejects.toMatchObject({ code: "precondition_failed" }); });
  expect(updateAutomaticTagSettings).toHaveBeenCalledExactlyOnceWith(input, expect.anything());
  expect(client.getQueryData(aiKeys.automaticTags("lib_1"))).toEqual(before);
  expect(client.getQueryState(aiKeys.automaticTags("lib_1"))?.isInvalidated).toBe(true);
  expect(client.getQueryState(aiKeys.automaticTags("lib_2"))?.isInvalidated).toBe(false);
});

it.each([snapshot("lib_1", 4, 4), snapshot("lib_1", 3, 5)])("does not overwrite a newer settings or progress snapshot with a late mutation response", async newer => {
  client.setQueryData(aiKeys.automaticTags("lib_1"), newer);
  vi.mocked(updateAutomaticTagSettings).mockResolvedValue({ ...snapshot("lib_1", 3, 4), enabled: false });
  const { result } = renderHook(() => useUpdateAutomaticTags(), { wrapper });
  await act(async () => { await result.current.mutateAsync({ libraryId: "lib_1", enabled: false, csrfToken: "csrf", etag: snapshot().etag }); });
  expect(client.getQueryData(aiKeys.automaticTags("lib_1"))).toEqual(newer);
});

it("seeds the canonical operation cache and refreshes admission state, including ambiguous failures", async () => {
  const operation: AIOperationSnapshot = { id: "aio_tags", kind: "automatic_tags_rebuild", state: "queued", phase: "queued", revision: 1, completedItems: 0, totalItems: null, errorCode: null, createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z", etag: '"aio_tags-r1"' };
  vi.mocked(requestAutomaticTagJob).mockResolvedValueOnce(operation).mockRejectedValueOnce(new Error("connection lost"));
  client.setQueryData(aiKeys.automaticTags("lib_1"), snapshot());
  const { result } = renderHook(() => useRequestAutomaticTags(), { wrapper });
  const input = { libraryId: "lib_1", mode: "all" as const, csrfToken: "csrf", idempotencyKey: "stable-request-key" };
  await act(async () => { await result.current.mutateAsync(input); });
  expect(client.getQueryData(aiKeys.operation(operation.id))).toEqual(operation);
  expect(client.getQueryState(aiKeys.automaticTags("lib_1"))?.isInvalidated).toBe(true);
  client.setQueryData(aiKeys.automaticTags("lib_1"), snapshot());
  await act(async () => { await expect(result.current.mutateAsync(input)).rejects.toThrow("connection lost"); });
  expect(client.getQueryState(aiKeys.automaticTags("lib_1"))?.isInvalidated).toBe(true);
  expect(requestAutomaticTagJob).toHaveBeenCalledTimes(2);
});
