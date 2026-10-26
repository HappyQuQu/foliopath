import type {
  AssetKind,
  AssetSort,
  SortOrder,
} from "../../lib/api/catalog";
import type { MediaSortPreference } from "../../lib/storage/preferences";
import { paths } from "../../routes/paths";

export type SearchScope = "library" | "directory" | "all";
export type SearchMode = "all" | "filename" | "semantic" | "ocr";
export type SearchKind = "all" | AssetKind;
export type SearchDate = "any" | "30d" | "year";

export interface SearchUrlState {
  dateBefore?: string;
  date: SearchDate;
  directoryId?: string;
  kind: SearchKind;
  mode: SearchMode;
  order: SortOrder;
  q: string;
  recursive: boolean;
  scope: SearchScope;
  sort: AssetSort;
}

export function defaultSearchUrlState(
  libraryId?: string,
  directoryId?: string,
  mediaSort: MediaSortPreference = "contextual",
): SearchUrlState {
  const [sort, order] =
    mediaSort === "contextual"
      ? (["modifiedAt", "desc"] as const)
      : (mediaSort.split(":") as [AssetSort, SortOrder]);
  return {
    date: "any",
    ...(directoryId ? { directoryId } : {}),
    kind: "all",
    mode: "all",
    order,
    q: "",
    recursive: false,
    scope: libraryId ? "library" : "all",
    sort,
  };
}

export function parseSearchUrlState(
  search: URLSearchParams,
  libraryId?: string,
  mediaSort: MediaSortPreference = "contextual",
): SearchUrlState {
  const defaults = defaultSearchUrlState(libraryId, undefined, mediaSort);
  const requestedScope = search.get("scope");
  const directoryId = search.get("directoryId")?.trim() || undefined;
  const scope: SearchScope =
    !libraryId
      ? "all"
      : requestedScope === "all"
        ? "all"
        : requestedScope === "directory" && directoryId
          ? "directory"
          : "library";
  const kindValue = search.get("kind");
  const requestedKind: SearchKind =
    kindValue === "image" ||
    kindValue === "animated" ||
    kindValue === "video"
      ? kindValue
      : "all";
  const mode: SearchMode = search.get("mode") === "ocr" ? "ocr" : search.get("mode") === "semantic" ? "semantic" : search.get("mode") === "filename" ? "filename" : "all";
  const kind: SearchKind = mode === "semantic"
    ? requestedKind === "video" ? "video" : "image"
    : requestedKind;
  const dateValue = search.get("date");
  const date: SearchDate =
    dateValue === "30d" || dateValue === "year" ? dateValue : "any";
  const sortValue = search.get("sort");
  const hasExplicitSort =
    sortValue === "name" || sortValue === "modifiedAt" || sortValue === "size";
  const sort: AssetSort = hasExplicitSort ? sortValue : defaults.sort;
  const orderValue = search.get("order");
  const order: SortOrder =
    orderValue === "asc" || orderValue === "desc"
      ? orderValue
      : hasExplicitSort
        ? sort === "name"
          ? "asc"
          : "desc"
        : defaults.order;

  return {
    ...(date !== "any" && mode !== "semantic" ? { dateBefore: validDateAnchor(search.get("before")) ?? new Date().toISOString() } : {}),
    date: mode === "semantic" ? "any" : date,
    ...(libraryId && directoryId ? { directoryId } : {}),
    kind,
    mode,
    order,
    q: search.get("q")?.trim() ?? "",
    recursive: scope === "directory" && search.get("recursive") === "1",
    scope,
    sort,
  };
}

export function serializeSearchUrlState(state: SearchUrlState): string {
  const search = new URLSearchParams();
  if (state.q) search.set("q", state.q);
  if (state.mode !== "all") search.set("mode", state.mode);
  if (state.scope !== "library") search.set("scope", state.scope);
  // Retain the originating directory while switching scope; only directory scope uses it for a request.
  if (state.directoryId) search.set("directoryId", state.directoryId);
  if (state.scope === "directory" && state.directoryId && state.recursive) search.set("recursive", "1");
  if (
    (state.mode !== "semantic" && state.kind !== "all") ||
    (state.mode === "semantic" && state.kind === "video")
  ) search.set("kind", state.kind);
  if (state.mode !== "semantic" && state.date !== "any") {
    search.set("date", state.date);
    search.set("before", validDateAnchor(state.dateBefore) ?? new Date().toISOString());
  }
  if (state.mode === "filename" && (state.sort !== "modifiedAt" || state.order !== "desc")) {
    search.set("sort", state.sort);
    search.set("order", state.order);
  }
  return search.toString();
}

function validDateAnchor(value: string | null | undefined): string | undefined {
  if (!value || !value.endsWith("Z")) return undefined;
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? date.toISOString() : undefined;
}

export function searchUrl(
  libraryId: string | undefined,
  state: SearchUrlState,
): string {
  const pathname = libraryId ? paths.librarySearch(libraryId) : paths.search;
  const search = serializeSearchUrlState(state);
  return search ? `${pathname}?${search}` : pathname;
}

export function kindsForSearch(kind: SearchKind): AssetKind[] | undefined {
  if (kind === "all") return undefined;
  return kind === "image" ? ["image"] : [kind];
}

export function modifiedRangeForSearch(
  date: SearchDate,
  now = new Date(),
): { modifiedFrom?: string; modifiedBefore?: string } {
  if (date === "any") return {};
  const end = new Date(now);
  const start = new Date(now);
  if (date === "30d") {
    start.setUTCDate(start.getUTCDate() - 30);
  } else {
    start.setUTCMonth(0, 1);
    start.setUTCHours(0, 0, 0, 0);
  }
  return {
    modifiedBefore: end.toISOString(),
    modifiedFrom: start.toISOString(),
  };
}
