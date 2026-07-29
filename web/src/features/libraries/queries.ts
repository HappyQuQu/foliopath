import {
  type QueryClient,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  getLibraryRemoval,
  getLibrary,
  createLibrary,
  listLibraries,
  listLibraryPaths,
  removeLibrary,
  renameLibrary,
  type CreateLibraryInput,
} from "../../lib/api/libraries";

export const libraryKeys = {
  all: ["libraries"] as const,
  list: () => ["libraries", "list"] as const,
  detail: (libraryId: string) => ["libraries", "detail", libraryId] as const,
  paths: (parent: string) => ["libraries", "paths", parent] as const,
  removal: (removalId: string) => ["libraries", "removal", removalId] as const,
};

export async function refreshLibraryDetail(
  queryClient: QueryClient,
  libraryId: string,
): Promise<void> {
  await queryClient.refetchQueries({
    exact: true,
    queryKey: libraryKeys.detail(libraryId),
    type: "active",
  });
}

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

export function useLibraryQuery(libraryId: string) {
  return useQuery({
    queryKey: libraryKeys.detail(libraryId),
    queryFn: () => {
      if (!libraryId) throw new Error("A library ID is required.");
      return getLibrary(libraryId);
    },
    enabled: libraryId.length > 0,
    staleTime: 5_000,
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

export function useRenameLibraryMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: renameLibrary,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}

export function useRemoveLibraryMutation() {
  return useMutation({
    mutationFn: removeLibrary,
  });
}

export function useLibraryRemovalQuery(removalId: string | undefined) {
  return useQuery({
    queryKey: libraryKeys.removal(removalId ?? "inactive"),
    queryFn: () => {
      if (!removalId) throw new Error("Removal query is not active.");
      return getLibraryRemoval(removalId);
    },
    enabled: Boolean(removalId),
    refetchInterval: (query) => {
      const removal = query.state.data;
      return removal?.status === "queued" || removal?.status === "running"
        ? 1_000
        : false;
    },
  });
}
