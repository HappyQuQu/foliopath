import type { components } from "./generated/schema";
import { apiClient } from "./client";
import { createApiError } from "./errors";

export type CacheCleanup = components["schemas"]["CacheCleanup"];
export type CacheSummary = components["schemas"]["CacheSummary"];

export async function getCacheSummary(): Promise<CacheSummary> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/cache");
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function startCacheCleanup(input: {
  csrfToken: string;
  idempotencyKey: string;
}): Promise<CacheCleanup> {
  try {
    const { data, error, response } = await apiClient.POST(
      "/api/v1/cache/cleanup",
      {
        headers: {
          "X-CSRF-Token": input.csrfToken,
        },
        params: {
          header: { "Idempotency-Key": input.idempotencyKey },
        },
      },
    );
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}
