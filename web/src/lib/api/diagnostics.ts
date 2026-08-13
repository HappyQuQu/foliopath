import type { components } from "./generated/schema";

import { apiClient } from "./client";
import { createApiError } from "./errors";

export type MediaFailure = components["schemas"]["MediaFailure"];
export type MediaFailureVariant = MediaFailure["variant"];
export type MediaFailureErrorCode = MediaFailure["errorCode"];
export type MediaFailurePage = components["schemas"]["MediaFailurePage"];
export type MediaFailureDetail = components["schemas"]["MediaFailureDetail"];
export type MediaFailureRetrySummary =
  components["schemas"]["MediaFailureRetrySummary"];

export async function listMediaFailures({
  cursor,
  errorCode,
  libraryId,
  limit = 50,
  variant,
}: {
  cursor?: string | undefined;
  errorCode?: MediaFailureErrorCode | undefined;
  libraryId?: string | undefined;
  limit?: number | undefined;
  variant?: MediaFailureVariant | undefined;
} = {}): Promise<MediaFailurePage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/diagnostics/media-failures",
      {
        params: {
          query: {
            limit,
            ...(cursor ? { cursor } : {}),
            ...(errorCode ? { errorCode } : {}),
            ...(libraryId ? { libraryId } : {}),
            ...(variant ? { variant } : {}),
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

export async function getMediaFailure(jobId: string): Promise<MediaFailureDetail> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/diagnostics/media-failures/{jobId}",
      { params: { path: { jobId } } },
    );
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function repairLibraryMediaProcessing({
  csrfToken,
  libraryId,
  mode,
}: {
  csrfToken: string;
  libraryId: string;
  mode: "missing" | "all";
}): Promise<MediaFailureRetrySummary> {
  try {
    const { data, error, response } = await apiClient.POST(
      "/api/v1/libraries/{libraryId}/media-processing/repair",
      {
        headers: { "X-CSRF-Token": csrfToken },
        params: { path: { libraryId }, query: { mode } },
      },
    );
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}
