import {
  useMutation,
  useQueries,
  useQuery,
  useQueryClient,
  type Query,
} from "@tanstack/react-query";
import { useEffect } from "react";

import {
  activateAIModel,
  getAIModelProxy, saveAIModelProxy, testAIModelProxy,
  downloadAIModel,
  listAIModelDownloads,
  applyFaceReview,
  clearDerivedFaceData,
  clearManualFaceRelationships,
  cancelAIOperation,
  clearLibrarySemanticData,
  getAIOperation,
  getLibrarySemanticSettings,
  getLibraryVideoSemanticStatus,
  getFaceCluster,
  getLibraryFaceSettings,
  getPerson,
  installAIModelCandidate,
  listAIModels,
  listAITagSuggestions,
  listFaceClusters,
  listPeople,
  listPersonAssets,
  requestLibrarySemanticJob,
  requestLibraryVideoSemanticJob,
  requestFaceAnalysisJob,
  scanAIModelCandidates,
  reviewAITagSuggestions,
  createPerson,
  renamePerson,
  updateLibraryFaceSettings,
  updateLibrarySemanticSettings,
  type AIModelOperation,
  type AIOperationSnapshot,
  type SemanticSettingsSnapshot,
} from "../../lib/api/ai";

export const aiKeys = {
  all: ["ai"] as const,
  videoStatus: (libraryId: string) => ["ai", "video-status", libraryId] as const,
  models: ["ai", "models"] as const,
  downloads: ["ai", "downloads"] as const,
  proxy: ["ai", "model-proxy"] as const,
  semantic: (libraryId: string) => ["ai", "semantic", libraryId] as const,
  faces: (libraryId: string) => ["ai", "faces", libraryId] as const,
  operation: (operationId: string) => ["ai", "operation", operationId] as const,
  suggestions: (libraryId: string, status: string, cursor = "") => ["ai", "suggestions", libraryId, status, cursor] as const,
  people: (query: string, cursor = "") => ["ai", "people", query, cursor] as const,
  person: (personId: string) => ["ai", "person", personId] as const,
  personAssets: (personId: string, cursor = "") => ["ai", "person-assets", personId, cursor] as const,
  clusters: (libraryId: string, kind: string, cursor = "") => ["ai", "clusters", libraryId, kind, cursor] as const,
  cluster: (libraryId: string, clusterId: string, cursor = "") => ["ai", "cluster", libraryId, clusterId, cursor] as const,
};

export function useAIModelsQuery() {
  return useQuery({ queryKey: aiKeys.models, queryFn: listAIModels, refetchInterval: 10_000 });
}

export function useVideoSemanticStatusQuery(libraryId: string) {
  return useQuery({
    queryKey: aiKeys.videoStatus(libraryId),
    queryFn: () => getLibraryVideoSemanticStatus(libraryId),
    refetchInterval: 10_000,
  });
}

export function useAIModelProxyQuery() {
  return useQuery({ queryKey: aiKeys.proxy, queryFn: getAIModelProxy, staleTime: Infinity, refetchOnWindowFocus: false });
}
export function useSaveAIModelProxyMutation() {
  const client = useQueryClient();
  return useMutation({ mutationFn: saveAIModelProxy, onSuccess: (value) => client.setQueryData(aiKeys.proxy, value) });
}
export function useTestAIModelProxyMutation() { return useMutation({ mutationFn: testAIModelProxy }); }

export function useAIModelDownloadsQuery() {
  return useQuery({ queryKey: aiKeys.downloads, queryFn: listAIModelDownloads, refetchInterval: (query) =>
    query.state.data?.items.some((item) => item.operation && operationIsActive(item.operation)) ? 1_500 : 10_000 });
}

export function useSemanticSettingsQueries(libraryIds: string[]) {
  return useQueries({
    queries: libraryIds.map((libraryId) => ({
      queryKey: aiKeys.semantic(libraryId),
      queryFn: () => getLibrarySemanticSettings(libraryId),
      refetchInterval: (query: Query<SemanticSettingsSnapshot>) =>
        query.state.data && ["building", "clearing"].includes(query.state.data.state)
          ? 1_500
          : 10_000,
    })),
  });
}

