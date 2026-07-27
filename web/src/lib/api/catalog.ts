import { apiClient } from "./client";
import { createApiError } from "./errors";

export interface Breadcrumb {
  id: string;
  name: string;
  relativePath: string;
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
export type AssetSort = "name" | "modifiedAt";
export type SortOrder = "asc" | "desc";

export interface Asset {
  directoryId: string;
  durationMs: number | null;
  height: number | null;
  id: string;
  kind: AssetKind;
  libraryId: string;
  libraryName: string;
  mimeType: string;
  modifiedAt: string;
  name: string;
  playbackStatus: "playable" | "unsupported_codec" | "not_applicable" | "unknown";
  probeStatus: "pending" | "ready" | "failed" | "unsupported";
  relativePath: string;
  sizeBytes: number;
  sourceAvailability: "available" | "offline" | "missing" | "unreadable";
  thumbnail: {
    errorCode: string | null;
    status: "pending" | "ready" | "failed" | "unavailable";
    url: string | null;
  };
  width: number | null;
}

export interface AssetPage {
  items: Asset[];
  nextCursor: string | null;
}

export async function listAssets({
  cursor,
  directoryId,
  libraryId,
  limit = 50,
  order,
  recursive,
  sort,
}: {
  cursor?: string;
  directoryId?: string;
  libraryId: string;
  limit?: number;
  order: SortOrder;
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
            ...(recursive ? { recursive: true } : {}),
          },
        },
      },
    );
    if (data) {
      return {
        items: data.items.map((item) => ({
          ...item,
          thumbnail: { ...item.thumbnail },
        })),
        nextCursor: data.nextCursor,
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
}: {
  cursor?: string;
  libraryId: string;
  limit?: number;
  parentId?: string;
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

export async function getDirectory(directoryId: string): Promise<DirectoryDetail> {
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
