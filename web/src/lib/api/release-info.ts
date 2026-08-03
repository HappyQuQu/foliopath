import type { components } from "./generated/schema";

import { apiClient } from "./client";
import { createApiError } from "./errors";

export type SystemStatus = components["schemas"]["SystemStatus"];
export type ReleaseInformation = components["schemas"]["ReleaseInformation"];

export async function getSystemStatus(): Promise<SystemStatus> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/status");
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getReleaseInformation(
  refresh = false,
): Promise<ReleaseInformation> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/releases", {
      params: { query: { refresh } },
    });
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}
