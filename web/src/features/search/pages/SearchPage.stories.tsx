import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import type { Asset, AssetPage } from "../../../lib/api/catalog";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { libraryKeys } from "../../libraries";
import { searchKeys } from "../queries";
import { parseSearchUrlState } from "../urlState";
import { SearchPage } from "./SearchPage";

const meta = {
  title: "Features/Search/Mode guidance",
  component: SearchPage,
  args: {
    libraryId: "lib_story",
    session: { administrator: { id: "adm_story", username: "story", displayName: "预览管理员" }, csrfToken: "synthetic-story-csrf-token-not-a-credential", expiresAt: "2099-01-01T00:00:00Z" },
  },
  decorators: [(Story, context) => <SearchStory route={context.parameters.route ?? "/libraries/lib_story/search?q=汽车&mode=semantic"}><Story /></SearchStory>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Production SearchPage with read-only synthetic contract fixtures and gradient thumbnails, not model output or search quality evidence. Mode labels describe the actual endpoint, not per-field or OCR attribution. Unseeded interactions fail closed; no production API writes." } } },
} satisfies Meta<typeof SearchPage>;
export default meta;
type Story = StoryObj<typeof meta>;
export const SemanticResults: Story = {};
export const LiteralResults: Story = { parameters: { route: "/libraries/lib_story/search?q=汽车" } };
export const SemanticStart: Story = { parameters: { route: "/libraries/lib_story/search?mode=semantic" } };

const asset: Asset = {
  id: "ast_synthetic", libraryId: "lib_story", libraryName: "合成媒体库", directoryId: "dir_story",
  name: "汽车-合成示例.png", relativePath: "合成目录/汽车-合成示例.png", kind: "image", mimeType: "image/png",
  width: 1200, height: 800, durationMs: null, modifiedAt: "2026-10-23T00:00:00Z", sizeBytes: 1024,
  favorite: false, playbackStatus: "not_applicable", probeStatus: "ready", sourceAvailability: "available",
  storyboard: { cellHeight: null, cellWidth: null, columns: null, errorCode: null, frameCount: null, rows: null, status: "not_applicable", url: null },
  thumbnail: { errorCode: null, status: "ready", url: "/test-media/viewer-blue-violet.png" },
};
function SearchStory({ children, route }: { children: ReactNode; route: string }) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnMount: false, refetchOnWindowFocus: false, staleTime: Infinity } } });
    client.setQueryData(libraryKeys.detail("lib_story"), { etag: '"synthetic-library-r1"', library: {
      id: "lib_story", name: "合成媒体库", status: "ready", displayPath: "/synthetic", assetCount: 3, directoryCount: 1,
      contentRevision: 1, latestScanId: null, lastSuccessfulScanAt: null, automaticDiscoveryStatus: "active", automaticDiscoveryErrorCode: null, lastAutomaticDiscoveryAt: null,
    } });
    for (const mode of ["filename", "semantic"] as const) {
      const state = parseSearchUrlState(new URLSearchParams({ q: "汽车", mode }), "lib_story");
      const page: AssetPage = { items: [asset], nextCursor: null, ...(mode === "semantic" ? {
        semanticCoverage: { complete: false, completed: 2, eligible: 3, failed: 1, stale: 0, excludedLibraries: [] },
      } : {}) };
      client.setQueryData(searchKeys.results("lib_story", state), { pages: [page], pageParams: [undefined] });
    }
    return client;
  });
  useEffect(() => {
    const previous = window.fetch;
    window.fetch = async (input, init) => {
      const url = new URL(input instanceof Request ? input.url : String(input), window.location.origin);
      if (!url.pathname.startsWith("/api/v1/")) return previous(input, init);
      return Response.json({ error: { code: "precondition_failed", message: "Read-only synthetic fixture", requestId: "story-fixture" } }, { status: 412 });
    };
    return () => { window.fetch = previous; client.clear(); };
  }, [client]);
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[route]}>
    {children}
    <aside role="note">只读合成合同数据 · 渐变图不是模型搜索结果 / Synthetic contract data, not model output</aside>
  </MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
