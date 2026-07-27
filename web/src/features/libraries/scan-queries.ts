import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  cancelScan,
  getScan,
  listLibraryScans,
  requestLibraryScan,
  type ScanRun,
} from "../../lib/api/scans";
import { libraryKeys } from "./queries";

export const scanKeys = {
  all: ["scans"] as const,
  detail: (scanId: string) => ["scans", "detail", scanId] as const,
  library: (libraryId: string) => ["scans", "library", libraryId] as const,
};

export function useLibraryScansQuery(libraryId: string) {
  return useQuery({
    queryKey: scanKeys.library(libraryId),
    queryFn: () => listLibraryScans(libraryId),
    staleTime: 5_000,
  });
}

export function useScanQuery(scanId: string | undefined) {
  return useQuery({
    queryKey: scanKeys.detail(scanId ?? "inactive"),
    queryFn: () => {
      if (!scanId) throw new Error("Scan query is not active.");
      return getScan(scanId);
    },
    enabled: Boolean(scanId),
    refetchInterval: (query) =>
      isActiveScan(query.state.data) ? 1_500 : false,
  });
}

export function useRequestScanMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: requestLibraryScan,
    onSuccess: (scan) => {
      queryClient.setQueryData(scanKeys.detail(scan.id), scan);
      void queryClient.invalidateQueries({
        queryKey: scanKeys.library(scan.libraryId),
      });
      void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}

export function useCancelScanMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: cancelScan,
    onSuccess: (scan) => {
      queryClient.setQueryData(scanKeys.detail(scan.id), scan);
      void queryClient.invalidateQueries({
        queryKey: scanKeys.library(scan.libraryId),
      });
      void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}

export function isActiveScan(scan: ScanRun | undefined) {
  return scan?.status === "queued" || scan?.status === "running";
}
