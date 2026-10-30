import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type { Asset, Directory } from "../../../lib/api/catalog";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { BrowsePage } from "./BrowsePage";

const session: AuthenticatedSession = {
  administrator: {
    displayName: "管理员",
    id: "adm_story",
    username: "admin",
  },
  csrfToken: "storybook-csrf-token-that-is-long-enough",
  expiresAt: "2026-08-04T00:00:00Z",
};
const passthroughFetch = globalThis.fetch.bind(globalThis);

const directories: Directory[] = [
  directory("dir_root", "家庭影像", "", null, 12_480, true),
  directory("dir_year", "按年份", "按年份", "dir_root", 3_100, true),
  directory("dir_travel", "旅行", "旅行", "dir_root", 5_320, true),
  directory("dir_japan", "日本", "旅行/日本", "dir_travel", 1_236, true),
  directory("dir_tokyo", "东京", "旅行/日本/东京", "dir_japan", 186),
  directory("dir_kyoto", "京都", "旅行/日本/京都", "dir_japan", 312, true),
  directory("dir_osaka", "大阪", "旅行/日本/大阪", "dir_japan", 124),
  directory("dir_hokkaido", "北海道", "旅行/日本/北海道", "dir_japan", 98),
  directory("dir_kiyomizu", "清水寺", "旅行/日本/京都/清水寺", "dir_kyoto", 42),
  directory("dir_fushimi", "伏见稻荷大社", "旅行/日本/京都/伏见稻荷大社", "dir_kyoto", 36),
  directory("dir_kinkaku", "金阁寺", "旅行/日本/京都/金阁寺", "dir_kyoto", 31),
  directory("dir_arashiyama", "岚山", "旅行/日本/京都/岚山", "dir_kyoto", 28),
  directory("dir_gion", "祇园", "旅行/日本/京都/祇园", "dir_kyoto", 26),
  directory("dir_nijo", "二条城", "旅行/日本/京都/二条城", "dir_kyoto", 24),
  directory("dir_other", "其他", "旅行/日本/京都/其他", "dir_kyoto", 18),
];

const assets: Asset[] = Array.from({ length: 12 }, (_, index) => ({
  directoryId: "dir_kyoto",
  durationMs: index % 4 === 1 ? 18_000 + index * 1_000 : null,
  height: 3376,
  id: `ast_${index + 1}`,
  kind: index % 4 === 1 ? "video" : "image",
  libraryId: "lib_family",
  libraryName: "家庭影像",
  mimeType: index % 4 === 1 ? "video/mp4" : "image/jpeg",
  modifiedAt: `2026-07-${String(21 - index).padStart(2, "0")}T${String(19 - (index % 9)).padStart(2, "0")}:12:43Z`,
  name:
    index === 0
      ? "2026-07-21 19-12-43.jpg"
      : `2026-07-${String(21 - index).padStart(2, "0")} ${String(18 - (index % 8)).padStart(2, "0")}-33-${String(27 + index).padStart(2, "0")}.${index % 4 === 1 ? "mp4" : "jpg"}`,
  playbackStatus: index % 4 === 1 ? "playable" : "not_applicable",
  probeStatus: "ready",
  relativePath: `旅行/日本/京都/media-${index + 1}`,
  sizeBytes: 8_400_000 + index * 120_000,
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
  thumbnail: {
    errorCode: null,
    status: "unavailable",
    url: null,
  },
  width: 6000,
}));

const meta = {
  title: "Features/Browse/Directory",
  component: BrowsePage,
  decorators: [
    (Story, context) => (
      <StoryApiBoundary
        manyDirectories={context.parameters.manyDirectories === true}
        archiveFixtures={context.parameters.archiveFixtures === true}
        sevenZipFixtures={context.parameters.sevenZipFixtures === true}
        folderCoverFixtures={context.parameters.folderCoverFixtures === true}
      >
        <ThemeProvider>
          <QueryClientProvider client={new QueryClient()}>
            <ToastProvider>
              <MemoryRouter
                initialEntries={[
                  `/libraries/lib_family/browse/${context.args.directoryId ?? "dir_kyoto"}`,
                ]}
              >
                <Story />
              </MemoryRouter>
            </ToastProvider>
          </QueryClientProvider>
        </ThemeProvider>
      </StoryApiBoundary>
    ),
  ],
  parameters: { layout: "fullscreen" },
  args: {
    directoryId: "dir_kyoto",
    libraryId: "lib_family",
    session,
  },
} satisfies Meta<typeof BrowsePage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const DirectoryActions: Story = {
  name: "Directory actions and context menu (synthetic contract fixtures)",
  parameters: { docs: { description: { story: "Production navigation: use Actions to expand/collapse all, or right-click a directory for its own action. The tree loads and renders a bounded visible window. Deletion is disabled in these navigation-only fixtures; file management has separate accepted contract stories." } } },
};

