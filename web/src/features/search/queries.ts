import {
  type InfiniteData,
  type QueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";

import {
  searchAssets,
  searchSemanticAssets,
  searchSemanticVideos,
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
      state.mode,
      state.directoryId ?? "root",
      state.recursive,
      state.q,
      state.kind,
      state.date,
      state.sort,
      state.order,
    ] as const,
};

export async function refreshSearchResults(
  queryClient: QueryClient,
  libraryId: string | undefined,
  state: SearchUrlState,
): Promise<void> {
  const queryKey = searchKeys.results(libraryId, state);
  queryClient.setQueryData<InfiniteData<AssetPage, unknown>>(
    queryKey,
    (current) =>
      current && current.pages.length > 1
        ? {
            pageParams: current.pageParams.slice(0, 1),
            pages: current.pages.slice(0, 1),
          }
        : current,
  );
  await queryClient.refetchQueries({
    exact: true,
    queryKey,
    type: "active",
  });
}

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
      if (state.mode === "semantic") {
        if (state.kind === "video") {
          return searchSemanticVideos({
            q: state.q,
            ...(libraryId && state.scope !== "all" ? { libraryId } : {}),
            ...(state.scope === "directory" && state.directoryId
              ? { directoryId: state.directoryId, recursive: state.recursive }
              : {}),
            ...(pageParam ? { cursor: pageParam } : {}),
          });
        }
        return searchSemanticAssets({
          q: state.q,
          ...(libraryId && state.scope !== "all" ? { libraryId } : {}),
          ...(state.scope === "directory" && state.directoryId
            ? { directoryId: state.directoryId, recursive: state.recursive }
            : {}),
          ...(pageParam ? { cursor: pageParam } : {}),
        });
      }
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
