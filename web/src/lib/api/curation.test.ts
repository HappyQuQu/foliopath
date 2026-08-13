import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "./client";
import { listTagAssets, replaceAssetTags, setFavorite } from "./curation";

vi.mock("./client", () => ({
  apiClient: {
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
  },
}));

const state = {
  assetId: "ast_7",
  favorite: true,
  favoritedAt: "2026-08-10T00:00:00Z",
  revision: 9,
  tags: [],
};

describe("curation adapter", () => {
  beforeEach(() => {
    vi.mocked(apiClient.GET).mockReset();
    vi.mocked(apiClient.PUT).mockReset();
    vi.mocked(apiClient.PUT).mockResolvedValue({
      data: state,
      error: undefined,
      response: new Response(),
    } as never);
  });

  it("keeps tagged media scoped to the selected library workspace", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({
      data: {
        counts: { all: 0, images: 0, videos: 0 },
        items: [],
        nextCursor: null,
      },
      error: undefined,
      response: new Response(),
    } as never);

    await listTagAssets({
      libraryId: "lib_family",
      order: "desc",
      sort: "modifiedAt",
      tagId: "tag_travel",
    });

    expect(apiClient.GET).toHaveBeenCalledWith(
      "/api/v1/tags/{tagId}/assets",
      {
        params: {
          path: { tagId: "tag_travel" },
          query: {
            libraryId: "lib_family",
            limit: 50,
            order: "desc",
            sort: "modifiedAt",
          },
        },
      },
    );
  });

  it("sends favorite mutations with the authenticated CSRF boundary", async () => {
    await expect(
      setFavorite({ assetId: "ast_7", csrfToken: "csrf-token", favorite: true }),
    ).resolves.toMatchObject(state);

    expect(apiClient.PUT).toHaveBeenCalledWith(
      "/api/v1/assets/{assetId}/favorite",
      {
        body: { favorite: true },
        headers: { "X-CSRF-Token": "csrf-token" },
        params: { path: { assetId: "ast_7" } },
      },
    );
  });

  it("binds atomic tag replacement to the curation revision", async () => {
    await replaceAssetTags({
      assetId: "ast_7",
      csrfToken: "csrf-token",
      revision: 8,
      tagIds: ["tag_3"],
    });

    expect(apiClient.PUT).toHaveBeenCalledWith(
      "/api/v1/assets/{assetId}/tags",
      {
        body: { tagIds: ["tag_3"] },
        headers: {
          "If-Match": '"curation-r8"',
          "X-CSRF-Token": "csrf-token",
        },
        params: {
          header: { "If-Match": '"curation-r8"' },
          path: { assetId: "ast_7" },
        },
      },
    );
  });
});
