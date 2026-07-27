import type { components } from "./generated/schema";

import { apiClient } from "./client";
import { createApiError } from "./errors";

type NotReadyResponse = components["schemas"]["NotReadyResponse"];

export type ReadinessReason = NotReadyResponse["reasonCode"];

export type SystemReadiness =
  | {
      reasonCode: null;
      status: "ready";
    }
  | NotReadyResponse;

export async function getSystemReadiness(): Promise<SystemReadiness> {
  try {
    const { data, error, response } = await apiClient.GET("/health/ready");
    if (data) return { status: "ready", reasonCode: null };
    if (response.status === 503 && isNotReadyResponse(error)) return error;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function isNotReadyResponse(value: unknown): value is NotReadyResponse {
  if (typeof value !== "object" || value === null) return false;
  if (!("status" in value) || value.status !== "not_ready") return false;
  if (!("reasonCode" in value) || typeof value.reasonCode !== "string") return false;

  return [
    "application_data_unavailable",
    "migration_failed",
    "database_unavailable",
    "shutting_down",
  ].includes(value.reasonCode);
}
