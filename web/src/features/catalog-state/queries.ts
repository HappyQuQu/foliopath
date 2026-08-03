import {
  type InfiniteData,
  type QueryClient,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { getCatalogState } from "../../lib/api/catalog";

export const catalogStateKey = ["catalog-state"] as const;
export const catalogStateRefreshMs = 5_000;

export interface CatalogStateSnapshot {
  contentRevision: number;
  etag: string;
}

export function useCatalogStateQuery(enabled: boolean) {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: catalogStateKey,
    queryFn: async (): Promise<CatalogStateSnapshot> => {
      const previous = queryClient.getQueryData<CatalogStateSnapshot>(catalogStateKey);
      const next = await getCatalogState(previous?.etag);
      if (next.contentRevision === null) {
        if (!previous) throw new Error("Catalog state validator has no local revision.");
        return previous;
      }
      return {
        contentRevision: next.contentRevision,
        etag: next.etag,
      };
    },
    enabled,
    refetchInterval: () =>
      enabled && document.visibilityState === "visible"
        ? catalogStateRefreshMs
        : false,
    refetchIntervalInBackground: false,
    staleTime: 0,
  });
}

export async function refreshChangedCatalogQueries(
  queryClient: QueryClient,
  queryKeys: readonly (readonly unknown[])[],
) {
  for (const queryKey of queryKeys) {
    queryClient.setQueriesData<InfiniteData<unknown, unknown>>(
      { queryKey },
      (current) =>
        current && current.pages.length > 1
          ? {
              pageParams: current.pageParams.slice(0, 1),
              pages: current.pages.slice(0, 1),
            }
          : current,
    );
  }
  await Promise.all(
    queryKeys.map((queryKey) =>
      queryClient.invalidateQueries({ queryKey, refetchType: "active" }),
    ),
  );
}
