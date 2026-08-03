import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { expect, it } from "vitest";

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