export const CollapsibleNavigationAndDock: Story = {
  name: "Collapsible navigation and scroll dock (synthetic fixtures)",
  parameters: {
    manyDirectories: true,
    docs: { description: { story: "Production BrowsePage with synthetic contract fixtures. Collapse Quick access to expand the directory tree, then scroll to see the compact toolbar transition. The same controls retain their values and focus." } },
  },
};

export const ManyDirectories: Story = {
  name: "Many directories (synthetic contract fixtures)",
  parameters: { manyDirectories: true, docs: { description: { story: "Production directory tree and grid with paginated synthetic directory contracts. Scroll the tree near its bottom to append; expand the grid before scrolling to load subsequent pages. Manual buttons remain available." } } },
};

export const FolderCovers: Story = {
  name: "Folder covers (synthetic contract fixtures)",
  parameters: { folderCoverFixtures: true, docs: { description: { story: "Production folder cards with synthetic directory contracts: existing thumbnail, cover selected from a descendant, no thumbnail, and thumbnail failure. Folder names, counts, navigation and the folder icon remain available." } } },
};

export const ZipArchives: Story = {
  name: "ZIP archives (synthetic contract fixtures)",
  parameters: {
    archiveFixtures: true,
    docs: { description: { story: "Production BrowsePage with synthetic ZIP directory contracts: ready cover, long filename, unreadable archive, encrypted archive. Originals in the application remain read-only." } },
  },
};

export const ZipContents: Story = {
  name: "ZIP contents (synthetic contract fixtures)",
  args: { directoryId: "dir_zip" },
  parameters: { archiveFixtures: true },
};

export const ZipUnreadable: Story = {
  name: "Unreadable ZIP (synthetic contract fixtures)",
  args: { directoryId: "dir_zip_bad" },
  parameters: { archiveFixtures: true },
};

export const ZipEncrypted: Story = {
  name: "Encrypted ZIP (synthetic contract fixtures)",
  args: { directoryId: "dir_zip_encrypted" },
  parameters: { archiveFixtures: true },
};

export const SevenZipArchives: Story = {
  name: "7z archives (synthetic contract fixtures)",
  parameters: { archiveFixtures: true, sevenZipFixtures: true },
};

export const SevenZipContents: Story = {
  name: "7z contents (synthetic contract fixtures)",
  args: { directoryId: "dir_zip" },
  parameters: { archiveFixtures: true, sevenZipFixtures: true },
};

function StoryApiBoundary({ children, manyDirectories, archiveFixtures, sevenZipFixtures, folderCoverFixtures }: { children: ReactNode; manyDirectories: boolean; archiveFixtures: boolean; sevenZipFixtures: boolean; folderCoverFixtures: boolean }) {
  window.fetch = storyFetch(passthroughFetch, manyDirectories, archiveFixtures, sevenZipFixtures, folderCoverFixtures);
  useEffect(() => () => {
    window.fetch = passthroughFetch;
  }, []);
  return children;
}

