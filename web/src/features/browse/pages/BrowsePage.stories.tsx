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
      <StoryApiBoundary manyDirectories={context.parameters.manyDirectories === true}>
        <ThemeProvider>
          <QueryClientProvider client={new QueryClient()}>
            <ToastProvider>
              <MemoryRouter
                initialEntries={[
                  "/libraries/lib_family/browse/dir_kyoto",
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

export const ManyDirectories: Story = {
  parameters: { manyDirectories: true },
  name: "Many directories (synthetic contract fixtures)",
};

function StoryApiBoundary({ children, manyDirectories }: { children: ReactNode; manyDirectories: boolean }) {
  window.fetch = storyFetch(passthroughFetch, manyDirectories);
  useEffect(() => () => {
    window.fetch = passthroughFetch;
  }, []);
  return children;
}

function storyFetch(originalFetch: typeof window.fetch, manyDirectories: boolean): typeof window.fetch {
  const fixtureDirectories = manyDirectories ? [
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
  ] : directories;
  return async (input, init) => {
    const url = new URL(
      input instanceof Request ? input.url : String(input),
      window.location.origin,
    );
    if (!url.pathname.startsWith("/api/v1/")) {
      return originalFetch(input, init);
    }

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
            status: "scanning",
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
          status: "scanning",
        },
        { ETag: '"library-story"' },
      );
    }
    if (url.pathname === "/api/v1/directories/dir_kyoto") {
      return json({
        ...fixtureDirectories.find(({ id }) => id === "dir_kyoto"),
        breadcrumbs: [
          { id: "dir_root", name: "家庭影像", relativePath: "" },
          { id: "dir_travel", name: "旅行", relativePath: "旅行" },
          { id: "dir_japan", name: "日本", relativePath: "旅行/日本" },
          {
            id: "dir_kyoto",
            name: "京都",
            relativePath: "旅行/日本/京都",
          },
        ],
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
      return json({
        items: assets.filter(
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
