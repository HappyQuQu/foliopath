import { apiClient } from "./client";
import { createApiError } from "./errors";
import type { components } from "./generated/schema";

type Schema = components["schemas"];
export type DeletionSettings = Schema["DeletionSettings"];
export type DeletionSettingsUpdate = Schema["DeletionSettingsUpdate"];
export type DeletionTarget = Schema["DeletionTarget"];
export type DeletionPreflight = Schema["DeletionPreflight"];
export type DeletionOperation = Schema["DeletionOperation"];

export async function getDeletionSettings() {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/deletion/settings");
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
export async function updateDeletionSettings(body: DeletionSettingsUpdate, revision: number, csrfToken: string) {
  try {
    const { data, error, response } = await apiClient.PATCH("/api/v1/deletion/settings", {
      body, headers: { "X-CSRF-Token": csrfToken }, params: { header: { "If-Match": `"deletion-settings-r${revision}"` } },
    });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
export async function preflightDeletion(targets: DeletionTarget[], csrfToken: string) {
  try {
    const { data, error, response } = await apiClient.POST("/api/v1/deletion/preflight", { body: { targets }, headers: { "X-CSRF-Token": csrfToken } });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
export async function submitDeletion(preflight: DeletionPreflight, csrfToken: string, key: string) {
  try {
    const { data, error, response } = await apiClient.POST("/api/v1/deletion/operations", {
      body: { preflightId: preflight.id!, confirmation: "permanently_delete" },
      headers: { "X-CSRF-Token": csrfToken }, params: { header: { "Idempotency-Key": key } },
    });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
export async function listDeletionOperations(cursor?: string) {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/deletion/operations", { params: { query: { limit: 25, ...(cursor ? { cursor } : {}) } } });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
export async function getDeletionOperation(operationId: string, cursor?: string) {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/deletion/operations/{operationId}", { params: { path: { operationId }, query: { limit: 25, ...(cursor ? { cursor } : {}) } } });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
export async function changeDeletionOperation(operation: DeletionOperation, action: "stop" | "resume", csrfToken: string, key: string) {
  try {
    const options = { params: { path: { operationId: operation.id }, header: { "If-Match": `"${operation.id}-r${operation.revision}"`, "Idempotency-Key": key } }, headers: { "X-CSRF-Token": csrfToken } };
    const { data, error, response } = action === "stop"
      ? await apiClient.POST("/api/v1/deletion/operations/{operationId}/stop", options)
      : await apiClient.POST("/api/v1/deletion/operations/{operationId}/resume", { ...options, body: { confirmation: "continue" } });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) { throw createApiError(error); }
}
