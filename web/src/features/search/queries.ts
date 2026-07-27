import { useInfiniteQuery } from "@tanstack/react-query";

import {
  searchAssets,
  searchLibraryAssets,
  type AssetPage,
} from "../../lib/api/catalog";
import { pendingThumbnailRefreshInterval } from "../browse/queries";
import {
  kindsForSearch,
  modifiedRangeForSearch,
  type SearchUrlState,
} from "./urlState";

export const searchKeys = {
  all: ["search"] as const,
  results: (libraryId: string | undefined, state: SearchUrlState) =>
    [
      "search",
      "results",
      libraryId ?? "all",
      state.scope,
      state.directoryId ?? "root",
      state.recursive,
      state.q,
      state.kind,
      state.date,
      state.sort,
      state.order,
    ] as const,
};

export function useSearchResultsQuery({
  libraryId,
  state,
}: {
  libraryId?: string;
  state: SearchUrlState;
}) {
  const range = modifiedRangeForSearch(state.date);
  const kinds = kindsForSearch(state.kind);
  return useInfiniteQuery({
    queryKey: searchKeys.results(libraryId, state),
    queryFn: ({ pageParam }) => {
      const shared = {
        order: state.order,
        q: state.q,
        sort: state.sort,
        ...range,
        ...(kinds ? { kinds } : {}),
        ...(pageParam ? { cursor: pageParam } : {}),
      };
      if (state.scope === "all" || !libraryId) {
        return searchAssets(shared);
      }
      return searchLibraryAssets({
        ...shared,
        libraryId,
        ...(state.scope === "directory" && state.directoryId
          ? {
              directoryId: state.directoryId,
              recursive: state.recursive,
            }
          : {}),
      });
    },
    enabled: state.q.length > 0,
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage: AssetPage) =>
      lastPage.nextCursor ?? undefined,
    refetchInterval: (query) =>
      pendingThumbnailRefreshInterval(query.state.data?.pages),
    staleTime: 15_000,
  });
}
