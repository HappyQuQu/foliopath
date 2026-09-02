import type { components } from "./generated/schema";
import { apiClient } from "./client";
import { createApiError } from "./errors";

export type AIModel = components["schemas"]["AIModel"];
export type AIModelCandidate = components["schemas"]["AIModelCandidate"];
export type AIModelCandidateScan = components["schemas"]["AIModelCandidateScan"];
export type AIModelOperation = components["schemas"]["AIModelOperation"];
export type AIModelPage = components["schemas"]["AIModelPage"];
export type SemanticLibrarySettings = components["schemas"]["SemanticLibrarySettings"];
export type AITagSuggestion = components["schemas"]["AITagSuggestion"];
export type AITagSuggestionPage = components["schemas"]["AITagSuggestionPage"];
export type AITagSuggestionReviewResponse = components["schemas"]["AITagSuggestionReviewResponse"];
export type AssetFace = components["schemas"]["AssetFace"];
export type FaceCluster = components["schemas"]["FaceCluster"];
export type FaceClusterDetailPage = components["schemas"]["FaceClusterDetailPage"];
export type FaceClusterPage = components["schemas"]["FaceClusterPage"];
export type FaceReviewRequest = components["schemas"]["FaceReviewRequest"];
export type FaceReviewResult = components["schemas"]["FaceReviewResult"];
export type FaceLibrarySettings = components["schemas"]["FaceLibrarySettings"];
export interface FaceSettingsSnapshot extends FaceLibrarySettings { etag: string }
export type Person = components["schemas"]["Person"];
export type PersonAssetPage = components["schemas"]["PersonAssetPage"];
export type PersonPage = components["schemas"]["PersonPage"];

export interface SemanticSettingsSnapshot extends SemanticLibrarySettings {
  etag: string;
}

export interface AIOperationSnapshot extends AIModelOperation {
  etag: string;
}

export async function listAIModels(): Promise<AIModelPage> {
  return unwrap(apiClient.GET("/api/v1/ai/models"));
}

export async function scanAIModelCandidates(csrfToken: string): Promise<AIModelCandidateScan> {
  return unwrap(apiClient.POST("/api/v1/ai/model-candidate-scans", {
    headers: { "X-CSRF-Token": csrfToken },
  }));
}

export async function installAIModelCandidate(input: {
  candidateId: string;
  csrfToken: string;
  idempotencyKey: string;
  storageMode: "managed" | "direct";
}): Promise<AIOperationSnapshot> {
  return unwrapWithEtag(apiClient.POST("/api/v1/ai/model-candidates/{candidateId}/install", {
    body: { storageMode: input.storageMode },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: {
      header: { "Idempotency-Key": input.idempotencyKey },
      path: { candidateId: input.candidateId },
    },
  }));
}

export async function activateAIModel(input: {
  csrfToken: string;
  idempotencyKey: string;
  model: AIModel;
}): Promise<AIOperationSnapshot> {
  return unwrapWithEtag(apiClient.POST("/api/v1/ai/models/{modelId}/activate", {
    headers: { "X-CSRF-Token": input.csrfToken },
    params: {
      header: {
        "Idempotency-Key": input.idempotencyKey,
        "If-Match": `"${input.model.id}-r${input.model.availabilityRevision}"`,
      },
      path: { modelId: input.model.id },
    },
  }));
}

export async function getAIOperation(operationId: string): Promise<AIOperationSnapshot> {
  return unwrapWithEtag(apiClient.GET("/api/v1/ai/operations/{operationId}", {
    params: { path: { operationId } },
  }));
}

export async function cancelAIOperation(input: {
  csrfToken: string;
  etag: string;
  operationId: string;
}): Promise<AIOperationSnapshot> {
  return unwrapWithEtag(apiClient.POST("/api/v1/ai/operations/{operationId}/cancel", {
    headers: { "X-CSRF-Token": input.csrfToken },
    params: {
      header: { "If-Match": input.etag },
      path: { operationId: input.operationId },
    },
  }));
}

export async function getLibrarySemanticSettings(
  libraryId: string,
): Promise<SemanticSettingsSnapshot> {
  return unwrapWithEtag(apiClient.GET("/api/v1/libraries/{libraryId}/ai/semantic", {
    params: { path: { libraryId } },
  }));
}

