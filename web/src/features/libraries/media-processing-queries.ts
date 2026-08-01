import { useQuery } from "@tanstack/react-query";

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

export function mediaProcessingRefreshInterval(
  progress: MediaProcessingProgress | undefined,
  scanActive: boolean,
) {
  return scanActive || progress?.active ? 1_500 : false;
}
