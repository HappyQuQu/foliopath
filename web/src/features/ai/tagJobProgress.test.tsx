import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import type { AIOperationSnapshot } from "../../lib/api/ai";
import { aiKeys } from "./queries";
import { useAITagJobCoverageRefresh } from "./tagQueries";

let client: QueryClient;

beforeEach(() => {
  vi.useFakeTimers();
  client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
});

afterEach(() => {
  cleanup();
  client.clear();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function operation(state: AIOperationSnapshot["state"]): AIOperationSnapshot {
  return { id: "aio_tags", kind: "tag_suggestion_missing", state, phase: "building", revision: 1, completedItems: 1, totalItems: 10, errorCode: null, createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z", etag: '"aio_tags-r1"' };
}

it("does not poll when the current page has not submitted a tag job", async () => {
  const refresh = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
  renderHook(() => useAITagJobCoverageRefresh("lib_1", null), { wrapper });
  await act(() => vi.advanceTimersByTimeAsync(6_000));
  expect(refresh).not.toHaveBeenCalled();
});

it.each(["queued", "running", "cancelling"] as const)("refreshes only the mounted library's first coverage page every 1.5 seconds while %s", async (state) => {
  client.setQueryData(aiKeys.operation("aio_tags"), operation(state));
  const refresh = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
  const view = renderHook(() => useAITagJobCoverageRefresh("lib_1", "aio_tags"), { wrapper });
  await act(() => vi.advanceTimersByTimeAsync(1_499));
  expect(refresh).not.toHaveBeenCalled();
  await act(() => vi.advanceTimersByTimeAsync(1));
  expect(refresh).toHaveBeenCalledExactlyOnceWith({ queryKey: aiKeys.suggestions("lib_1", "pending"), exact: true, refetchType: "active" }, { cancelRefetch: false });
  view.unmount();
  await act(() => vi.advanceTimersByTimeAsync(6_000));
  expect(refresh).toHaveBeenCalledOnce();
});

it.each(["succeeded", "failed", "cancelled"] as const)("stops coverage refresh when the shared operation cache reaches %s", async (state) => {
  client.setQueryData(aiKeys.operation("aio_tags"), operation("running"));
  const refresh = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
  renderHook(() => useAITagJobCoverageRefresh("lib_1", "aio_tags"), { wrapper });
  await act(() => vi.advanceTimersByTimeAsync(1_500));
  await act(async () => {
    client.setQueryData(aiKeys.operation("aio_tags"), { ...operation(state), revision: 2 });
    await vi.advanceTimersByTimeAsync(0);
  });
  await act(() => vi.advanceTimersByTimeAsync(6_000));
  expect(refresh).toHaveBeenCalledOnce();
});

it("stops the previous library timer after switching to a page with no submitted operation", async () => {
  client.setQueryData(aiKeys.operation("aio_tags"), operation("running"));
  const refresh = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
  const view = renderHook(({ libraryId, operationId }: { libraryId: string; operationId: string | null }) => useAITagJobCoverageRefresh(libraryId, operationId), {
    initialProps: { libraryId: "lib_1", operationId: "aio_tags" as string | null }, wrapper,
  });
  await act(() => vi.advanceTimersByTimeAsync(1_500));
  view.rerender({ libraryId: "lib_2", operationId: null });
  await act(() => vi.advanceTimersByTimeAsync(6_000));
  expect(refresh).toHaveBeenCalledOnce();
});

it("does not refresh coverage in a hidden browser tab", async () => {
  client.setQueryData(aiKeys.operation("aio_tags"), operation("running"));
  const visibility = vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden");
  const refresh = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
  renderHook(() => useAITagJobCoverageRefresh("lib_1", "aio_tags"), { wrapper });
  await act(() => vi.advanceTimersByTimeAsync(3_000));
  expect(refresh).not.toHaveBeenCalled();
  visibility.mockReturnValue("visible");
  await act(() => vi.advanceTimersByTimeAsync(1_500));
  expect(refresh).toHaveBeenCalledOnce();
});
