import { useQueries } from "@tanstack/react-query";

import type { LibrarySummary } from "../../lib/api/libraries";
import { listLibraryScans, type ScanRun } from "../../lib/api/scans";

export interface LibraryScanRecord {
  id: string;
  libraryId: string;
  libraryName: string;
  scan: ScanRun;
}

export function useLibraryScanRecordQueries(libraries: LibrarySummary[]) {
  return useQueries({
    queries: libraries.map((library) => ({
      queryKey: ["libraries", "scan-records", library.id] as const,
      queryFn: () => listLibraryScans(library.id),
      staleTime: 5_000,
    })),
  });
}

export function mergeLibraryScanRecords(
  libraries: LibrarySummary[],
  pages: Array<{ items: ScanRun[] } | undefined>,
): LibraryScanRecord[] {
  return libraries
    .flatMap((library, index) =>
      (pages[index]?.items ?? []).map((scan) => ({
        id: `scan:${scan.id}`,
        libraryId: library.id,
        libraryName: library.name,
        scan,
      })),
    )
    .sort((left, right) =>
      right.scan.createdAt.localeCompare(left.scan.createdAt),
    );
}
