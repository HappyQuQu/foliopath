import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

import {
  createLibrary,
  listLibraries,
  listLibraryPaths,
  type CreateLibraryInput,
} from "../../lib/api/libraries";

export const libraryKeys = {
  all: ["libraries"] as const,
  list: () => ["libraries", "list"] as const,
  paths: (parent: string) => ["libraries", "paths", parent] as const,
};

export function useLibrariesQuery() {
  return useInfiniteQuery({
    queryKey: libraryKeys.list(),
    queryFn: ({ pageParam }) =>
      listLibraries(pageParam ? { cursor: pageParam } : undefined),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    staleTime: 15_000,
  });
}

export function useLibraryPathsQuery(parent: string) {
  return useInfiniteQuery({
    queryKey: libraryKeys.paths(parent),
    queryFn: ({ pageParam }) =>
      listLibraryPaths({
        parent,
        ...(pageParam ? { cursor: pageParam } : {}),
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    staleTime: 15_000,
  });
}

export function useCreateLibraryMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateLibraryInput) => createLibrary(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}
