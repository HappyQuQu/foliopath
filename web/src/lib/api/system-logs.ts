import type { components } from "./generated/schema";

import { apiClient } from "./client";
import { createApiError } from "./errors";

export type SystemEvent = components["schemas"]["SystemEvent"];
export type SystemEventLevel = components["schemas"]["SystemEventLevel"];
export type SystemEventPage = components["schemas"]["SystemEventPage"];

export async function listSystemEvents({
  cursor,
  level,
  module,
  limit = 50,
}: {
  cursor?: string | undefined;
  level?: SystemEventLevel | undefined;
  module?: string | undefined;
  limit?: number | undefined;
} = {}): Promise<SystemEventPage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/system-logs",
      {
        params: {
          query: {
            limit,
            ...(cursor ? { cursor } : {}),
            ...(level ? { level } : {}),
            ...(module ? { module } : {}),
          },
        },
      },
    );
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}
