import { useQuery } from "@tanstack/react-query";

import { getAsset, getAssetExif } from "../../lib/api/catalog";

export const mediaKeys = {
  exif: (assetId: string) => ["media", "asset", assetId, "exif"] as const,
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

export function useAssetExifQuery(assetId: string, enabled: boolean) {
  return useQuery({
    queryKey: mediaKeys.exif(assetId),
    queryFn: ({ signal }) => getAssetExif(assetId, signal),
    enabled,
    staleTime: 0,
  });
}