export function useAIOperationQueries(operationIds: string[]) {
  const client = useQueryClient();
  const queries = useQueries({
    queries: operationIds.map((operationId) => ({
      queryKey: aiKeys.operation(operationId),
      queryFn: () => getAIOperation(operationId),
      refetchInterval: (query: Query<AIOperationSnapshot>) =>
        query.state.data && operationIsActive(query.state.data) ? 1_500 : false,
    })),
  });
  const finished = queries.map((query) => query.data && !operationIsActive(query.data) ? `${query.data.id}:${query.data.revision}` : "").filter(Boolean).join(",");
  useEffect(() => {
    if (!finished) return;
    void client.invalidateQueries({ queryKey: aiKeys.models });
    void client.invalidateQueries({ queryKey: aiKeys.downloads });
    void client.invalidateQueries({ queryKey: ["ai", "semantic"] });
    void client.invalidateQueries({ queryKey: ["ai", "video-status"] });
    void client.invalidateQueries({ queryKey: ["ai", "faces"] });
  }, [client, finished]);
  return queries;
}

function mutationWithInvalidation<TInput, TOutput>(
  mutationFn: (input: TInput) => Promise<TOutput>,
) {
  return function useAIActionMutation() {
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn,
      onSuccess: () => queryClient.invalidateQueries({ queryKey: aiKeys.all }),
    });
  };
}

export const useScanAIModelCandidatesMutation = mutationWithInvalidation(scanAIModelCandidates);
export const useDownloadAIModelMutation = mutationWithInvalidation(downloadAIModel);
export const useInstallAIModelCandidateMutation = mutationWithInvalidation(installAIModelCandidate);
export const useActivateAIModelMutation = mutationWithInvalidation(activateAIModel);
export const useUpdateSemanticSettingsMutation = mutationWithInvalidation(updateLibrarySemanticSettings);
export const useRequestSemanticJobMutation = mutationWithInvalidation(requestLibrarySemanticJob);
export const useRequestVideoSemanticJobMutation = mutationWithInvalidation(requestLibraryVideoSemanticJob);
export const useClearSemanticDataMutation = mutationWithInvalidation(clearLibrarySemanticData);
export const useCancelAIOperationMutation = mutationWithInvalidation(cancelAIOperation);
export const useReviewAITagSuggestionsMutation = mutationWithInvalidation(reviewAITagSuggestions);
export const useCreatePersonMutation = mutationWithInvalidation(createPerson);
export const useRenamePersonMutation = mutationWithInvalidation(renamePerson);
export const useApplyFaceReviewMutation = mutationWithInvalidation(applyFaceReview);
export const useUpdateFaceSettingsMutation = mutationWithInvalidation(updateLibraryFaceSettings);
export const useRequestFaceJobMutation = mutationWithInvalidation(requestFaceAnalysisJob);
export const useClearDerivedFaceDataMutation = mutationWithInvalidation(clearDerivedFaceData);
export const useClearManualFaceRelationshipsMutation = mutationWithInvalidation(clearManualFaceRelationships);

export function useFaceSettingsQueries(libraryIds: string[]) {
  return useQueries({ queries: libraryIds.map((libraryId) => ({ queryKey: aiKeys.faces(libraryId), queryFn: () => getLibraryFaceSettings(libraryId) })) });
}

export function useAITagSuggestionsQuery(libraryId: string, status: "pending" | "accepted" | "dismissed", cursor = "") {
  return useQuery({
    enabled: Boolean(libraryId),
    queryKey: aiKeys.suggestions(libraryId, status, cursor),
    queryFn: () => listAITagSuggestions({ libraryId, status, cursor }),
  });
}

export function usePeopleQuery(query: string, cursor = "") {
  return useQuery({ queryKey: aiKeys.people(query, cursor), queryFn: () => listPeople({ cursor, ...(query ? { query } : {}) }) });
}

export function usePersonQuery(personId: string) {
  return useQuery({ enabled: Boolean(personId), queryKey: aiKeys.person(personId), queryFn: () => getPerson(personId) });
}

export function usePersonAssetsQuery(personId: string, cursor = "") {
  return useQuery({ enabled: Boolean(personId), queryKey: aiKeys.personAssets(personId, cursor), queryFn: () => listPersonAssets({ personId, cursor }) });
}

export function useFaceClustersQuery(libraryId: string, kind: "core" | "edge", cursor = "") {
  return useQuery({
    enabled: Boolean(libraryId),
    queryKey: aiKeys.clusters(libraryId, kind, cursor),
    queryFn: () => listFaceClusters({ kind, libraryId, cursor }),
  });
}

export function useFaceClusterQuery(libraryId: string, clusterId: string, cursor = "") {
  return useQuery({
    enabled: Boolean(libraryId && clusterId),
    queryKey: aiKeys.cluster(libraryId, clusterId, cursor),
    queryFn: () => getFaceCluster({ clusterId, libraryId, cursor }),
  });
}

export function operationIsActive(operation: AIModelOperation): boolean {
  return ["queued", "running", "cancelling"].includes(operation.state);
}
