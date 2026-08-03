import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getMediaFailure,
  listMediaFailures,
  repairLibraryMediaProcessing,
} from "../../lib/api/diagnostics";
import {
  listSystemEvents,
  type SystemEventLevel,
} from "../../lib/api/system-logs";

export const diagnosticsKeys = {
  all: ["diagnostics", "media-failures"] as const,
  failure: (jobId: string) => ["diagnostics", "media-failure", jobId] as const,
  systemEvents: (level?: SystemEventLevel, module?: string) =>
    ["diagnostics", "system-events", level ?? "all", module ?? "all"] as const,
};

export function useMediaFailureQuery(jobId: string, enabled = true) {
  return useQuery({
    queryKey: diagnosticsKeys.failure(jobId),
    queryFn: () => getMediaFailure(jobId),
    enabled,
  });
}

export function useMediaFailuresQuery(filters: {
  libraryId?: string | undefined;
  variant?: "grid" | "storyboard" | undefined;
  errorCode?:
    | "invalid_media"
    | "unsupported_media"
    | "media_processing_failed"
    | "media_processing_timeout"
    | "source_unavailable"
    | "cache_unavailable"
    | undefined;
} = {}) {
  return useInfiniteQuery({
    queryKey: [...diagnosticsKeys.all, filters] as const,
    queryFn: ({ pageParam }) =>
      listMediaFailures({
        ...filters,
        ...(pageParam ? { cursor: pageParam } : {}),
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.nextCursor ?? undefined,
  });
}

export function useSystemEventsQuery(filters: {
  level?: SystemEventLevel | undefined;
  module?: string | undefined;
} = {}) {
  return useInfiniteQuery({
    queryKey: diagnosticsKeys.systemEvents(filters.level, filters.module),
    queryFn: ({ pageParam }) =>
      listSystemEvents({
        ...filters,
        ...(pageParam ? { cursor: pageParam } : {}),
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.nextCursor ?? undefined,
  });
}

export function useRepairMediaProcessingMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: repairLibraryMediaProcessing,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: diagnosticsKeys.all });
    },
  });
}
