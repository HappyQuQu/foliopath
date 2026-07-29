import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

import {
  getDirectory,
  listAssets,
  listDirectories,
  type AssetPage,
  type AssetKind,
  type AssetSort,
  type SortOrder,
} from "../../lib/api/catalog";
import { mediaDerivedStatePending } from "../../lib/media/availability";

export const pendingThumbnailRefreshMs = 2_500;
export const pendingThumbnailRefreshPageBudget = 4;

export function pendingThumbnailRefreshInterval(
  pages: AssetPage[] | undefined,
): number | false {
  return pages &&
    pages.length <= pendingThumbnailRefreshPageBudget &&
    pages.some((page) => page.items.some(mediaDerivedStatePending))
    ? pendingThumbnailRefreshMs
    : false;
}

export const catalogKeys = {
  all: ["catalog"] as const,
  directories: (libraryId: string, parentId?: string) =>
    ["catalog", "directories", libraryId, parentId ?? "root"] as const,
  directory: (directoryId: string) =>
    ["catalog", "directory", directoryId] as const,
  assets: (
    libraryId: string,
    directoryId: string | undefined,
    recursive: boolean,
    kinds: AssetKind[] | undefined,
    sort: AssetSort,
    order: SortOrder,
  ) =>
    [
      "catalog",
      "assets",
      libraryId,
      directoryId ?? "root",
      recursive,
      kinds?.join(",") ?? "all",
      sort,
      order,
    ] as const,
};

export function useDirectoriesQuery({
  enabled = true,
  libraryId,
  parentId,
}: {
  enabled?: boolean;
  libraryId: string;
  parentId?: string | undefined;
}) {
  return useInfiniteQuery({
    queryKey: catalogKeys.directories(libraryId, parentId),
    queryFn: ({ pageParam }) =>
      listDirectories({
        libraryId,
        ...(parentId ? { parentId } : {}),
        ...(pageParam ? { cursor: pageParam } : {}),
      }),
    enabled: enabled && libraryId.length > 0,
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    staleTime: 15_000,
  });
}

export function useAssetsQuery({
  directoryId,
  kinds,
  libraryId,
  order,
  recursive,
  sort,
}: {
  directoryId?: string | undefined;
  kinds?: AssetKind[] | undefined;
  libraryId: string;
  order: SortOrder;
  recursive: boolean;
  sort: AssetSort;
}) {
  return useInfiniteQuery({
    queryKey: catalogKeys.assets(
      libraryId,
      directoryId,
      recursive,
      kinds,
      sort,
      order,
    ),
    queryFn: ({ pageParam }) =>
      listAssets({
        libraryId,
        ...(kinds ? { kinds } : {}),
        order,
        recursive,
        sort,
        ...(directoryId ? { directoryId } : {}),
        ...(pageParam ? { cursor: pageParam } : {}),
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    refetchInterval: (query) =>
      pendingThumbnailRefreshInterval(query.state.data?.pages),
    staleTime: 15_000,
  });
}

export function useDirectoryQuery(directoryId: string | undefined) {
  return useQuery({
    queryKey: catalogKeys.directory(directoryId ?? "root"),
    queryFn: () => {
      if (!directoryId) throw new Error("A directory ID is required.");
      return getDirectory(directoryId);
    },
    enabled: Boolean(directoryId),
    staleTime: 15_000,
  });
}
