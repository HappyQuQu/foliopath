import { useQuery, type QueryClient } from "@tanstack/react-query";
import { getDeletionSettings, getDeletionOperation, listDeletionOperations } from "../../lib/api/deletion";
import { refreshChangedCatalogQueries } from "../catalog-state";
import { catalogKeys } from "../browse";
import { curationKeys } from "../curation";
import { libraryKeys } from "../libraries";
import { mediaKeys } from "../media";
import { searchKeys } from "../search";
import { deletionOperationIsActive } from "./presentation";

export const deletionKeys = {
  all: ["deletion"] as const,
  settings: ["deletion", "settings"] as const,
  operations: (cursor: string) => ["deletion", "operations", cursor] as const,
  operation: (id: string, cursor: string) => ["deletion", "operation", id, cursor] as const,
};
export function useDeletionSettingsQuery(enabled = true) {
  return useQuery({ queryKey: deletionKeys.settings, queryFn: getDeletionSettings, enabled, staleTime: 15_000 });
}
export function useDeletionOperationsQuery(cursor: string) {
  return useQuery({ queryKey: deletionKeys.operations(cursor), queryFn: () => listDeletionOperations(cursor || undefined), refetchInterval: 2_000 });
}
export function useDeletionOperationQuery(id: string, cursor: string, pendingOnly = false) {
  return useQuery({ queryKey: deletionKeys.operation(id, cursor), queryFn: () => getDeletionOperation(id, cursor || undefined), enabled: Boolean(id), refetchInterval: query => pendingOnly ? (!query.state.data || deletionOperationIsActive(query.state.data.state) ? 1_000 : false) : 2_000 });
}
export async function refreshDeletionState(client: QueryClient) {
  await refreshChangedCatalogQueries(client, [deletionKeys.all, catalogKeys.all, curationKeys.all, libraryKeys.all, mediaKeys.all, searchKeys.all]);
}
