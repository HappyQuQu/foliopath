import { InfiniteQueryObserver, QueryClient, QueryObserver, type InfiniteData } from "@tanstack/react-query";
import { expect, it, vi } from "vitest";

import { refreshChangedCatalogQueries } from "./queries";

it("drops stale cursor pages before invalidating changed catalog consumers", async () => {
  const queryClient = new QueryClient();
  const catalogKey = ["catalog", "assets", "lib_1"] as const;
  const searchKey = ["search", "results", "lib_1"] as const;
  const fixture: InfiniteData<{ items: number[] }, string | undefined> = {
    pageParams: [undefined, "next"],
    pages: [{ items: [1] }, { items: [2] }],
  };
  queryClient.setQueryData(catalogKey, fixture);
  queryClient.setQueryData(searchKey, fixture);

  await refreshChangedCatalogQueries(queryClient, [["catalog"], ["search"]]);

  expect(queryClient.getQueryData<InfiniteData<{ items: number[] }>>(catalogKey)?.pages)
    .toEqual([{ items: [1] }]);
  expect(queryClient.getQueryData<InfiniteData<{ items: number[] }>>(searchKey)?.pages)
    .toEqual([{ items: [1] }]);
});

it("preserves regular tag-cloud and asset caches while invalidating a mixed prefix", async () => {
  const queryClient = new QueryClient();
  const vocabularyKey = ["curation", "tags"] as const;
  const cloudKey = [...vocabularyKey, "image-cloud", "lib_1", "", ""] as const;
  const assetKey = ["curation", "asset", "asset_1"] as const;
  const cloud = { items: [{ id: "tag_1", name: "花", assetCount: 3 }], nextCursor: null };
  const asset = { assetId: "asset_1", favorite: true, revision: 2, tags: [] };
  queryClient.setQueryData(vocabularyKey, {
    pageParams: [undefined, "next"],
    pages: [{ items: [1] }, { items: [2] }],
  });
  queryClient.setQueryData(cloudKey, cloud);
  queryClient.setQueryData(assetKey, asset);
  const cachedCloud = queryClient.getQueryData(cloudKey);
  const cachedAsset = queryClient.getQueryData(assetKey);

  await expect(refreshChangedCatalogQueries(queryClient, [["curation"]])).resolves.toBeUndefined();

  expect(queryClient.getQueryData(vocabularyKey)).toEqual({
    pageParams: [undefined],
    pages: [{ items: [1] }],
  });
  expect(queryClient.getQueryData(cloudKey)).toBe(cachedCloud);
  expect(queryClient.getQueryData(assetKey)).toBe(cachedAsset);
  for (const key of [vocabularyKey, cloudKey, assetKey]) {
    expect(queryClient.getQueryState(key)?.isInvalidated).toBe(true);
  }
});

it.each([
  null,
  { items: [], nextCursor: null },
  { pages: [{ items: [1] }, { items: [2] }] },
  { pages: [{ items: [1] }, { items: [2] }], pageParams: null },
  { pages: null, pageParams: [] },
  { pages: "not infinite data", pageParams: [] },
])("does not truncate a cache without both infinite-query arrays: %j", async (fixture) => {
  const queryClient = new QueryClient();
  const key = ["consumer"] as const;
  queryClient.setQueryData(key, fixture);
  const cached = queryClient.getQueryData(key);

  await refreshChangedCatalogQueries(queryClient, [key]);

  expect(queryClient.getQueryData(key)).toBe(cached);
  expect(queryClient.getQueryState(key)?.isInvalidated).toBe(true);
});

it("refetches active regular and infinite consumers without requesting stale cursor pages", async () => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  const vocabularyKey = ["curation", "tags"] as const;
  const cloudKey = [...vocabularyKey, "image-cloud", "lib_1", "", ""] as const;
  const staleCloud = { items: [1], nextCursor: null };
  const freshCloud = { items: [1, 2], nextCursor: null };
  queryClient.setQueryData(vocabularyKey, {
    pageParams: [undefined, "stale-cursor"],
    pages: [{ items: [1], nextCursor: "stale-cursor" }, { items: [2], nextCursor: null }],
  });
  queryClient.setQueryData(cloudKey, staleCloud);
  const cloudQuery = vi.fn(async () => freshCloud);
  const vocabularyQuery = vi.fn(async (_pageParam: string | undefined) => ({ items: [3], nextCursor: "fresh-cursor" }));
  const cloudObserver = new QueryObserver(queryClient, { queryKey: cloudKey, queryFn: cloudQuery });
  const vocabularyObserver = new InfiniteQueryObserver(queryClient, {
    queryKey: vocabularyKey,
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page: { items: number[]; nextCursor: string }) => page.nextCursor,
    queryFn: ({ pageParam }) => vocabularyQuery(pageParam),
  });
  const unsubscribeCloud = cloudObserver.subscribe(() => {});
  const unsubscribeVocabulary = vocabularyObserver.subscribe(() => {});

  try {
    await refreshChangedCatalogQueries(queryClient, [["curation"]]);

    expect(cloudQuery).toHaveBeenCalledOnce();
    expect(vocabularyQuery).toHaveBeenCalledExactlyOnceWith(undefined);
    expect(queryClient.getQueryData(cloudKey)).toEqual(freshCloud);
    expect(queryClient.getQueryData(vocabularyKey)).toEqual({
      pageParams: [undefined],
      pages: [{ items: [3], nextCursor: "fresh-cursor" }],
    });
  } finally {
    unsubscribeCloud();
    unsubscribeVocabulary();
    queryClient.clear();
  }
});
