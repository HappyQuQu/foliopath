import type {
  AssetKind,
  AssetSort,
  SortOrder,
} from "../../lib/api/catalog";
import { paths } from "../../routes/paths";

export type BrowseKind = "all" | "image" | "video";

export interface BrowseUrlState {
  allMedia?: true;
  kind: BrowseKind;
  order: SortOrder;
  recursive: boolean;
  sort: AssetSort;
}

export function defaultBrowseUrlState(recursive = false): BrowseUrlState {
  return recursive
    ? { kind: "all", order: "desc", recursive: true, sort: "modifiedAt" }
    : { kind: "all", order: "asc", recursive: false, sort: "name" };
}

export function parseBrowseUrlState(search: URLSearchParams): BrowseUrlState {
  const allMedia = search.get("view") === "all";
  const recursive = allMedia || search.get("recursive") === "1";
  const defaults = defaultBrowseUrlState(recursive);
  const kindValue = search.get("kind");
  const kind: BrowseKind =
    kindValue === "image" || kindValue === "video" ? kindValue : "all";
  const sortValue = search.get("sort");
  const sort: AssetSort =
    sortValue === "name" || sortValue === "modifiedAt"
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
    ...(allMedia ? { allMedia: true as const } : {}),
    kind,
    order,
    recursive,
    sort,
  };
}

export function serializeBrowseUrlState(state: BrowseUrlState): string {
  const search = new URLSearchParams();
  const defaults = defaultBrowseUrlState(state.recursive);
  if (state.recursive) search.set("recursive", "1");
  if (state.allMedia) search.set("view", "all");
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
