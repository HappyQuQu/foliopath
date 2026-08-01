import type { components } from "./generated/schema";
import { apiClient } from "./client";
import { createApiError } from "./errors";

export type MediaJobProgress = components["schemas"]["MediaJobProgress"];
export type MediaProcessingProgress =
  components["schemas"]["MediaProcessingProgress"];

export async function getLibraryMediaProcessingProgress(
  libraryId: string,
): Promise<MediaProcessingProgress> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/libraries/{libraryId}/media-processing",
      { params: { path: { libraryId } } },
    );
    if (data) return data;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}
