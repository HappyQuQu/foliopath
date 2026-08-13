import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { listDirectories, type Asset } from "../../../lib/api/catalog";
import {
  getAssetCuration,
  listFavorites,
  listTagAssets,
  listTags,
} from "../../../lib/api/curation";
import { listLibraries } from "../../../lib/api/libraries";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { CurationPage } from "./CurationPage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../lib/api/catalog")>();
  return { ...actual, listDirectories: vi.fn() };
});

vi.mock("../../../lib/api/curation", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../lib/api/curation")>();
  return {
    ...actual,
    getAssetCuration: vi.fn(),
    listFavorites: vi.fn(),
    listTagAssets: vi.fn(),
    listTags: vi.fn(),
  };
});

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../lib/api/libraries")>();
  return { ...actual, listLibraries: vi.fn() };
});

const session: AuthenticatedSession = {
  administrator: {
    displayName: "Admin",
    id: "adm_test",
    username: "admin",
  },
  csrfToken: "csrf-token-that-is-long-enough-for-the-contract",
  expiresAt: "2026-08-12T00:00:00Z",
};

const asset: Asset = {
  directoryId: "dir_travel",
  durationMs: null,
  favorite: true,
  height: 800,
  id: "asset_favorite",
  kind: "image",
  libraryId: "lib_family",
  libraryName: "Family",
  mimeType: "image/jpeg",
  modifiedAt: "2026-08-10T00:00:00Z",
  name: "favorite.jpg",
  playbackStatus: "not_applicable",
  probeStatus: "ready",
  relativePath: "Travel/favorite.jpg",
  sizeBytes: 1024,
  sourceAvailability: "available",
  storyboard: {
    cellHeight: null,
    cellWidth: null,
    columns: null,
    errorCode: null,
    frameCount: null,
    rows: null,
    status: "not_applicable",
    url: null,
  },
  thumbnail: { errorCode: null, status: "ready", url: "/thumbnail.jpg" },
  width: 1200,
};

beforeEach(() => {
  vi.mocked(listLibraries).mockResolvedValue({
    items: [{
      assetCount: 1,
      automaticDiscoveryErrorCode: null,
      automaticDiscoveryStatus: "active",
      contentRevision: 2,
      directoryCount: 2,
      displayPath: "/library/family",
      id: "lib_family",
      lastAutomaticDiscoveryAt: "2026-08-10T00:00:00Z",
      lastSuccessfulScanAt: "2026-08-10T00:00:00Z",
      latestScanId: "scan_test",
      name: "Family",
      status: "ready",
    }],
    nextCursor: null,
  });
  vi.mocked(listDirectories).mockResolvedValue({
    items: [{
      directAssetCount: 1,
      hasChildren: false,
      id: "dir_travel",
      libraryId: "lib_family",
      name: "Travel",
      parentId: null,
      recursiveAssetCount: 1,
      relativePath: "Travel",
    }],
    nextCursor: null,
  });
  vi.mocked(listTags).mockResolvedValue({ items: [], nextCursor: null });
  vi.mocked(listTagAssets).mockResolvedValue({
    counts: { all: 0, images: 0, videos: 0 },
    items: [],
    nextCursor: null,
  });
  vi.mocked(listFavorites).mockResolvedValue({
    counts: { all: 1, images: 1, videos: 0 },
    items: [{
      asset,
      curation: {
        assetId: asset.id,
        favorite: true,
        favoritedAt: "2026-08-10T00:00:00Z",
        revision: 2,
        tags: [],
      },
    }],
    nextCursor: null,
  });
  vi.mocked(getAssetCuration).mockResolvedValue({
    assetId: asset.id,
    favorite: true,
    favoritedAt: "2026-08-10T00:00:00Z",
    revision: 2,
    tags: [],
  });
});

it("keeps quick access, hides directories, and previews favorites", async () => {
  const user = userEvent.setup();
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <MemoryRouter initialEntries={["/favorites?libraryId=lib_family"]}>
            <CurationPage session={session} />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );

  const favoritesLink = await screen.findByRole("link", { name: /Favorites|收藏/ });
  expect(favoritesLink).toHaveAttribute(
    "aria-current",
    "page",
  );
  expect(favoritesLink).toHaveTextContent("1");
  expect(screen.queryByRole("navigation", { name: /Directory navigation|目录导航/ }))
    .not.toBeInTheDocument();

  await user.click(
    await screen.findByRole("button", {
      name: /^(Preview|预览) favorite\.jpg$/,
    }),
  );

  expect(
    await screen.findByRole("complementary", { name: /Preview|预览/ }),
  ).toBeVisible();
  expect(screen.getByText("Travel/favorite.jpg")).toBeVisible();
});

it("replaces the directory tree with a count-weighted tag wall", async () => {
  vi.mocked(listTags).mockResolvedValue({
    items: [
      {
        assetCount: 2,
        createdAt: "2026-08-10T00:00:00Z",
        id: "tag_small",
        name: "Travel",
        updatedAt: "2026-08-10T00:00:00Z",
      },
      {
        assetCount: 64,
        createdAt: "2026-08-10T00:00:00Z",
        id: "tag_large",
        name: "Family",
        updatedAt: "2026-08-10T00:00:00Z",
      },
    ],
    nextCursor: null,
  });
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <MemoryRouter initialEntries={["/tags?libraryId=lib_family"]}>
            <CurationPage session={session} />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );

  const wall = await screen.findByRole("navigation", { name: /All tags|全部标签/ });
  const smallTag = screen.getByRole("link", { name: /Travel/ });
  const largeTag = screen.getByRole("link", { name: /Family/ });
  expect(wall).toContainElement(smallTag);
  expect(largeTag).toHaveAttribute("data-weight", "5");
  expect(Number(smallTag.getAttribute("data-weight"))).toBeLessThan(5);
  expect(screen.getByLabelText(/64 media items|64 项媒体/)).toBeVisible();
  expect(screen.queryByRole("navigation", { name: /Directory navigation|目录导航/ }))
    .not.toBeInTheDocument();
});
