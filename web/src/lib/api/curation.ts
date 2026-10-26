import { apiClient } from "./client";
import { createApiError } from "./errors";
import type { Asset, AssetCounts, AssetKind, AssetSort, SortOrder } from "./catalog";

export interface Tag {
  assetCount: number;
  createdAt: string;
  id: string;
  name: string;
  updatedAt: string;
  localization?: { key: string; zhCN: string; en: string };
}

export interface AssetCuration {
  assetId: string;
  favorite: boolean;
  favoritedAt: string | null;
  revision: number;
  tags: AssetTag[];
}

export interface AssetTag extends Tag {
  /** Older server responses omitted source; they only supported manual tags. */
  source?: "manual" | "ai";
}

export async function addManualAssetTag(input: { assetId: string; tagId: string; revision: number; csrfToken: string }): Promise<AssetCuration> {
  try {
    const { data, error, response } = await apiClient.PUT("/api/v1/assets/{assetId}/tags/{tagId}", {
      headers: { "X-CSRF-Token": input.csrfToken },
      params: { path: { assetId: input.assetId, tagId: input.tagId }, header: { "If-Match": `"curation-r${input.revision}"` } },
    });
    if (data) return mapAssetCuration(data);
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}

export interface CuratedAsset {
  asset: Asset;
  curation: AssetCuration;
}

export interface CuratedAssetPage {
  counts: AssetCounts;
  items: CuratedAsset[];
  nextCursor: string | null;
}

export interface TagPage {
  items: Tag[];
  nextCursor: string | null;
}

export async function getAssetCuration(assetId: string): Promise<AssetCuration> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/assets/{assetId}/curation",
      { params: { path: { assetId } } },
    );
    if (data) return mapAssetCuration(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function setFavorite({
  assetId,
  csrfToken,
  favorite,
}: {
  assetId: string;
  csrfToken: string;
  favorite: boolean;
}): Promise<AssetCuration> {
  try {
    const { data, error, response } = await apiClient.PUT(
      "/api/v1/assets/{assetId}/favorite",
      {
        body: { favorite },
        headers: { "X-CSRF-Token": csrfToken },
        params: { path: { assetId } },
      },
    );
    if (data) return mapAssetCuration(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function listFavorites({
  cursor,
  kinds,
  libraryId,
  limit = 50,
  order = "desc",
  sort = "favoritedAt",
}: {
  cursor?: string;
  kinds?: AssetKind[];
  libraryId?: string;
  limit?: number;
  order?: SortOrder;
  sort?: AssetSort | "favoritedAt";
} = {}): Promise<CuratedAssetPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/favorites", {
      params: {
        query: {
          limit,
          order,
          sort,
          ...(cursor ? { cursor } : {}),
          ...(kinds?.length ? { kind: kinds } : {}),
          ...(libraryId ? { libraryId } : {}),
        },
      },
    });
    if (data) return mapCuratedAssetPage(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function listTagAssets({
  cursor,
  kinds,
  libraryId,
  limit = 50,
  order = "desc",
  sort = "modifiedAt",
  tagId,
}: {
  cursor?: string;
  kinds?: AssetKind[];
  libraryId?: string;
  limit?: number;
  order?: SortOrder;
  sort?: AssetSort;
  tagId: string;
}): Promise<CuratedAssetPage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/tags/{tagId}/assets",
      {
        params: {
          path: { tagId },
          query: {
            limit,
            order,
            sort,
            ...(cursor ? { cursor } : {}),
            ...(kinds?.length ? { kind: kinds } : {}),
            ...(libraryId ? { libraryId } : {}),
          },
        },
      },
    );
    if (data) return mapCuratedAssetPage(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export interface TagListScope {
  kinds?: AssetKind[];
  libraryId?: string;
  q?: string;
}

// Omitted scope retains the global vocabulary, including unassigned tags.
export async function listTags(cursor?: string, scope: TagListScope = {}): Promise<TagPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/tags", {
      params: { query: {
        limit: 200,
        ...(cursor ? { cursor } : {}),
        ...(scope.libraryId ? { libraryId: scope.libraryId } : {}),
        ...(scope.kinds?.length ? { kind: scope.kinds } : {}),
        ...(scope.q ? { q: scope.q } : {}),
      } },
    });
    if (data) return { items: data.items.map(mapTag), nextCursor: data.nextCursor };
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function createTag({
  csrfToken,
  name,
}: {
  csrfToken: string;
  name: string;
}): Promise<Tag> {
  try {
    const { data, error, response } = await apiClient.POST("/api/v1/tags", {
      body: { name },
      headers: { "X-CSRF-Token": csrfToken },
    });
    if (data) return mapTag(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function replaceAssetTags({
  assetId,
  csrfToken,
  revision,
  tagIds,
}: {
  assetId: string;
  csrfToken: string;
  revision: number;
  tagIds: string[];
}): Promise<AssetCuration> {
  try {
    const { data, error, response } = await apiClient.PUT(
      "/api/v1/assets/{assetId}/tags",
      {
        body: { tagIds },
        headers: {
          "If-Match": `"curation-r${revision}"`,
          "X-CSRF-Token": csrfToken,
        },
        params: {
          header: { "If-Match": `"curation-r${revision}"` },
          path: { assetId },
        },
      },
    );
    if (data) return mapAssetCuration(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function mapTag(tag: Tag): Tag {
  return { ...tag };
}

function mapAssetCuration(state: AssetCuration): AssetCuration {
  return { ...state, tags: state.tags.map(tag => ({ ...mapTag(tag), source: tag.source ?? "manual" })) };
}

function mapCuratedAssetPage(page: CuratedAssetPage): CuratedAssetPage {
  return {
    counts: { ...page.counts },
    items: page.items.map((item) => ({
      asset: {
        ...item.asset,
        storyboard: { ...item.asset.storyboard },
        thumbnail: { ...item.asset.thumbnail },
      },
      curation: mapAssetCuration(item.curation),
    })),
    nextCursor: page.nextCursor,
  };
}
