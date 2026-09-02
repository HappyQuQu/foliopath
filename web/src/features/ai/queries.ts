import {
  useMutation,
  useQueries,
  useQuery,
  useQueryClient,
  type Query,
} from "@tanstack/react-query";

import {
  activateAIModel,
  applyFaceReview,
  clearDerivedFaceData,
  clearManualFaceRelationships,
  cancelAIOperation,
  clearLibrarySemanticData,
  getAIOperation,
  getLibrarySemanticSettings,
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
  models: ["ai", "models"] as const,
  semantic: (libraryId: string) => ["ai", "semantic", libraryId] as const,
  faces: (libraryId: string) => ["ai", "faces", libraryId] as const,
  operation: (operationId: string) => ["ai", "operation", operationId] as const,
  suggestions: (libraryId: string, status: string) => ["ai", "suggestions", libraryId, status] as const,
  people: (query: string) => ["ai", "people", query] as const,
  person: (personId: string) => ["ai", "person", personId] as const,
  personAssets: (personId: string) => ["ai", "person-assets", personId] as const,
  clusters: (libraryId: string, kind: string) => ["ai", "clusters", libraryId, kind] as const,
  cluster: (libraryId: string, clusterId: string) => ["ai", "cluster", libraryId, clusterId] as const,
};

export function useAIModelsQuery() {
  return useQuery({ queryKey: aiKeys.models, queryFn: listAIModels });
}

export function useSemanticSettingsQueries(libraryIds: string[]) {
  return useQueries({
    queries: libraryIds.map((libraryId) => ({
      queryKey: aiKeys.semantic(libraryId),
      queryFn: () => getLibrarySemanticSettings(libraryId),
      refetchInterval: (query: Query<SemanticSettingsSnapshot>) =>
        query.state.data && ["building", "clearing"].includes(query.state.data.state)
          ? 1_500
          : false,
    })),
  });
}

export function useAIOperationQueries(operationIds: string[]) {
  return useQueries({
    queries: operationIds.map((operationId) => ({
      queryKey: aiKeys.operation(operationId),
      queryFn: () => getAIOperation(operationId),
      refetchInterval: (query: Query<AIOperationSnapshot>) =>
        query.state.data && operationIsActive(query.state.data) ? 1_500 : false,
    })),
  });
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
export const useInstallAIModelCandidateMutation = mutationWithInvalidation(installAIModelCandidate);
export const useActivateAIModelMutation = mutationWithInvalidation(activateAIModel);
export const useUpdateSemanticSettingsMutation = mutationWithInvalidation(updateLibrarySemanticSettings);
export const useRequestSemanticJobMutation = mutationWithInvalidation(requestLibrarySemanticJob);
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

export function useAITagSuggestionsQuery(libraryId: string, status: "pending" | "accepted" | "dismissed") {
  return useQuery({
    enabled: Boolean(libraryId),
    queryKey: aiKeys.suggestions(libraryId, status),
    queryFn: () => listAITagSuggestions({ libraryId, status }),
  });
}

export function usePeopleQuery(query: string) {
  return useQuery({ queryKey: aiKeys.people(query), queryFn: () => listPeople({ ...(query ? { query } : {}) }) });
}

export function usePersonQuery(personId: string) {
  return useQuery({ enabled: Boolean(personId), queryKey: aiKeys.person(personId), queryFn: () => getPerson(personId) });
}

export function usePersonAssetsQuery(personId: string) {
  return useQuery({ enabled: Boolean(personId), queryKey: aiKeys.personAssets(personId), queryFn: () => listPersonAssets({ personId }) });
}

export function useFaceClustersQuery(libraryId: string, kind: "core" | "edge") {
  return useQuery({
    enabled: Boolean(libraryId),
    queryKey: aiKeys.clusters(libraryId, kind),
    queryFn: () => listFaceClusters({ kind, libraryId }),
  });
}

export function useFaceClusterQuery(libraryId: string, clusterId: string) {
  return useQuery({
    enabled: Boolean(libraryId && clusterId),
    queryKey: aiKeys.cluster(libraryId, clusterId),
    queryFn: () => getFaceCluster({ clusterId, libraryId }),
  });
}

export function operationIsActive(operation: AIModelOperation): boolean {
  return ["queued", "running", "cancelling"].includes(operation.state);
}