function storyFetch(originalFetch: typeof window.fetch, manyDirectories: boolean, archiveFixtures: boolean, sevenZipFixtures: boolean, folderCoverFixtures: boolean): typeof window.fetch {
  const archivePath = "旅行/日本/京都/京都图集.zip";
  const zipDirectories: Directory[] = [
    { ...directory("dir_zip", "京都图集.zip", archivePath, "dir_kyoto", 6, true), archivePath, archiveStatus: "ready", coverAssetId: "ast_zip_1", directAssetCount: 3 },
    { ...directory("dir_zip_day", "第一天", `${archivePath}/第一天`, "dir_zip", 3), archivePath, archiveStatus: "ready" },
    { ...directory("dir_zip_empty", "空目录", `${archivePath}/空目录`, "dir_zip", 0), archivePath, archiveStatus: "ready" },
    { ...directory("dir_zip_long", "一份很长名称的旅行图集 · 京都街道与山间日常的合成演示图片.zip", "旅行/日本/京都/长名称图集.zip", "dir_kyoto", 12), archivePath: "旅行/日本/京都/长名称图集.zip", archiveStatus: "ready", coverAssetId: "ast_zip_2" },
    { ...directory("dir_zip_bad", "损坏压缩包.zip", "旅行/日本/京都/损坏压缩包.zip", "dir_kyoto", 0), archivePath: "旅行/日本/京都/损坏压缩包.zip", archiveStatus: "unreadable" },
    { ...directory("dir_zip_encrypted", "需要密码.zip", "旅行/日本/京都/需要密码.zip", "dir_kyoto", 0), archivePath: "旅行/日本/京都/需要密码.zip", archiveStatus: "encrypted" },
  ];
  const fixtureDirectories = archiveFixtures ? [
    ...directories.filter(item => item.id === "dir_root" || item.id === "dir_travel" || item.id === "dir_japan" || item.id === "dir_kyoto" || item.id === "dir_kiyomizu"),
    ...zipDirectories.map((entry) => sevenZipFixtures ? {
      ...entry,
      name: entry.name.replace(/\.zip$/i, ".7z"),
      relativePath: entry.relativePath.replace(/\.zip(?=\/|$)/gi, ".7z"),
      archivePath: entry.archivePath?.replace(/\.zip$/i, ".7z") ?? null,
    } : entry),
  ] : manyDirectories ? [
    ...directories,
    ...Array.from({ length: 24 }, (_, index) => directory(
      `dir_extra_${index}`,
      index % 2 === 0
        ? `2026 年旅行影像 · 第 ${index + 1} 辑 · 京都与周边城市的街道、公园和日常记录`
        : `Travel Collection ${index + 1} — A very long folder name without spaces: ${"photographs".repeat(6)}`,
      `旅行/日本/京都/合集${index + 1}`,
      "dir_kyoto",
      index * 11,
    )),
  ] : folderCoverFixtures ? directories.map(entry => ({
    ...entry,
    ...(entry.id === "dir_kiyomizu" ? { coverAssetId: "ast_folder_1" } : {}),
    ...(entry.id === "dir_fushimi" ? { coverAssetId: "ast_folder_2", directAssetCount: 0, hasChildren: true } : {}),
    ...(entry.id === "dir_arashiyama" ? { coverAssetId: "ast_folder_failed" } : {}),
  })) : directories;
  return async (input, init) => {
    const url = new URL(
      input instanceof Request ? input.url : String(input),
      window.location.origin,
    );
    if (!url.pathname.startsWith("/api/v1/")) {
      return originalFetch(input, init);
    }

    if (url.pathname === "/api/v1/assets/ast_folder_failed/thumbnail") return new Response("", { status: 404 });
    if (/^\/api\/v1\/assets\/ast_(zip|folder)_\d+\/(thumbnail|content)$/.test(url.pathname)) {
      const warm = url.pathname.includes("_2");
      const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="960" height="640"><rect width="960" height="640" fill="${warm ? "#ad755c" : "#447b91"}"/><circle cx="710" cy="160" r="70" fill="#f1d59a"/><path d="M0 550L280 230L640 640H0" fill="${warm ? "#785c46" : "#335b69"}"/><text x="50" y="590" fill="white" font-family="sans-serif" font-size="35">Synthetic image</text></svg>`;
      return new Response(svg, { headers: { "Content-Type": "image/svg+xml" } });
    }

    if (/^\/api\/v1\/assets\/[^/]+\/exif$/.test(url.pathname)) return Response.json({ fields: [{ key: "cameraModel", value: "Synthetic Camera" }, { key: "aperture", value: "f/2.8" }] });
    if (url.pathname === "/api/v1/libraries") {
      return json({
        items: [
          {
            assetCount: 12_480,
            directoryCount: fixtureDirectories.length,
            displayPath: "/library/family",
            id: "lib_family",
            lastSuccessfulScanAt: "2026-07-29T08:00:00Z",
            latestScanId: "scan_story",
            name: "家庭影像",
            status: archiveFixtures ? "ready" : "scanning",
          },
        ],
        nextCursor: null,
      });
    }
    if (url.pathname === "/api/v1/libraries/lib_family") {
      return json(
        {
          assetCount: 12_480,
          directoryCount: fixtureDirectories.length,
          displayPath: "/library/family",
          id: "lib_family",
          lastSuccessfulScanAt: "2026-07-29T08:00:00Z",
          latestScanId: "scan_story",
          name: "家庭影像",
          status: archiveFixtures ? "ready" : "scanning",
        },
        { ETag: '"library-story"' },
      );
    }
    if (url.pathname.startsWith("/api/v1/directories/")) {
      const selected = fixtureDirectories.find(item => item.id === url.pathname.split("/").at(-1));
      if (!selected) return json({ code: "not_found", message: "Not found" }, {}, 404);
      const breadcrumbs = [];
      let ancestor: Directory | undefined = selected;
      while (ancestor) {
        breadcrumbs.unshift({ id: ancestor.id, name: ancestor.name, relativePath: ancestor.relativePath });
        const parentId: string | null = ancestor.parentId;
        ancestor = fixtureDirectories.find(item => item.id === parentId);
      }
      return json({
        ...selected,
        breadcrumbs,
      });
    }
    if (url.pathname === "/api/v1/libraries/lib_family/directories") {
      const parentId = url.searchParams.get("parentId");
      const q = url.searchParams.get("q")?.toLocaleLowerCase() ?? "";
      const matches = fixtureDirectories.filter(
        (item) => item.parentId === (parentId || "dir_root") &&
          (!q || item.name.toLocaleLowerCase().includes(q)),
      );
      const start = Number(url.searchParams.get("cursor") ?? 0);
      return json({
        items: matches.slice(start, start + 16),
        nextCursor: start + 16 < matches.length ? String(start + 16) : null,
      });
    }
    if (url.pathname === "/api/v1/libraries/lib_family/assets") {
      const q = url.searchParams.get("q")?.toLocaleLowerCase() ?? "";
      const kinds = url.searchParams.getAll("kind");
      const directoryId = url.searchParams.get("directoryId");
      const fixtureAssets = archiveFixtures && directoryId?.startsWith("dir_zip")
        ? directoryId === "dir_zip_bad" || directoryId === "dir_zip_encrypted"
          ? []
          : [1, 2, 10].map((ordinal, index): Asset => ({
              ...assets[index]!,
              directoryId,
              durationMs: null,
              id: `ast_zip_${ordinal}`,
              kind: "image",
              mimeType: "image/png",
              name: `${ordinal}.png`,
              relativePath: `${archivePath}/${ordinal}.png`,
              playbackStatus: "not_applicable",
              thumbnail: { errorCode: null, status: "ready", url: `/api/v1/assets/ast_zip_${ordinal}/thumbnail?variant=grid` },
              width: 960, height: 640,
            }))
        : assets;
      return json({
        items: fixtureAssets.filter(
          (item) =>
            (!q || item.name.toLocaleLowerCase().includes(q)) &&
            (kinds.length === 0 || kinds.includes(item.kind)),
        ),
        nextCursor: null,
      });
    }
    return json({ code: "not_found", message: "Not found" }, {}, 404);
  };
}

function directory(
  id: string,
  name: string,
  relativePath: string,
  parentId: string | null,
  recursiveAssetCount: number,
  hasChildren = false,
): Directory {
  return {
    directAssetCount: Math.min(recursiveAssetCount, 312),
    hasChildren,
    id,
    libraryId: "lib_family",
    name,
    parentId,
    recursiveAssetCount,
    relativePath,
  };
}

function json(
  value: unknown,
  headers: Record<string, string> = {},
  status = 200,
): Response {
  return new Response(JSON.stringify(value), {
    headers: { "Content-Type": "application/json", ...headers },
    status,
  });
}
