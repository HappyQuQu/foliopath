import type { Asset } from "../api/catalog";

export interface ViewerSequenceItem {
  id: string;
  libraryId: string;
}

export interface ViewerLocationState {
  returnTo: string;
  sequence: ViewerSequenceItem[];
}

export interface ViewerReturnState {
  restoreFocusAssetId: string;
}

export function createViewerLocationState(
  assets: Asset[],
  returnTo: string,
): ViewerLocationState {
  return {
    returnTo,
    sequence: assets.map(({ id, libraryId }) => ({ id, libraryId })),
  };
}

export function readViewerLocationState(value: unknown): ViewerLocationState | undefined {
  if (!value || typeof value !== "object") return undefined;
  const candidate = value as Partial<ViewerLocationState>;
  if (
    typeof candidate.returnTo !== "string" ||
    !Array.isArray(candidate.sequence) ||
    !candidate.sequence.every(
      (item) =>
        Boolean(item) &&
        typeof item === "object" &&
        typeof item.id === "string" &&
        typeof item.libraryId === "string",
    )
  ) {
    return undefined;
  }
  return {
    returnTo: candidate.returnTo,
    sequence: candidate.sequence.map(({ id, libraryId }) => ({ id, libraryId })),
  };
}

export function safeViewerReturnPath(
  rawPath: string | null | undefined,
  libraryId: string,
): string {
  const fallback = `/libraries/${encodeURIComponent(libraryId)}/browse`;
  if (!rawPath?.startsWith("/") || rawPath.startsWith("//")) return fallback;

  try {
    const url = new URL(rawPath, "https://foliopath.invalid");
    const libraryPrefix = `/libraries/${encodeURIComponent(libraryId)}/`;
    const isLibrarySearch =
      url.pathname.startsWith("/libraries/") &&
      url.pathname.endsWith("/search") &&
      url.pathname.split("/").filter(Boolean).length === 3;
    const isCurrentLibraryBrowse =
      url.pathname === `${libraryPrefix}browse` ||
      url.pathname.startsWith(`${libraryPrefix}browse/`);
    const allowed =
      url.origin === "https://foliopath.invalid" &&
      (url.pathname === "/search" ||
        isLibrarySearch ||
        isCurrentLibraryBrowse);
    return allowed ? `${url.pathname}${url.search}${url.hash}` : fallback;
  } catch {
    return fallback;
  }
}

export function readViewerReturnState(value: unknown): ViewerReturnState | undefined {
  if (!value || typeof value !== "object") return undefined;
  const restoreFocusAssetId = (value as Partial<ViewerReturnState>)
    .restoreFocusAssetId;
  return typeof restoreFocusAssetId === "string"
    ? { restoreFocusAssetId }
    : undefined;
}
