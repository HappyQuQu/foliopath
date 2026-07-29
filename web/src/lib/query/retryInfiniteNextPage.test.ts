import { expect, it, vi } from "vitest";

import { ApiError } from "../api/errors";
import { retryInfiniteNextPage } from "./retryInfiniteNextPage";

it("retries a transient next-page failure without refreshing loaded pages", async () => {
  const loadNextPage = vi.fn().mockResolvedValue({
    isError: false,
  });
  const refresh = vi.fn();

  await retryInfiniteNextPage({
    error: new ApiError({
      code: "network_error",
      message: "offline",
      requestId: undefined,
      status: 0,
    }),
    loadNextPage,
    refresh,
  });

  expect(loadNextPage).toHaveBeenCalledOnce();
  expect(refresh).not.toHaveBeenCalled();
});

it("refreshes an expired cursor chain before loading the next page", async () => {
  const loadNextPage = vi.fn().mockResolvedValue({
    isError: false,
  });
  const refresh = vi.fn().mockResolvedValue({
    isError: false,
  });

  await retryInfiniteNextPage({
    error: new ApiError({
      code: "invalid_cursor",
      message: "expired",
      requestId: "req_test",
      status: 400,
    }),
    loadNextPage,
    refresh,
  });

  expect(refresh).toHaveBeenCalledOnce();
  expect(loadNextPage).toHaveBeenCalledOnce();
  expect(refresh.mock.invocationCallOrder[0]).toBeLessThan(
    loadNextPage.mock.invocationCallOrder[0]!,
  );
});

it("does not request another page when refreshing the cursor chain fails", async () => {
  const loadNextPage = vi.fn();
  const refresh = vi.fn().mockResolvedValue({
    isError: true,
  });

  await retryInfiniteNextPage({
    error: new ApiError({
      code: "invalid_cursor",
      message: "expired",
      requestId: undefined,
      status: 400,
    }),
    loadNextPage,
    refresh,
  });

  expect(loadNextPage).not.toHaveBeenCalled();
});