export async function updateLibrarySemanticSettings(input: {
  csrfToken: string;
  enabled: boolean;
  etag: string;
  libraryId: string;
}): Promise<SemanticSettingsSnapshot> {
  return unwrapWithEtag(apiClient.PATCH("/api/v1/libraries/{libraryId}/ai/semantic", {
    body: { enabled: input.enabled },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: {
      header: { "If-Match": input.etag },
      path: { libraryId: input.libraryId },
    },
  }));
}

export async function requestLibrarySemanticJob(input: {
  csrfToken: string;
  idempotencyKey: string;
  libraryId: string;
  mode: "missing" | "all";
}): Promise<AIOperationSnapshot> {
  return unwrapWithEtag(apiClient.POST("/api/v1/libraries/{libraryId}/ai/semantic/jobs", {
    body: { mode: input.mode },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: {
      header: { "Idempotency-Key": input.idempotencyKey },
      path: { libraryId: input.libraryId },
    },
  }));
}

export async function clearLibrarySemanticData(input: {
  csrfToken: string;
  etag: string;
  idempotencyKey: string;
  libraryId: string;
}): Promise<AIOperationSnapshot> {
  return unwrapWithEtag(apiClient.POST("/api/v1/libraries/{libraryId}/ai/semantic/clear", {
    body: { confirmation: "clear_semantic_data" },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: {
      header: {
        "Idempotency-Key": input.idempotencyKey,
        "If-Match": input.etag,
      },
      path: { libraryId: input.libraryId },
    },
  }));
}

export async function listAITagSuggestions(input: {
  cursor?: string;
  libraryId: string;
  status?: "pending" | "accepted" | "dismissed";
}): Promise<AITagSuggestionPage> {
  return unwrap(apiClient.GET("/api/v1/libraries/{libraryId}/ai/tag-suggestions", {
    params: {
      path: { libraryId: input.libraryId },
      query: {
        limit: 50,
        ...(input.cursor ? { cursor: input.cursor } : {}),
        ...(input.status ? { status: input.status } : {}),
      },
    },
  }));
}

export async function reviewAITagSuggestions(input: {
  action: "accept" | "dismiss";
  csrfToken: string;
  idempotencyKey: string;
  items: Array<{ curationRevision: number; suggestion: AITagSuggestion }>;
}): Promise<AITagSuggestionReviewResponse> {
  return unwrap(apiClient.POST("/api/v1/ai/tag-suggestion-reviews", {
    body: { items: input.items.map(({ curationRevision, suggestion }) => ({
      action: input.action,
      expectedCurationRevision: curationRevision,
      expectedSuggestionRevision: suggestion.revision,
      suggestionId: suggestion.id,
    })) },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "Idempotency-Key": input.idempotencyKey } },
  }));
}

export async function listPeople(input: { cursor?: string; query?: string }): Promise<PersonPage> {
  return unwrap(apiClient.GET("/api/v1/people", {
    params: { query: {
      limit: 50,
      ...(input.cursor ? { cursor: input.cursor } : {}),
      ...(input.query ? { query: input.query } : {}),
    } },
  }));
}

export async function getPerson(personId: string): Promise<Person & { etag: string }> {
  return unwrapWithEtag(apiClient.GET("/api/v1/people/{personId}", {
    params: { path: { personId } },
  }));
}

export async function listPersonAssets(input: { cursor?: string; personId: string }): Promise<PersonAssetPage> {
  return unwrap(apiClient.GET("/api/v1/people/{personId}/assets", {
    params: { path: { personId: input.personId }, query: {
      limit: 50,
      ...(input.cursor ? { cursor: input.cursor } : {}),
    } },
  }));
}

export async function createPerson(input: {
  csrfToken: string;
  idempotencyKey: string;
  name: string;
  sourceCluster?: FaceCluster;
}): Promise<Person & { etag: string }> {
  return unwrapWithEtag(apiClient.POST("/api/v1/people", {
    body: {
      name: input.name,
      ...(input.sourceCluster ? {
        expectedClusterRevision: input.sourceCluster.revision,
        sourceClusterId: input.sourceCluster.id,
      } : {}),
    },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "Idempotency-Key": input.idempotencyKey } },
  }));
}

export async function renamePerson(input: {
  csrfToken: string;
  etag: string;
  name: string;
  personId: string;
}): Promise<Person & { etag: string }> {
  return unwrapWithEtag(apiClient.PATCH("/api/v1/people/{personId}", {
    body: { name: input.name },
    headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "If-Match": input.etag }, path: { personId: input.personId } },
  }));
}

