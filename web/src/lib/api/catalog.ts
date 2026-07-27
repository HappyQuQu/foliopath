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
