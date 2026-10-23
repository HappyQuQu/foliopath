import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "./client";
import {
  assetContentUrl,
  getCatalogState,
  getAsset,
  getDirectory,
  listAssets,
  listDirectories,
  searchAssets,
  searchLibraryAssets,
  searchSemanticAssets,
} from "./catalog";

vi.mock("./client", () => ({
  apiClient: {
    GET: vi.fn(),
  },
}));

describe("catalog adapter", () => {
  beforeEach(() => {
    vi.mocked(apiClient.GET).mockReset();
  });

  it("builds an encoded same-origin asset content URL", () => {
    expect(assetContentUrl("asset/with space")).toBe(
      "/api/v1/assets/asset%2Fwith%20space/content",
    );
  });

  it("uses the catalog validator for lightweight revision checks", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: undefined,
      error: undefined,
      response: new Response(null, {
        headers: { ETag: '"catalog-r7"' },
        status: 304,
      }),
    } as never);

    await expect(getCatalogState('"catalog-r7"')).resolves.toEqual({
      contentRevision: null,
      etag: '"catalog-r7"',
    });
    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/catalog/state", {
      params: { header: { "If-None-Match": '"catalog-r7"' } },
    });
  });

  it("returns a changed catalog revision and its new validator", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: { contentRevision: 8 },
      error: undefined,
      response: new Response(null, {
        headers: { ETag: '"catalog-r8"' },
        status: 200,
      }),
    } as never);

    await expect(getCatalogState('"catalog-r7"')).resolves.toEqual({
      contentRevision: 8,
      etag: '"catalog-r8"',
    });
  });

  it("loads one asset through the generated detail operation", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        directoryId: "dir_japan",
        durationMs: null,
        height: 800,
        id: "ast_photo",
        kind: "image",
        libraryId: "lib_family",
        libraryName: "家庭影像",
        mimeType: "image/jpeg",
        modifiedAt: "2026-07-28T00:00:00Z",
        name: "photo.jpg",
        playbackStatus: "not_applicable",
        probeStatus: "ready",
        relativePath: "旅行/photo.jpg",
        sizeBytes: 1024,
        sourceAvailability: "available",
        thumbnail: { errorCode: null, status: "ready", url: "/thumbnail" },
        width: 1200,
      },
      error: undefined,
      response: new Response(),
    } as never);

    await expect(getAsset("ast_photo")).resolves.toMatchObject({
      id: "ast_photo",
      libraryId: "lib_family",
    });
    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/assets/{assetId}", {
      params: { path: { assetId: "ast_photo" } },
    });
  });

  it("requests a bounded direct-child page through the generated client", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        counts: { all: 8, images: 6, videos: 2 },
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

  it("binds recursive scope and explicit sorting to a bounded asset page", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        counts: { all: 8, images: 6, videos: 2 },
        items: [
          {
            directoryId: "dir_japan",
            durationMs: 18_000,
            height: 1080,
            id: "ast_clip",
            kind: "video",
            libraryId: "lib_family",
            libraryName: "家庭影像",
            mimeType: "video/mp4",
            modifiedAt: "2026-07-28T00:00:00Z",
            name: "clip.mp4",
            playbackStatus: "playable",
            probeStatus: "ready",
            relativePath: "旅行/日本/clip.mp4",
            sizeBytes: 1024,
            sourceAvailability: "available",
            thumbnail: {
              errorCode: null,
              status: "ready",
              url: "/api/v1/assets/ast_clip/thumbnail?variant=grid",
            },
            width: 1920,
          },
        ],
        nextCursor: null,
      },
      error: undefined,
      response: new Response(),
    } as never);

    const page = await listAssets({
      directoryId: "dir_travel",
      kinds: ["video"],
      libraryId: "lib_family",
      order: "desc",
      recursive: true,
      sort: "modifiedAt",
    });

    expect(page.items[0]).toMatchObject({
      id: "ast_clip",
      relativePath: "旅行/日本/clip.mp4",
    });
    expect(page.counts).toEqual({ all: 8, images: 6, videos: 2 });
    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/libraries/{libraryId}/assets",
      {
        params: {
          path: { libraryId: "lib_family" },
          query: {
            directoryId: "dir_travel",
            kind: ["video"],
            limit: 50,
            order: "desc",
            recursive: true,
            sort: "modifiedAt",
          },
        },
      },
    );

  });

  it("keeps a filtered library root bound to its explicit recursive mode", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        counts: { all: 0, images: 0, videos: 0 },
        items: [],
        nextCursor: null,
      },
      error: undefined,
      response: new Response(),
    } as never);

    await listAssets({
      libraryId: "lib_family",
      order: "desc",
      q: "日本",
      recursive: true,
      sort: "modifiedAt",
    });

    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/libraries/{libraryId}/assets",
      {
        params: {
          path: { libraryId: "lib_family" },
          query: {
            limit: 50,
            order: "desc",
            q: "日本",
            recursive: true,
            sort: "modifiedAt",
          },
        },
      },
    );

    await listAssets({
      libraryId: "lib_family",
      order: "desc",
      q: "日本",
      recursive: false,
      sort: "modifiedAt",
    });

    expect(apiClient.GET).toHaveBeenLastCalledWith(
      "/api/v1/libraries/{libraryId}/assets",
      {
        params: {
          path: { libraryId: "lib_family" },
          query: {
            limit: 50,
            order: "desc",
            q: "日本",
            recursive: false,
            sort: "modifiedAt",
          },
        },
      },
    );
  });

  it("binds a filtered directory search without leaking path strings", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: { items: [], nextCursor: null },
      error: undefined,
      response: new Response(),
    } as never);

    await searchLibraryAssets({
      directoryId: "dir_japan",
      kinds: ["image", "animated"],
      libraryId: "lib_family",
      modifiedBefore: "2026-07-29T00:00:00.000Z",
      modifiedFrom: "2026-06-29T00:00:00.000Z",
      order: "desc",
      q: "京都",
      recursive: true,
      sort: "modifiedAt",
    });

    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/libraries/{libraryId}/assets",
      {
        params: {
          path: { libraryId: "lib_family" },
          query: {
            directoryId: "dir_japan",
            kind: ["image", "animated"],
            limit: 50,
            modifiedBefore: "2026-07-29T00:00:00.000Z",
            modifiedFrom: "2026-06-29T00:00:00.000Z",
            order: "desc",
            q: "京都",
            recursive: true,
            sort: "modifiedAt",
          },
        },
      },
    );
  });

  it("uses the global endpoint only for all-library search", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: { items: [], nextCursor: "next-global" },
      error: undefined,
      response: new Response(),
    } as never);

    await expect(
      searchAssets({
        cursor: "cursor-global",
        kinds: ["video"],
        order: "asc",
        q: "clip",
        sort: "name",
      }),
    ).resolves.toEqual({ items: [], nextCursor: "next-global" });

    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/assets", {
      params: {
        query: {
          cursor: "cursor-global",
          kind: ["video"],
          limit: 50,
          order: "asc",
          q: "clip",
          sort: "name",
        },
      },
    });
  });

  it("uses the dedicated semantic endpoint without filename sort or filter parameters", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        coverage: {
          complete: false,
          completed: 8,
          eligible: 12,
          excludedLibraries: [],
          failed: 1,
          stale: 3,
        },
        items: [],
        nextCursor: "semantic-next",
      },
      error: undefined,
      response: new Response(),
    } as never);

    await expect(searchSemanticAssets({
      cursor: "semantic-cursor",
      directoryId: "dir_japan",
      libraryId: "lib_family",
      q: "夜晚城市灯光",
      recursive: true,
    })).resolves.toMatchObject({
      items: [],
      nextCursor: "semantic-next",
      semanticCoverage: { completed: 8, eligible: 12 },
    });

    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/semantic/assets", {
      params: {
        query: {
          cursor: "semantic-cursor",
          directoryId: "dir_japan",
          libraryId: "lib_family",
          limit: 50,
          q: "夜晚城市灯光",
          recursive: true,
        },
      },
    });
  });
});
