import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type { Tag } from "../../../lib/api/curation";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { CurationPage } from "./CurationPage";

const session: AuthenticatedSession = {
  administrator: { displayName: "管理员", id: "adm_story", username: "admin" },
  csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-01-01T00:00:00Z",
};
const passthroughFetch = globalThis.fetch.bind(globalThis);
type Fixture = "overview" | "pagination" | "empty" | "error";
const meta = {
  title: "Features/Curation/Photo tags",
  component: CurationPage,
  args: { session },
  render: args => <FixtureRoute session={args.session} />,
  decorators: [(Story, context) => <TagStory route={context.parameters.route ?? "/tags?libraryId=lib_family"}
    fixture={context.parameters.fixture ?? "overview"}><Story /></TagStory>],
  parameters: { layout: "fullscreen", docs: { description: { component:
    "CR-2026-031: production CurationPage/LibraryTagCloud with synthetic API fixtures. Counts are fixture data, not AI output. Search, library selection and bounded pagination demonstrate the real contract. Automatic tagging and AI/manual ownership are implemented separately in production settings and the photo editor; this story does not run a model. The V2 PhotoWorkspacePrototype remains a historical design archive, not a second production tag cloud." } } },
} satisfies Meta<typeof CurationPage>;
export default meta;
type Story = StoryObj<typeof meta>;
export const FavoritesSidebar: Story = { parameters: { route: "/favorites?libraryId=lib_family" } };
export const Overview: Story = {};
export const AnotherLibrary: Story = { parameters: { route: "/tags?libraryId=lib_travel" } };
export const Pagination: Story = { parameters: { fixture: "pagination" } };
export const Empty: Story = { parameters: { fixture: "empty" } };
export const LoadFailed: Story = { parameters: { fixture: "error" } };
export const ExpiredCursor: Story = { parameters: { route: "/tags?libraryId=lib_family&tagCursor=expired" } };
export const UnknownLibrary: Story = { parameters: { route: "/tags?libraryId=lib_missing" } };
export const SelectedTagOutsideCloudPage: Story = { parameters: { route: "/tags/tag_elsewhere?libraryId=lib_family&kind=video" } };

function FixtureRoute({ session }: { session: AuthenticatedSession }) {
  return <Routes>
    <Route path="/favorites" element={<CurationPage session={session} />} />
    <Route path="/tags" element={<CurationPage session={session} />} />
    <Route path="/tags/:tagId" element={<SelectedTag session={session} />} />
  </Routes>;
}
function SelectedTag({ session }: { session: AuthenticatedSession }) {
  const { tagId } = useParams();
  return <CurationPage session={session} {...(tagId ? { tagId } : {})} />;
}
function TagStory({ children, route, fixture }: { children: ReactNode; route: string; fixture: Fixture }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } }));
  window.fetch = fixtureFetch(fixture);
  useEffect(() => () => { window.fetch = passthroughFetch; client.clear(); }, [client]);
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[route]}>
    {children}
    <aside role="note" style={{
      position: "fixed", right: "var(--space-3)", bottom: "var(--space-3)",
      zIndex: "var(--layer-header)", pointerEvents: "none",
      maxWidth: "calc(100vw - var(--space-6))", padding: "var(--space-2) var(--space-3)",
      border: "var(--border-width) solid var(--color-border)", borderRadius: "var(--radius-1)",
      color: "var(--color-text-muted)", background: "var(--color-surface)", fontSize: "var(--font-size-1)",
    }}>
      合成数据 · 非 AI 输出 / Synthetic data · not AI output
    </aside>
  </MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}

const libraries: LibrarySummary[] = ["family", "travel"].map((id, index) => ({
  id: `lib_${id}`, name: index ? "旅行记录" : "家庭影像", status: "ready", displayPath: `/${id}`,
  assetCount: index ? 80 : 420, directoryCount: 4, contentRevision: 1,
  lastSuccessfulScanAt: "2026-10-22T00:00:00Z", latestScanId: null,
  automaticDiscoveryErrorCode: null, automaticDiscoveryStatus: "active", lastAutomaticDiscoveryAt: null,
}));
const names = ["人物", "旅行", "汽车", "植物", "海边", "建筑"];
function makeTags(libraryId: string, paginated: boolean): Tag[] {
  const counts = libraryId === "lib_travel" ? [10, 70, 1, 3, 40, 9] : [240, 52, 10, 3, 2, 1];
  return Array.from({ length: paginated ? 205 : names.length }, (_, index) => ({
    id: `tag_${index}`, name: paginated ? `标签 ${String(index + 1).padStart(3, "0")}` : names[index]!,
    assetCount: counts[index % counts.length]!, createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z",
    ...(!paginated && index === 2 ? { name: "car", localization: { key: "car", zhCN: "汽车", en: "Car" } } : {}),
  }));
}
function fixtureFetch(fixture: Fixture): typeof window.fetch {
  return async (input, init) => {
    const url = new URL(input instanceof Request ? input.url : String(input), window.location.origin);
    if (!url.pathname.startsWith("/api/v1/")) return passthroughFetch(input, init);
    if (url.pathname === "/api/v1/libraries") return Response.json({ items: libraries, nextCursor: null });
    if (url.pathname === "/api/v1/favorites") return Response.json({ counts: { all: 0, images: 0, videos: 0 }, items: [], nextCursor: null });
    if (url.pathname === "/api/v1/tags") {
      if (fixture === "error") return error("internal_error", 500);
      const libraryId = url.searchParams.get("libraryId") ?? "";
      if (!libraries.some(library => library.id === libraryId)) return error("not_found", 404);
      const cursor = url.searchParams.get("cursor");
      if (cursor && cursor !== "fixture-page-2") return error("invalid_cursor", 400);
      const q = url.searchParams.get("q") ?? "";
      const items = fixture === "empty" ? [] : makeTags(libraryId, fixture === "pagination").filter(tag => tag.name.includes(q) || tag.localization?.zhCN.includes(q) || tag.localization?.en.toLowerCase().includes(q.toLowerCase()));
      return Response.json({ items: cursor ? items.slice(200) : items.slice(0, 200), nextCursor: !cursor && items.length > 200 ? "fixture-page-2" : null });
    }
    if (/^\/api\/v1\/tags\/[^/]+\/assets$/.test(url.pathname)) {
      return Response.json({ counts: { all: 0, images: 0, videos: 0 }, items: [], nextCursor: null });
    }
    // No synthetic writes or passthrough to a real API.
    return error("not_found", 404);
  };
}
function error(code: string, status: number) {
  return Response.json({ error: { code, message: "Synthetic contract fixture", requestId: "story-fixture" } }, { status });
}
