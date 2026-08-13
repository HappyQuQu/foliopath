import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
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
  tags: ["curation", "tags"] as const,
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

export function useFavoritesQuery(options: CurationAssetQuery = { order: "desc", sort: "favoritedAt" }) {
  return useInfiniteQuery({
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

export function useTagAssetsQuery(
  tagId: string | undefined,
  options: CurationAssetQuery = { order: "desc", sort: "modifiedAt" },
) {
  return useInfiniteQuery({
    enabled: Boolean(tagId),
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
      if (current.tags.some((item) => item.id === tag.id)) return current;
      return replaceAssetTags({
        assetId,
        csrfToken,
        revision: current.revision,
        tagIds: [...current.tags.map((item) => item.id), tag.id],
      });
    },
    onSuccess: (state) => {
      queryClient.setQueryData(curationKeys.asset(state.assetId), state);
      void queryClient.invalidateQueries({ queryKey: curationKeys.tags });
      for (const tag of state.tags) {
        void queryClient.invalidateQueries({ queryKey: curationKeys.tagAssets(tag.id) });
      }
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
      void queryClient.invalidateQueries({ queryKey: ["curation", "tag-assets"] });
    },
  });
}
