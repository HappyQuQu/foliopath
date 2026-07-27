import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

import {
  getDirectory,
  listDirectories,
} from "../../lib/api/catalog";

export const catalogKeys = {
  all: ["catalog"] as const,
  directories: (libraryId: string, parentId?: string) =>
    ["catalog", "directories", libraryId, parentId ?? "root"] as const,
  directory: (directoryId: string) =>
    ["catalog", "directory", directoryId] as const,
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
