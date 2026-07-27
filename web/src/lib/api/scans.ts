import { apiClient } from "./client";
import { createApiError } from "./errors";

export type ScanStatus =
  | "queued"
  | "running"
  | "succeeded"
  | "failed"
  | "cancelled"
  | "offline"
  | "interrupted";

export type ScanPhase =
  | "queued"
  | "checking_root"
  | "walking"
  | "indexing"
  | "finalizing"
  | "completed";

export interface ScanIssue {
  code:
    | "unreadable_directory"
    | "unsupported_file"
    | "invalid_media"
    | "media_probe_failed"
    | "symlink_skipped"
    | "maintained_directory_skipped"
    | "source_changed"
    | "io_error";
  count: number;
  sampleRelativePath: string | null;
}

export interface ScanRun {
  canCancel: boolean;
  cancelRequestedAt: string | null;
  createdAt: string;
  discoveredAssets: number;
  discoveredDirectories: number;
  errorCode: string | null;
  errorCount: number;
  finishedAt: string | null;
  id: string;
  issues: ScanIssue[];
  issuesTruncated: boolean;
  libraryId: string;
  phase: ScanPhase;
  processedAssets: number;
  progressRatio: number | null;
  skippedDirectories: number;
  skippedFiles: number;
  startedAt: string | null;
  status: ScanStatus;
  trigger: "library_created" | "startup" | "manual" | "scheduled";
}

export interface ScanPage {
  items: ScanRun[];
  nextCursor: string | null;
}

export async function listLibraryScans(
  libraryId: string,
): Promise<ScanPage> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/libraries/{libraryId}/scans",
      {
        params: {
          path: { libraryId },
          query: { limit: 20 },
        },
      },
    );
    if (data) {
      return {
        items: data.items.map(mapScan),
        nextCursor: data.nextCursor,
      };
    }
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getScan(scanId: string): Promise<ScanRun> {
  try {
    const { data, error, response } = await apiClient.GET(
      "/api/v1/scans/{scanId}",
      {
        params: {
          header: {},
          path: { scanId },
        },
      },
    );
    if (data) return mapScan(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function requestLibraryScan({
  csrfToken,
  libraryId,
}: {
  csrfToken: string;
  libraryId: string;
}): Promise<ScanRun> {
  try {
    const { data, error, response } = await apiClient.POST(
      "/api/v1/libraries/{libraryId}/scans",
      {
        headers: { "X-CSRF-Token": csrfToken },
        params: { path: { libraryId } },
      },
    );
    if (data) return mapScan(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function cancelScan({
  csrfToken,
  scanId,
}: {
  csrfToken: string;
  scanId: string;
}): Promise<ScanRun> {
  try {
    const { data, error, response } = await apiClient.POST(
      "/api/v1/scans/{scanId}/cancel",
      {
        headers: { "X-CSRF-Token": csrfToken },
        params: { path: { scanId } },
      },
    );
    if (data) return mapScan(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function mapScan(scan: {
  canCancel: boolean;
  cancelRequestedAt: string | null;
  createdAt: string;
  discoveredAssets: number;
  discoveredDirectories: number;
  errorCode: string | null;
  errorCount: number;
  finishedAt: string | null;
  id: string;
  issues: {
    code: ScanIssue["code"];
    count: number;
    sampleRelativePath: string | null;
  }[];
  issuesTruncated: boolean;
  libraryId: string;
  phase: ScanPhase;
  processedAssets: number;
  progressRatio: number | null;
  skippedDirectories: number;
  skippedFiles: number;
  startedAt: string | null;
  status: ScanStatus;
  trigger: ScanRun["trigger"];
}): ScanRun {
  return {
    canCancel: scan.canCancel,
    cancelRequestedAt: scan.cancelRequestedAt,
    createdAt: scan.createdAt,
    discoveredAssets: scan.discoveredAssets,
    discoveredDirectories: scan.discoveredDirectories,
    errorCode: scan.errorCode,
    errorCount: scan.errorCount,
    finishedAt: scan.finishedAt,
    id: scan.id,
    issues: scan.issues.map((issue) => ({ ...issue })),
    issuesTruncated: scan.issuesTruncated,
    libraryId: scan.libraryId,
    phase: scan.phase,
    processedAssets: scan.processedAssets,
    progressRatio: scan.progressRatio,
    skippedDirectories: scan.skippedDirectories,
    skippedFiles: scan.skippedFiles,
    startedAt: scan.startedAt,
    status: scan.status,
    trigger: scan.trigger,
  };
}
