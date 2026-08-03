import { describe, expect, it } from "vitest";

import type { LibrarySummary } from "../../lib/api/libraries";
import type { ScanRun } from "../../lib/api/scans";
import { mergeLibraryScanRecords } from "./library-record-queries";

function scan(id: string, libraryId: string, createdAt: string): ScanRun {
  return {
    canCancel: false,
    cancelRequestedAt: null,
    createdAt,
    discoveredAssets: 2,
    discoveredDirectories: 1,
    errorCode: null,
    errorCount: 0,
    finishedAt: createdAt,
    id,
    issues: [],
    issuesTruncated: false,
    libraryId,
    phase: "completed",
    processedAssets: 2,
    progressRatio: 1,
    skippedDirectories: 0,
    skippedFiles: 0,
    startedAt: createdAt,
    status: "succeeded",
    trigger: "manual",
  };
}

const libraries: LibrarySummary[] = [
  {
    assetCount: 2,
    directoryCount: 1,
    displayPath: "first",
    id: "library-1",
    lastSuccessfulScanAt: null,
    latestScanId: null,
    name: "First",
    status: "ready",
  },
  {
    assetCount: 2,
    directoryCount: 1,
    displayPath: "second",
    id: "library-2",
    lastSuccessfulScanAt: null,
    latestScanId: null,
    name: "Second",
    status: "ready",
  },
];

describe("mergeLibraryScanRecords", () => {
  it("labels scan history with its library and sorts newest first", () => {
    const result = mergeLibraryScanRecords(libraries, [
      { items: [scan("scan-1", "library-1", "2026-08-01T00:00:00Z")] },
      { items: [scan("scan-2", "library-2", "2026-08-02T00:00:00Z")] },
    ]);

    expect(result.map((item) => [item.libraryName, item.scan.id])).toEqual([
      ["Second", "scan-2"],
      ["First", "scan-1"],
    ]);
  });
});
