import { useQueries, useQuery } from "@tanstack/react-query";

import type { LibrarySummary } from "../../lib/api/libraries";
import {
  getLibraryMediaProcessingProgress,
  type MediaProcessingProgress,
} from "../../lib/api/media-processing";

export const mediaProcessingKeys = {
  library: (libraryId: string) =>
    ["media-processing", "library", libraryId] as const,
};

export function useMediaProcessingProgressQuery(
  libraryId: string,
  scanActive: boolean,
) {
  return useQuery({
    queryKey: mediaProcessingKeys.library(libraryId),
    queryFn: () => getLibraryMediaProcessingProgress(libraryId),
    enabled: libraryId.length > 0,
    refetchInterval: (query) =>
      mediaProcessingRefreshInterval(query.state.data, scanActive),
  });
}

export function useLibrariesMediaProcessingProgressQueries(
  libraries: LibrarySummary[],
) {
  return useQueries({
    queries: libraries.map((library) => ({
      queryKey: mediaProcessingKeys.library(library.id),
      queryFn: () => getLibraryMediaProcessingProgress(library.id),
      refetchInterval: (query: {
        state: { data: MediaProcessingProgress | undefined };
      }) =>
        mediaProcessingRefreshInterval(
          query.state.data,
          library.status === "scanning",
        ),
      staleTime: 1_000,
    })),
  });
}

export function mediaProcessingRefreshInterval(
  progress: MediaProcessingProgress | undefined,
  scanActive: boolean,
) {
  return scanActive || mediaProcessingIsActive(progress) ? 1_500 : false;
}

export function mediaProcessingIsActive(
  progress: MediaProcessingProgress | undefined,
) {
  if (!progress) return false;
  return (
    progress.active ||
    progress.thumbnails.queued > 0 ||
    progress.thumbnails.running > 0 ||
    progress.videoPreviews.queued > 0 ||
    progress.videoPreviews.running > 0 ||
    progress.videoPreviewsPendingEligibility > 0
  );
}