export async function listFaceClusters(input: {
  cursor?: string;
  kind?: "core" | "edge";
  libraryId: string;
}): Promise<FaceClusterPage> {
  return unwrap(apiClient.GET("/api/v1/libraries/{libraryId}/ai/face-clusters", {
    params: {
      path: { libraryId: input.libraryId },
      query: {
        limit: 50,
        ...(input.cursor ? { cursor: input.cursor } : {}),
        ...(input.kind ? { kind: input.kind } : {}),
      },
    },
  }));
}

export async function getFaceCluster(input: {
  clusterId: string;
  cursor?: string;
  libraryId: string;
}): Promise<FaceClusterDetailPage> {
  return unwrap(apiClient.GET("/api/v1/libraries/{libraryId}/ai/face-clusters/{clusterId}", {
    params: {
      path: { clusterId: input.clusterId, libraryId: input.libraryId },
      query: { limit: 50, ...(input.cursor ? { cursor: input.cursor } : {}) },
    },
  }));
}

export async function listAssetFaces(assetId: string) {
  return unwrap(apiClient.GET("/api/v1/assets/{assetId}/faces", {
    params: { path: { assetId } },
  }));
}

export async function applyFaceReview(input: {
  csrfToken: string;
  idempotencyKey: string;
  review: FaceReviewRequest;
}): Promise<FaceReviewResult> {
  return unwrap(apiClient.POST("/api/v1/ai/face-reviews", {
    body: input.review,
    headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "Idempotency-Key": input.idempotencyKey } },
  }));
}

export async function getLibraryFaceSettings(libraryId: string): Promise<FaceSettingsSnapshot> {
  return unwrapWithEtag(apiClient.GET("/api/v1/libraries/{libraryId}/ai/faces", {
    params: { path: { libraryId } },
  }));
}

export async function updateLibraryFaceSettings(input: { csrfToken: string; enabled: boolean; etag: string; libraryId: string }): Promise<FaceSettingsSnapshot> {
  return unwrapWithEtag(apiClient.PATCH("/api/v1/libraries/{libraryId}/ai/faces", {
    body: { enabled: input.enabled }, headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "If-Match": input.etag }, path: { libraryId: input.libraryId } },
  }));
}

export async function requestFaceAnalysisJob(input: { csrfToken: string; idempotencyKey: string; libraryId: string; mode: "missing" | "all" }): Promise<AIModelOperation> {
  return unwrap(apiClient.POST("/api/v1/libraries/{libraryId}/ai/faces/jobs", {
    body: { mode: input.mode }, headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "Idempotency-Key": input.idempotencyKey }, path: { libraryId: input.libraryId } },
  }));
}

export async function clearDerivedFaceData(input: { csrfToken: string; etag: string; idempotencyKey: string; libraryId: string }): Promise<AIModelOperation> {
  return unwrap(apiClient.POST("/api/v1/libraries/{libraryId}/ai/faces/derived-clear", {
    body: { confirmation: "clear_derived_face_data" }, headers: { "X-CSRF-Token": input.csrfToken },
    params: { header: { "Idempotency-Key": input.idempotencyKey, "If-Match": input.etag }, path: { libraryId: input.libraryId } },
  }));
}

export async function clearManualFaceRelationships(input: { assignmentCount: number; constraintCount: number; csrfToken: string; etag: string; idempotencyKey: string; libraryId: string; personCount: number }): Promise<AIModelOperation> {
  return unwrap(apiClient.POST("/api/v1/libraries/{libraryId}/ai/faces/manual-clear", {
    body: { confirmation: "clear_manual_face_relationships", expectedAssignmentCount: input.assignmentCount, expectedConstraintCount: input.constraintCount, expectedPersonCount: input.personCount },
    headers: { "X-CSRF-Token": input.csrfToken }, params: { header: { "Idempotency-Key": input.idempotencyKey, "If-Match": input.etag }, path: { libraryId: input.libraryId } },
  }));
}

async function unwrap<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  try {
    const { data, error, response } = await request;
    if (data !== undefined) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

async function unwrapWithEtag<T extends object>(
  request: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T & { etag: string }> {
  try {
    const { data, error, response } = await request;
    if (data !== undefined) return { ...data, etag: requireEtag(response) };
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function requireEtag(response: Response): string {
  const etag = response.headers.get("ETag");
  if (!etag) throw new Error("Required representation validator was not returned.");
  return etag;
}
