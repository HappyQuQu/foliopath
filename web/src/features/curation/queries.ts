import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  addManualAssetTag,
  createTag,
  getAssetCuration,
  listFavorites,
  listTagAssets,
  listTags,
  replaceAssetTags,
  setFavorite,
} from "../../lib/api/curation";
import type { CuratedAssetPage, TagPage } from "../../lib/api/curation";
import type { AssetKind, AssetSort, SortOrder } from "../../lib/api/catalog";
import { catalogKeys } from "../browse";
import { searchKeys } from "../search";

export const curationKeys = {
  all: ["curation"] as const,
  asset: (assetId: string) => ["curation", "asset", assetId] as const,
  favorites: ["curation", "favorites"] as const,
  tagAssets: (tagId: string) => ["curation", "tag-assets", tagId] as const,
  tagAssetsRoot: ["curation", "tag-assets"] as const,
  tags: ["curation", "tags"] as const,
  tagCloud: (libraryId: string, q = "", cursor = "") =>
    ["curation", "tags", "image-cloud", libraryId, q, cursor] as const,
};

export function useAssetCurationQuery(assetId: string | undefined) {
  return useQuery({
    enabled: Boolean(assetId),
    queryFn: () => {
      if (!assetId) throw new Error("An asset ID is required.");
      return getAssetCuration(assetId);
    },
    queryKey: curationKeys.asset(assetId ?? "inactive"),
  });
}

export interface CurationAssetQuery {
  kinds?: AssetKind[];
  libraryId?: string;
  order: SortOrder;
  sort: AssetSort | "favoritedAt";
}

export function useFavoritesQuery(options: CurationAssetQuery = { order: "desc", sort: "favoritedAt" }, enabled = true) {
  return useInfiniteQuery({
    enabled,
    getNextPageParam: (page: CuratedAssetPage) => page.nextCursor ?? undefined,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => listFavorites({
      ...options,
      ...(pageParam ? { cursor: pageParam as string } : {}),
    }),
    queryKey: [...curationKeys.favorites, options],
  });
}

export function useTagsQuery() {
  return useInfiniteQuery({
    getNextPageParam: (page: TagPage) => page.nextCursor ?? undefined,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => listTags(pageParam as string | undefined),
    queryKey: curationKeys.tags,
  });
}

export function useLibraryTagCloudQuery(libraryId: string, q = "", cursor = "") {
  return useQuery({
    enabled: Boolean(libraryId),
    queryKey: curationKeys.tagCloud(libraryId, q, cursor),
    queryFn: () => {
      if (!libraryId) throw new Error("A library ID is required.");
      return listTags(cursor || undefined, {
        libraryId,
        kinds: ["image", "animated"],
        ...(q ? { q } : {}),
      });
    },
  });
}

export function useTagAssetsQuery(
  tagId: string | undefined,
  options: CurationAssetQuery = { order: "desc", sort: "modifiedAt" },
  enabled = true,
) {
  return useInfiniteQuery({
    enabled: Boolean(tagId) && enabled,
    getNextPageParam: (page: CuratedAssetPage) => page.nextCursor ?? undefined,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => {
      if (!tagId) throw new Error("A tag ID is required.");
      return listTagAssets({
        tagId,
        ...(options.kinds ? { kinds: options.kinds } : {}),
        ...(options.libraryId ? { libraryId: options.libraryId } : {}),
        order: options.order,
        sort: options.sort === "favoritedAt" ? "modifiedAt" : options.sort,
        ...(pageParam ? { cursor: pageParam as string } : {}),
      });
    },
    queryKey: [...curationKeys.tagAssets(tagId ?? "inactive"), options],
  });
}

export function useFavoriteMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: setFavorite,
    onSuccess: (state) => {
      queryClient.setQueryData(curationKeys.asset(state.assetId), state);
      void queryClient.invalidateQueries({ queryKey: curationKeys.favorites });
      void queryClient.invalidateQueries({ queryKey: catalogKeys.all });
      void queryClient.invalidateQueries({ queryKey: searchKeys.all });
    },
  });
}

export function useCreateAndAssignTagMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ assetId, csrfToken, name }: { assetId: string; csrfToken: string; name: string }) => {
      const tag = await createTag({ csrfToken, name });
      const current = await getAssetCuration(assetId);
      return addManualAssetTag({
        assetId,
        csrfToken,
        revision: current.revision,
        tagId: tag.id,
      });
    },
    onSuccess: (state) => {
      queryClient.setQueryData(curationKeys.asset(state.assetId), state);
      void queryClient.invalidateQueries({ queryKey: curationKeys.tags });
      for (const tag of state.tags) {
        void queryClient.invalidateQueries({ queryKey: curationKeys.tagAssets(tag.id) });
      }
    },
    onError: (_error, input) => {
      // Creating the global label can succeed before its CAS assignment fails.
      // Refresh both resources, but never silently replay the user's edit.
      void queryClient.invalidateQueries({ queryKey: curationKeys.asset(input.assetId) });
      void queryClient.invalidateQueries({ queryKey: curationKeys.tags });
    },
  });
}

export function useAddManualAssetTagMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: addManualAssetTag,
    onSuccess: state => {
      queryClient.setQueryData(curationKeys.asset(state.assetId), state);
      void queryClient.invalidateQueries({ queryKey: curationKeys.tags });
      void queryClient.invalidateQueries({ queryKey: curationKeys.tagAssetsRoot });
    },
    onError: (_error, input) => {
      void queryClient.invalidateQueries({ queryKey: curationKeys.asset(input.assetId) });
    },
  });
}

export function useReplaceAssetTagsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: replaceAssetTags,
    onSuccess: (state) => {
      queryClient.setQueryData(curationKeys.asset(state.assetId), state);
      void queryClient.invalidateQueries({ queryKey: curationKeys.tags });
      void queryClient.invalidateQueries({ queryKey: curationKeys.tagAssetsRoot });
    },
    // Refresh, but never silently replay a stale full-set edit over AI commits.
    onError: (_error, input) => {
      void queryClient.invalidateQueries({ queryKey: curationKeys.asset(input.assetId) });
    },
  });
}
