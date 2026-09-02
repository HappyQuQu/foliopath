import { apiClient } from "./client";
import { createApiError } from "./errors";
import type { components } from "./generated/schema";

export interface Breadcrumb {
  id: string;
  name: string;
  relativePath: string;
}

export interface CatalogStateResult {
  contentRevision: number | null;
  etag: string;
}

export async function getCatalogState(etag?: string): Promise<CatalogStateResult> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/catalog/state", {
      params: {
        header: {
          ...(etag ? { "If-None-Match": etag } : {}),
        },
      },
    });
    const nextEtag = response.headers.get("ETag") ?? etag;
    if (response.status === 304 && nextEtag) {
      return { contentRevision: null, etag: nextEtag };
    }
    if (data && nextEtag) {
      return { contentRevision: data.contentRevision, etag: nextEtag };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export interface Directory {
  directAssetCount: number;
  hasChildren: boolean;
  id: string;
  libraryId: string;
  name: string;
  parentId: string | null;
  recursiveAssetCount: number;
  relativePath: string;
}

export interface DirectoryDetail extends Directory {
  breadcrumbs: Breadcrumb[];
}

export interface DirectoryPage {
  items: Directory[];
  nextCursor: string | null;
}

export type AssetKind = "image" | "animated" | "video";
export type AssetSort = "name" | "modifiedAt" | "size";
export type SortOrder = "asc" | "desc";
export type StoryboardReference = components["schemas"]["StoryboardReference"];

export interface Asset {
  directoryId: string;
  durationMs: number | null;
  favorite?: boolean;
  height: number | null;
  id: string;
  kind: AssetKind;
  libraryId: string;
  libraryName: string;
  mimeType: string;
  modifiedAt: string;
  name: string;
  playbackStatus:
    "playable" | "unsupported_codec" | "not_applicable" | "unknown";
  probeStatus: "pending" | "ready" | "failed" | "unsupported";
  relativePath: string;
  sizeBytes: number;
  sourceAvailability: "available" | "offline" | "missing" | "unreadable";
  storyboard: StoryboardReference;
  thumbnail: {
    errorCode: string | null;
    status: "pending" | "ready" | "failed" | "unavailable";
    url: string | null;
  };
  width: number | null;
}

export interface AssetPage {
  counts?: AssetCounts;
  items: Asset[];
  nextCursor: string | null;
  semanticCoverage?: SemanticSearchCoverage;
  semanticVideoMatches?: Record<string, SemanticVideoMatch>;
}

export interface SemanticVideoMatch {
  ordinal: number;
  planSize: 4 | 10;
  timestampMs: number;
}

export type SemanticSearchCoverage = components["schemas"]["SemanticSearchCoverage"];

export interface SemanticAssetPage extends AssetPage {
  semanticCoverage: SemanticSearchCoverage;
}

export interface AssetCounts {
  all: number;
  images: number;
  videos: number;
}

export interface SearchAssetsInput {
  cursor?: string;
  kinds?: AssetKind[];
  limit?: number;
  modifiedBefore?: string;
  modifiedFrom?: string;
  order: SortOrder;
  q: string;
  sort: AssetSort;
}

export function assetContentUrl(assetId: string): string {
  return `/api/v1/assets/${encodeURIComponent(assetId)}/content`;
}

export async function listAssets({
  cursor,
  directoryId,
  kinds,
  libraryId,
  limit = 50,
  order,
  q,
  recursive,
  sort,
}: {
  cursor?: string;
  directoryId?: string;
  kinds?: AssetKind[];
  libraryId: string;
  limit?: number;
  order: SortOrder;
  q?: string;
  recursive: boolean;
  sort: AssetSort;
}): Promise<AssetPage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/libraries/{libraryId}/assets",
      {
        params: {
          path: { libraryId },
          query: {
            limit,
            order,
            sort,
            ...(cursor ? { cursor } : {}),
            ...(directoryId ? { directoryId } : {}),
            ...(kinds?.length ? { kind: kinds } : {}),
            ...(q ? { q } : {}),
            ...(q && !directoryId
              ? { recursive }
              : recursive
                ? { recursive: true }
                : {}),
          },
        },
      },
    );
    if (data) return mapAssetPage(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getAsset(assetId: string): Promise<Asset> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/assets/{assetId}",
      {
        params: { path: { assetId } },
      },
    );
    if (data) return mapAsset(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function searchLibraryAssets({
  cursor,
  directoryId,
  kinds,
  libraryId,
  limit = 50,
  modifiedBefore,
  modifiedFrom,
  order,
  q,
  recursive,
  sort,
}: SearchAssetsInput & {
  directoryId?: string;
  libraryId: string;
  recursive?: boolean;
}): Promise<AssetPage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/libraries/{libraryId}/assets",
      {
        params: {
          path: { libraryId },
          query: {
            limit,
            order,
            q,
            sort,
            ...(cursor ? { cursor } : {}),
            ...(directoryId ? { directoryId } : {}),
            ...(kinds?.length ? { kind: kinds } : {}),
            ...(modifiedBefore ? { modifiedBefore } : {}),
            ...(modifiedFrom ? { modifiedFrom } : {}),
            ...(directoryId && recursive ? { recursive: true } : {}),
          },
        },
      },
    );
    if (data) return mapAssetPage(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function searchAssets({
  cursor,
  kinds,
  limit = 50,
  modifiedBefore,
  modifiedFrom,
  order,
  q,
  sort,
}: SearchAssetsInput): Promise<AssetPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/assets", {
      params: {
        query: {
          limit,
          order,
          q,
          sort,
          ...(cursor ? { cursor } : {}),
          ...(kinds?.length ? { kind: kinds } : {}),
          ...(modifiedBefore ? { modifiedBefore } : {}),
          ...(modifiedFrom ? { modifiedFrom } : {}),
        },
      },
    });
    if (data) return mapAssetPage(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function searchSemanticAssets({
  cursor,
  directoryId,
  libraryId,
  limit = 50,
  q,
  recursive,
}: {
  cursor?: string;
  directoryId?: string;
  libraryId?: string;
  limit?: number;
  q: string;
  recursive?: boolean;
}): Promise<SemanticAssetPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/semantic/assets", {
      params: {
        query: {
          limit,
          q,
          ...(cursor ? { cursor } : {}),
          ...(libraryId ? { libraryId } : {}),
          ...(directoryId ? { directoryId } : {}),
          ...(directoryId && recursive ? { recursive: true } : {}),
        },
      },
    });
    if (data) return { ...mapAssetPage(data), semanticCoverage: data.coverage };
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function searchSemanticVideos({
  cursor,
  directoryId,
  libraryId,
  limit = 50,
  q,
  recursive,
}: {
  cursor?: string;
  directoryId?: string;
  libraryId?: string;
  limit?: number;
  q: string;
  recursive?: boolean;
}): Promise<AssetPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/semantic/videos", {
      params: {
        query: {
          limit,
          q,
          ...(cursor ? { cursor } : {}),
          ...(libraryId ? { libraryId } : {}),
          ...(directoryId ? { directoryId } : {}),
          ...(directoryId && recursive ? { recursive: true } : {}),
        },
      },
    });
    if (data) {
      return {
        items: data.items.map((hit) => mapAsset(hit.asset)),
        nextCursor: data.nextCursor,
        semanticCoverage: {
          complete: data.coverage.complete,
          completed: data.coverage.completed,
          eligible: data.coverage.eligible,
          excludedLibraries: data.coverage.excludedLibraries ?? [],
          failed: data.coverage.failed,
          stale: data.coverage.stale,
        },
        semanticVideoMatches: Object.fromEntries(
          data.items.map((hit) => [hit.asset.id, hit.matchedFrame]),
        ),
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function listDirectories({
  cursor,
  libraryId,
  limit = 50,
  parentId,
  q,
}: {
  cursor?: string;
  libraryId: string;
  limit?: number;
  parentId?: string;
  q?: string;
}): Promise<DirectoryPage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/libraries/{libraryId}/directories",
      {
        params: {
          path: { libraryId },
          query: {
            limit,
            ...(cursor ? { cursor } : {}),
            ...(parentId ? { parentId } : {}),
            ...(q ? { q } : {}),
          },
        },
      },
    );
    if (data) {
      return {
        items: data.items.map(mapDirectory),
        nextCursor: data.nextCursor,
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getDirectory(
  directoryId: string,
): Promise<DirectoryDetail> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/directories/{directoryId}",
      {
        params: { path: { directoryId } },
      },
    );
    if (data) {
      return {
        ...mapDirectory(data),
        breadcrumbs: data.breadcrumbs.map((item) => ({ ...item })),
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function mapDirectory(data: {
  directAssetCount: number;
  hasChildren: boolean;
  id: string;
  libraryId: string;
  name: string;
  parentId: string | null;
  recursiveAssetCount: number;
  relativePath: string;
}): Directory {
  return { ...data };
}

function mapAssetPage(data: {
  counts?: AssetCounts;
  items: Asset[];
  nextCursor: string | null;
}): AssetPage {
  return {
    ...(data.counts ? { counts: { ...data.counts } } : {}),
    items: data.items.map(mapAsset),
    nextCursor: data.nextCursor,
  };
}

function mapAsset(data: Asset): Asset {
  return {
    ...data,
    storyboard: { ...data.storyboard },
    thumbnail: { ...data.thumbnail },
  };
}
