import { useQuery } from "@tanstack/react-query";

import {
  getReleaseInformation,
  getSystemStatus,
} from "../../lib/api/release-info";

export const releaseInfoKeys = {
  status: ["release-info", "status"] as const,
  releases: ["release-info", "releases"] as const,
};

export function useApplicationStatusQuery() {
  return useQuery({
    queryKey: releaseInfoKeys.status,
    queryFn: getSystemStatus,
    staleTime: Number.POSITIVE_INFINITY,
  });
}

export function useReleaseInformationQuery() {
  return useQuery({
    queryKey: releaseInfoKeys.releases,
    queryFn: () => getReleaseInformation(false),
    staleTime: 6 * 60 * 60 * 1_000,
  });
}
