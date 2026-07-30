import {
  type InfiniteData,
  type QueryClient,
  useInfiniteQuery,
  useQuery,
} from "@tanstack/react-query";

import {
  getDirectory,
  listAssets,
  listDirectories,
  type AssetPage,
  type AssetKind,
  type AssetSort,
  type DirectoryPage,
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
  directories: (libraryId: string, parentId: string | undefined, q: string) =>
    ["catalog", "directories", libraryId, parentId ?? "root", q] as const,
  directory: (directoryId: string) =>
    ["catalog", "directory", directoryId] as const,
  assets: (
    libraryId: string,
    directoryId: string | undefined,
    recursive: boolean,
    q: string,
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
      q,
      kinds?.join(",") ?? "all",
      sort,
      order,
    ] as const,
};

export interface CatalogScope {
  directoryId?: string | undefined;
  kinds?: AssetKind[] | undefined;
  libraryId: string;
  order: SortOrder;
  q: string;
  recursive: boolean;
  sort: AssetSort;
}

export async function refreshCatalogScope(
  queryClient: QueryClient,
  scope: CatalogScope,
): Promise<void> {
  const directoriesKey = catalogKeys.directories(
    scope.libraryId,
    scope.directoryId,
    scope.q,
  );
  const assetsKey = catalogKeys.assets(
    scope.libraryId,
    scope.directoryId,
    scope.recursive,
    scope.q,
    scope.kinds,
    scope.sort,
    scope.order,
  );

  keepOnlyFirstPage<DirectoryPage>(queryClient, directoriesKey);
  keepOnlyFirstPage<AssetPage>(queryClient, assetsKey);

  const keys = [
    directoriesKey,
    assetsKey,
    ...(scope.directoryId ? [catalogKeys.directory(scope.directoryId)] : []),
  ];
  await Promise.all(
    keys.map((queryKey) =>
      queryClient.refetchQueries({ exact: true, queryKey, type: "active" }),
    ),
  );
}

export function useDirectoriesQuery({
  enabled = true,
  libraryId,
  parentId,
  q = "",
}: {
  enabled?: boolean;
  libraryId: string;
  parentId?: string | undefined;
  q?: string | undefined;
}) {
  return useInfiniteQuery({
    queryKey: catalogKeys.directories(libraryId, parentId, q),
    queryFn: ({ pageParam }) =>
      listDirectories({
        libraryId,
        ...(parentId ? { parentId } : {}),
        ...(q ? { q } : {}),
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
  q = "",
  recursive,
  sort,
}: {
  directoryId?: string | undefined;
  kinds?: AssetKind[] | undefined;
  libraryId: string;
  order: SortOrder;
  q?: string | undefined;
  recursive: boolean;
  sort: AssetSort;
}) {
  return useInfiniteQuery({
    queryKey: catalogKeys.assets(
      libraryId,
      directoryId,
      recursive,
      q,
      kinds,
      sort,
      order,
    ),
    queryFn: ({ pageParam }) =>
      listAssets({
        libraryId,
        ...(kinds ? { kinds } : {}),
        order,
        ...(q ? { q } : {}),
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

function keepOnlyFirstPage<Page>(
  queryClient: QueryClient,
  queryKey: readonly unknown[],
): void {
  queryClient.setQueryData<InfiniteData<Page, unknown>>(queryKey, (current) =>
    current && current.pages.length > 1
      ? {
          pageParams: current.pageParams.slice(0, 1),
          pages: current.pages.slice(0, 1),
        }
      : current,
  );
}
