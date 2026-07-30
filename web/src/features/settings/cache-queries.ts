import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getCacheSummary,
  startCacheCleanup,
} from "../../lib/api/cache";

export const cacheKeys = {
  summary: ["cache", "summary"] as const,
};

export function useCacheSummaryQuery() {
  return useQuery({
    queryKey: cacheKeys.summary,
    queryFn: getCacheSummary,
    refetchInterval: (query) => {
      const status = query.state.data?.cleanup.status;
      return status === "queued" || status === "running" ? 1_500 : false;
    },
  });
}

export function useStartCacheCleanupMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: startCacheCleanup,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: cacheKeys.summary }),
  });
}
