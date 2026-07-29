import { ApiError } from "../api/errors";

interface InfiniteNextPageResult {
  isError: boolean;
}

interface RetryableInfiniteQuery {
  error: unknown;
  loadNextPage: () => Promise<InfiniteNextPageResult>;
  refresh: () => Promise<InfiniteNextPageResult>;
}

export async function retryInfiniteNextPage(
  query: RetryableInfiniteQuery,
): Promise<void> {
  if (!(query.error instanceof ApiError) || query.error.code !== "invalid_cursor") {
    await query.loadNextPage();
    return;
  }

  const refreshed = await query.refresh();
  if (!refreshed.isError) {
    await query.loadNextPage();
  }
}
