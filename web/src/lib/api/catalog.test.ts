import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "./client";
import { getDirectory, listDirectories } from "./catalog";

vi.mock("./client", () => ({
  apiClient: {
    GET: vi.fn(),
  },
}));

describe("catalog adapter", () => {
  beforeEach(() => {
    vi.mocked(apiClient.GET).mockReset();
  });

  it("requests a bounded direct-child page through the generated client", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        items: [
          {
            directAssetCount: 2,
            hasChildren: true,
            id: "dir_child",
            libraryId: "lib_family",
            name: "旅行",
            parentId: "dir_root",
            recursiveAssetCount: 8,
            relativePath: "旅行",
          },
        ],
        nextCursor: "next-page",
      },
      error: undefined,
      response: new Response(),
    } as never);

    await expect(
      listDirectories({
        cursor: "cursor",
        libraryId: "lib_family",
        limit: 25,
        parentId: "dir_root",
      }),
    ).resolves.toMatchObject({
      items: [{ id: "dir_child", name: "旅行" }],
      nextCursor: "next-page",
    });
    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/libraries/{libraryId}/directories",
      {
        params: {
          path: { libraryId: "lib_family" },
          query: { cursor: "cursor", limit: 25, parentId: "dir_root" },
        },
      },
    );
  });

  it("keeps safe root-to-current breadcrumbs from directory detail", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        breadcrumbs: [
          { id: "dir_root", name: "家庭影像", relativePath: "" },
          { id: "dir_child", name: "旅行", relativePath: "旅行" },
        ],
        directAssetCount: 2,
        hasChildren: false,
        id: "dir_child",
        libraryId: "lib_family",
        name: "旅行",
        parentId: "dir_root",
        recursiveAssetCount: 2,
        relativePath: "旅行",
      },
      error: undefined,
      response: new Response(),
    } as never);

    const directory = await getDirectory("dir_child");

    expect(directory.breadcrumbs.map((item) => item.name)).toEqual([
      "家庭影像",
      "旅行",
    ]);
    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/directories/{directoryId}",
      { params: { path: { directoryId: "dir_child" } } },
    );
  });
});
