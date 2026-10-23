import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
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
import { getLibrary, listLibraries } from "../../../lib/api/libraries";
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
  return { ...actual, listLibraries: vi.fn(), getLibrary: vi.fn() };
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
  vi.clearAllMocks();
  vi.mocked(getLibrary).mockRejectedValue(new Error("Library not found"));
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
  // The navigation shell can mount before the asynchronous favorite count.
  await waitFor(() => expect(favoritesLink).toHaveTextContent("1"));
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

  const wall = await screen.findByRole("navigation", { name: /Photo tags|图片标签/ });
  const smallTag = screen.getByRole("link", { name: /Travel/ });
  const largeTag = screen.getByRole("link", { name: /Family/ });
  expect(wall).toContainElement(smallTag);
  expect(largeTag).toHaveAttribute("data-weight", "4");
  expect(Number(smallTag.getAttribute("data-weight"))).toBeLessThan(4);
  expect(screen.getByLabelText(/64 photos|64 张图片/)).toBeVisible();
  expect(listTags).toHaveBeenCalledWith(undefined, { libraryId: "lib_family", kinds: ["image", "animated"] });
  expect(screen.queryByRole("navigation", { name: /Directory navigation|目录导航/ }))
    .not.toBeInTheDocument();
});

function Location() { const location = useLocation(); return <output data-testid="location">{location.pathname}{location.search}</output>; }
function renderPage(route: string, tagId?: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<ThemeProvider><QueryClientProvider client={queryClient}><LocaleProvider><ToastProvider>
    <MemoryRouter initialEntries={[route]}><CurationPage session={session} {...(tagId ? { tagId } : {})} /><Location /></MemoryRouter>
  </ToastProvider></LocaleProvider></QueryClientProvider></ThemeProvider>);
  return userEvent.setup();
}

it("never falls back to another library for an explicit unknown library", async () => {
  renderPage("/tags/tag_hidden?libraryId=lib_missing", "tag_hidden");
  expect(await screen.findByRole("alert")).toHaveTextContent(/no other library|不会自动切换/);
  expect(getLibrary).toHaveBeenCalledWith("lib_missing");
  expect(listTags).not.toHaveBeenCalled();
  expect(listTagAssets).not.toHaveBeenCalled();
  expect(listFavorites).not.toHaveBeenCalled();
  expect(screen.getByTestId("location")).toHaveTextContent("libraryId=lib_missing");
});

it("resolves a valid deep-linked library beyond the loaded picker page", async () => {
  const page = await listLibraries();
  vi.mocked(getLibrary).mockResolvedValue({ library: { ...page.items[0]!, id: "lib_later", name: "Later library" }, etag: '"library-r1"' });
  renderPage("/tags/tag_later?libraryId=lib_later&kind=image", "tag_later");
  await waitFor(() => expect(listTagAssets).toHaveBeenCalledWith(expect.objectContaining({ libraryId: "lib_later", tagId: "tag_later", kinds: ["image", "animated"] })));
  expect(listTags).toHaveBeenCalledWith(undefined, { libraryId: "lib_later", kinds: ["image", "animated"] });
  expect(screen.getByTestId("location")).toHaveTextContent("libraryId=lib_later");
});

it("rejects an explicitly empty library without fetching a global tag or media list", async () => {
  renderPage("/tags?libraryId=");
  expect(await screen.findByRole("alert")).toHaveTextContent(/no other library|不会自动切换/);
  expect(listTags).not.toHaveBeenCalled();
  expect(listTagAssets).not.toHaveBeenCalled();
  expect(listFavorites).not.toHaveBeenCalled();
});

it("keeps a deep-linked tag active even when absent from this cloud page and preserves video viewing", async () => {
  renderPage("/tags/tag_elsewhere?libraryId=lib_family&kind=video", "tag_elsewhere");
  await waitFor(() => expect(listTagAssets).toHaveBeenCalledWith(expect.objectContaining({ tagId: "tag_elsewhere", libraryId: "lib_family", kinds: ["video"] })));
  expect(screen.getByRole("heading", { level: 1, name: /Filtered by tag|已按标签筛选/ })).toBeVisible();
  expect(screen.getByTestId("location")).toHaveTextContent("/tags/tag_elsewhere");
  expect(listTags).toHaveBeenCalledWith(undefined, { libraryId: "lib_family", kinds: ["image", "animated"] });
});

it("reads the selected tag name from real result curation even when absent from the cloud page", async () => {
  vi.mocked(listTagAssets).mockResolvedValue({ counts: { all: 1, images: 1, videos: 0 }, nextCursor: null, items: [{
    asset, curation: { assetId: asset.id, favorite: false, favoritedAt: null, revision: 1,
      tags: [{ id: "tag_elsewhere", name: "Road trips", assetCount: 1, createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z" }] },
  }] });
  renderPage("/tags/tag_elsewhere?libraryId=lib_family&kind=image", "tag_elsewhere");
  expect(await screen.findByRole("heading", { level: 1, name: "Road trips" })).toBeVisible();
  expect(screen.queryByRole("link", { name: /Road trips/ })).not.toBeInTheDocument();
});

it("makes an empty tag selection explicit and allows clearing it", async () => {
  const user = renderPage("/tags/tag_empty?libraryId=lib_family&kind=image", "tag_empty");
  expect(await screen.findByRole("heading", { level: 1, name: /Filtered by tag|已按标签筛选/ })).toBeVisible();
  await user.click(screen.getByRole("button", { name: /Clear tag filter|清除标签筛选/ }));
  expect(screen.getByTestId("location")).toHaveTextContent("/tags?libraryId=lib_family&kind=image");
  expect(screen.getByTestId("location")).not.toHaveTextContent("tag_empty");
});

it("exposes the existing mobile sidebar toggle and closes it with Escape", async () => {
  const user = renderPage("/tags?libraryId=lib_family");
  const button = await screen.findByLabelText(/Open navigation|打开导航/);
  fireEvent.click(button);
  expect(button).toHaveAttribute("aria-expanded", "true");
  await user.keyboard("{Escape}");
  expect(button).toHaveAttribute("aria-expanded", "false");
});

it("resets selected tag, search and cursor when switching libraries", async () => {
  const page = await listLibraries();
  vi.mocked(listLibraries).mockResolvedValue({ items: [...page.items, { ...page.items[0]!, id: "lib_two", name: "Second" }], nextCursor: null });
  const user = renderPage("/tags/tag_old?libraryId=lib_family&tagQ=Travel&tagCursor=old&kind=video", "tag_old");
  const picker = await screen.findByRole("button", { name: /Media library：Family|媒体库：Family/ });
  await user.click(picker);
  await user.click(screen.getByRole("option", { name: /Second/ }));
  expect(screen.getByTestId("location")).toHaveTextContent("/tags?libraryId=lib_two&kind=image");
  expect(screen.getByTestId("location")).not.toHaveTextContent("tagCursor");
  await waitFor(() => expect(listTags).toHaveBeenLastCalledWith(undefined, { libraryId: "lib_two", kinds: ["image", "animated"] }));
});
