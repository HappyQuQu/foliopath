import { apiClient } from "./client";
import { createApiError } from "./errors";

export type LibraryStatus = "pending" | "scanning" | "ready" | "offline" | "error";

export interface LibrarySummary {
  assetCount: number;
  directoryCount: number;
  displayPath: string;
  id: string;
  lastSuccessfulScanAt: string | null;
  latestScanId: string | null;
  name: string;
  status: LibraryStatus;
}

export interface LibraryPage {
  items: LibrarySummary[];
  nextCursor: string | null;
}

export type LibraryPathBlockedReason =
  | "overlapping_library"
  | "ancestor_of_library"
  | "descendant_of_library"
  | "unreadable"
  | "symlink"
  | "mount_boundary"
  | "unavailable";

export interface LibraryPathLocation {
  name: string;
  relativePath: string;
}

export interface LibraryPathEntry extends LibraryPathLocation {
  conflictingLibraryName: string | null;
  hasChildren: boolean;
  selectable: boolean;
  selectionBlockedReason: LibraryPathBlockedReason | null;
}

export interface LibraryPathPage {
  breadcrumbs: LibraryPathLocation[];
  items: LibraryPathEntry[];
  location: LibraryPathLocation;
  nextCursor: string | null;
}

export interface CreateLibraryInput {
  csrfToken: string;
  idempotencyKey: string;
  name: string;
  rootPath: string;
}

export interface CreateLibraryResult {
  library: LibrarySummary;
  scanId: string;
}

export interface LibraryDocument {
  etag: string;
  library: LibrarySummary;
}

export type LibraryRemovalStatus = "queued" | "running" | "succeeded" | "failed";

export interface LibraryRemoval {
  errorCode: "cleanup_interrupted" | "application_data_unavailable" | "internal_error" | null;
  id: string;
  libraryId: string;
  libraryName: string;
  status: LibraryRemovalStatus;
}

export async function listLibraries({
  cursor,
  limit = 50,
}: {
  cursor?: string;
  limit?: number;
} = {}): Promise<LibraryPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/libraries", {
      params: {
        query: {
          limit,
          ...(cursor ? { cursor } : {}),
        },
      },
    });
    if (data) {
      return {
        items: data.items.map(mapLibrary),
        nextCursor: data.nextCursor,
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function listLibraryPaths({
  cursor,
  limit = 100,
  parent = "",
}: {
  cursor?: string;
  limit?: number;
  parent?: string;
} = {}): Promise<LibraryPathPage> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/library-paths", {
      params: {
        query: {
          limit,
          ...(cursor ? { cursor } : {}),
          ...(parent ? { parent } : {}),
        },
      },
    });
    if (data) {
      return {
        breadcrumbs: data.breadcrumbs.map((item) => ({ ...item })),
        items: data.items.map((item) => ({
          conflictingLibraryName: item.conflictingLibraryName,
          hasChildren: item.hasChildren,
          name: item.name,
          relativePath: item.relativePath,
          selectable: item.selectable,
          selectionBlockedReason: item.selectionBlockedReason,
        })),
        location: { ...data.location },
        nextCursor: data.nextCursor,
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function createLibrary(
  input: CreateLibraryInput,
): Promise<CreateLibraryResult> {
  try {
    const { data, error, response } = await apiClient.POST("/api/v1/libraries", {
      body: {
        name: input.name,
        rootPath: input.rootPath,
      },
      headers: {
        "X-CSRF-Token": input.csrfToken,
      },
      params: {
        header: {
          "Idempotency-Key": input.idempotencyKey,
        },
      },
    });
    if (data) {
      return {
        library: mapLibrary(data.library),
        scanId: data.scan.id,
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getLibrary(libraryId: string): Promise<LibraryDocument> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/libraries/{libraryId}",
      {
        params: { path: { libraryId } },
      },
    );
    if (data) {
      return {
        etag: requireEtag(response),
        library: mapLibrary(data),
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function renameLibrary({
  csrfToken,
  libraryId,
  name,
}: {
  csrfToken: string;
  libraryId: string;
  name: string;
}): Promise<LibrarySummary> {
  const current = await getLibrary(libraryId);

  try {
    const { data, error, response } = await apiClient.PATCH(
      "/api/v1/libraries/{libraryId}",
      {
        body: { name },
        headers: { "X-CSRF-Token": csrfToken },
        params: {
          header: { "If-Match": current.etag },
          path: { libraryId },
        },
      },
    );
    if (data) return mapLibrary(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function removeLibrary({
  csrfToken,
  idempotencyKey,
  libraryId,
}: {
  csrfToken: string;
  idempotencyKey: string;
  libraryId: string;
}): Promise<LibraryRemoval> {
  const current = await getLibrary(libraryId);

  try {
    const { data, error, response } = await apiClient.DELETE(
      "/api/v1/libraries/{libraryId}",
      {
        headers: { "X-CSRF-Token": csrfToken },
        params: {
          header: {
            "Idempotency-Key": idempotencyKey,
            "If-Match": current.etag,
          },
          path: { libraryId },
        },
      },
    );
    if (data) return mapRemoval(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getLibraryRemoval(
  removalId: string,
): Promise<LibraryRemoval> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/library-removals/{removalId}",
      {
        params: {
          header: {},
          path: { removalId },
        },
      },
    );
    if (data) return mapRemoval(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function mapLibrary(library: {
  assetCount: number;
  directoryCount: number;
  displayPath: string;
  id: string;
  lastSuccessfulScanAt: string | null;
  latestScanId: string | null;
  name: string;
  status: LibraryStatus;
}): LibrarySummary {
  return {
    assetCount: library.assetCount,
    directoryCount: library.directoryCount,
    displayPath: library.displayPath,
    id: library.id,
    lastSuccessfulScanAt: library.lastSuccessfulScanAt,
    latestScanId: library.latestScanId,
    name: library.name,
    status: library.status,
  };
}

function mapRemoval(removal: {
  errorCode: LibraryRemoval["errorCode"];
  id: string;
  libraryId: string;
  libraryName: string;
  status: LibraryRemovalStatus;
}): LibraryRemoval {
  return {
    errorCode: removal.errorCode,
    id: removal.id,
    libraryId: removal.libraryId,
    libraryName: removal.libraryName,
    status: removal.status,
  };
}

function requireEtag(response: Response): string {
  const etag = response.headers.get("ETag");
  if (!etag) {
    throw new Error("Required representation validator was not returned.");
  }
  return etag;
}
