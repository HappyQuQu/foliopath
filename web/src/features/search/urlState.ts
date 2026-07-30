import type {
  AssetKind,
  AssetSort,
  SortOrder,
} from "../../lib/api/catalog";
import { paths } from "../../routes/paths";

export type SearchScope = "library" | "directory" | "all";
export type SearchKind = "all" | AssetKind;
export type SearchDate = "any" | "30d" | "year";

export interface SearchUrlState {
  date: SearchDate;
  directoryId?: string;
  kind: SearchKind;
  order: SortOrder;
  q: string;
  recursive: boolean;
  scope: SearchScope;
  sort: AssetSort;
}

export function defaultSearchUrlState(
  libraryId?: string,
  directoryId?: string,
): SearchUrlState {
  return {
    date: "any",
    ...(directoryId ? { directoryId } : {}),
    kind: "all",
    order: "desc",
    q: "",
    recursive: false,
    scope: libraryId ? "library" : "all",
    sort: "modifiedAt",
  };
}

export function parseSearchUrlState(
  search: URLSearchParams,
  libraryId?: string,
): SearchUrlState {
  const defaults = defaultSearchUrlState(libraryId);
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
  const kind: SearchKind =
    kindValue === "image" ||
    kindValue === "animated" ||
    kindValue === "video"
      ? kindValue
      : "all";
  const dateValue = search.get("date");
  const date: SearchDate =
    dateValue === "30d" || dateValue === "year" ? dateValue : "any";
  const sortValue = search.get("sort");
  const sort: AssetSort =
    sortValue === "name" || sortValue === "modifiedAt" || sortValue === "size"
      ? sortValue
      : defaults.sort;
  const orderValue = search.get("order");
  const order: SortOrder =
    orderValue === "asc" || orderValue === "desc"
      ? orderValue
      : sort === "name"
        ? "asc"
        : "desc";

  return {
    date,
    ...(scope === "directory" && directoryId ? { directoryId } : {}),
    kind,
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
  if (state.scope !== "library") search.set("scope", state.scope);
  if (state.scope === "directory" && state.directoryId) {
    search.set("directoryId", state.directoryId);
    if (state.recursive) search.set("recursive", "1");
  }
  if (state.kind !== "all") search.set("kind", state.kind);
  if (state.date !== "any") search.set("date", state.date);
  if (state.sort !== "modifiedAt" || state.order !== "desc") {
    search.set("sort", state.sort);
    search.set("order", state.order);
  }
  return search.toString();
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
