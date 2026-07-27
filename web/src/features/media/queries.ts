import { useQuery } from "@tanstack/react-query";

import { getAsset } from "../../lib/api/catalog";

export const mediaKeys = {
  all: ["media"] as const,
  asset: (assetId: string) => ["media", "asset", assetId] as const,
};

export function useAssetQuery(assetId: string) {
  return useQuery({
    queryKey: mediaKeys.asset(assetId),
    queryFn: () => getAsset(assetId),
    enabled: assetId.length > 0,
    staleTime: 15_000,
  });
}
