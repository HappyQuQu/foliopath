import { apiClient } from "./client";
import { createApiError } from "./errors";

export interface ApplicationSettings {
  etag: string;
  language: "browser" | "zh-CN" | "en";
  scheduledScanIntervalHours: number | null;
  backgroundConcurrency: number;
  contentReadConcurrency: number;
  thumbnailCacheQuotaBytes: number;
  updatedAt: string;
}

export interface SettingsUpdate {
  csrfToken: string;
  etag: string;
  scheduledScanIntervalHours: number | null;
  backgroundConcurrency: number;
  contentReadConcurrency: number;
  thumbnailCacheQuotaBytes: number;
}

export async function getSettings(): Promise<ApplicationSettings> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/settings");
    if (data) return mapSettings(data, requireEtag(response));
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function updateSettings(
  input: SettingsUpdate,
): Promise<ApplicationSettings> {
  try {
    const { data, error, response } = await apiClient.PATCH("/api/v1/settings", {
      body: {
        scheduledScanIntervalHours: input.scheduledScanIntervalHours,
        backgroundConcurrency: input.backgroundConcurrency,
        contentReadConcurrency: input.contentReadConcurrency,
        thumbnailCacheQuotaBytes: input.thumbnailCacheQuotaBytes,
      },
      headers: { "X-CSRF-Token": input.csrfToken },
      params: {
        header: { "If-Match": input.etag },
      },
    });
    if (data) return mapSettings(data, requireEtag(response));
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function mapSettings(
  settings: {
    language: ApplicationSettings["language"];
    scheduledScanIntervalHours: number | null;
    backgroundConcurrency: number;
    contentReadConcurrency: number;
    thumbnailCacheQuotaBytes: number;
    updatedAt: string;
  },
  etag: string,
): ApplicationSettings {
  return {
    etag,
    language: settings.language,
    scheduledScanIntervalHours: settings.scheduledScanIntervalHours,
    backgroundConcurrency: settings.backgroundConcurrency,
    contentReadConcurrency: settings.contentReadConcurrency,
    thumbnailCacheQuotaBytes: settings.thumbnailCacheQuotaBytes,
    updatedAt: settings.updatedAt,
  };
}

function requireEtag(response: Response): string {
  const etag = response.headers.get("ETag");
  if (!etag) throw new Error("Required representation validator was not returned.");
  return etag;
}
