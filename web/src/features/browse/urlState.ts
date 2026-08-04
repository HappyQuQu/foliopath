import type { AssetKind, AssetSort, SortOrder } from "../../lib/api/catalog";
import type { MediaSortPreference } from "../../lib/storage/preferences";
import { paths } from "../../routes/paths";

export type BrowseKind = "all" | "image" | "video";

export interface BrowseUrlState {
  allMedia?: true;
  kind: BrowseKind;
  order: SortOrder;
  q: string;
  recursive: boolean;
  sort: AssetSort;
}

export function defaultBrowseUrlState(
  recursive = false,
  q = "",
  mediaSort: MediaSortPreference = "contextual",
): BrowseUrlState {
  const contextual: BrowseUrlState = recursive || q
    ? { kind: "all", order: "desc", q, recursive, sort: "modifiedAt" }
    : { kind: "all", order: "asc", q, recursive: false, sort: "name" };
  if (mediaSort === "contextual") return contextual;
  const [sort, order] = mediaSort.split(":") as [AssetSort, SortOrder];
  return { ...contextual, order, sort };
}

export function parseBrowseUrlState(
  search: URLSearchParams,
  mediaSort: MediaSortPreference = "contextual",
): BrowseUrlState {
  const allMedia = search.get("view") === "all";
  const recursive = allMedia || search.get("recursive") === "1";
  const q = search.get("q")?.trim() ?? "";
  const defaults = defaultBrowseUrlState(recursive, q, mediaSort);
  const kindValue = search.get("kind");
  const kind: BrowseKind =
    kindValue === "image" || kindValue === "video" ? kindValue : "all";
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
    ...(allMedia ? { allMedia: true as const } : {}),
    kind,
    order,
    q,
    recursive,
    sort,
  };
}

export function serializeBrowseUrlState(state: BrowseUrlState): string {
  const search = new URLSearchParams();
  const defaults = defaultBrowseUrlState(state.recursive, state.q);
  if (state.recursive) search.set("recursive", "1");
  if (state.allMedia) search.set("view", "all");
  if (state.q) search.set("q", state.q);
  if (state.kind !== "all") search.set("kind", state.kind);
  if (state.sort !== defaults.sort || state.order !== defaults.order) {
    search.set("sort", state.sort);
    search.set("order", state.order);
  }
  return search.toString();
}

export function browseUrl(
  libraryId: string,
  directoryId: string | undefined,
  state: BrowseUrlState,
): string {
  const search = serializeBrowseUrlState(
    directoryId && state.allMedia
      ? {
          order: state.order,
          recursive: state.recursive,
          sort: state.sort,
          kind: state.kind,
          q: state.q,
        }
      : state,
  );
  const pathname = paths.browse(libraryId, directoryId);
  return search ? `${pathname}?${search}` : pathname;
}

export function kindsForBrowse(kind: BrowseKind): AssetKind[] | undefined {
  if (kind === "all") return undefined;
  return kind === "image" ? ["image", "animated"] : ["video"];
}
