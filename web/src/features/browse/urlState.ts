import type { AssetSort, SortOrder } from "../../lib/api/catalog";
import { paths } from "../../routes/paths";

export interface BrowseUrlState {
  order: SortOrder;
  recursive: boolean;
  sort: AssetSort;
}

export function defaultBrowseUrlState(recursive = false): BrowseUrlState {
  return recursive
    ? { order: "desc", recursive: true, sort: "modifiedAt" }
    : { order: "asc", recursive: false, sort: "name" };
}

export function parseBrowseUrlState(search: URLSearchParams): BrowseUrlState {
  const recursive = search.get("recursive") === "1";
  const defaults = defaultBrowseUrlState(recursive);
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
  return { order, recursive, sort };
}

export function serializeBrowseUrlState(state: BrowseUrlState): string {
  const search = new URLSearchParams();
  const defaults = defaultBrowseUrlState(state.recursive);
  if (state.recursive) search.set("recursive", "1");
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
  const search = serializeBrowseUrlState(state);
  const pathname = paths.browse(libraryId, directoryId);
  return search ? `${pathname}?${search}` : pathname;
}
