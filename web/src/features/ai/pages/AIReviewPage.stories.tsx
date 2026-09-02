import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AIReviewPage } from "./AIReviewPage";

const meta = {
  title: "Features/Intelligence/Review",
  component: AIReviewPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2026-09-03T00:00:00Z" } },
  decorators: [(Story) => <StoryBoundary><Story /></StoryBoundary>],
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AIReviewPage>;
export default meta;
type Story = StoryObj<typeof meta>;

export const PendingSuggestions: Story = {};

function StoryBoundary({ children }: { children: ReactNode }) {
  const originalFetch = globalThis.fetch;
  useEffect(() => {
    globalThis.fetch = async (input) => {
      const url = String(input);
      if (url.includes("/api/v1/settings/libraries") || url.endsWith("/api/v1/libraries")) return json({ items: [{ id: "lib-story", name: "家庭照片", status: "ready" }], nextCursor: null });
      if (url.includes("tag-suggestions")) return json({ items: [{ id: "suggestion-story", libraryId: "lib-story", generationId: "gen-story", status: "pending", confidence: .93, revision: 2, vocabularyRevision: 1, reviewedAt: null, tag: { tagId: "tag-story", name: "海滩" }, asset: { id: "asset-story", libraryId: "lib-story", libraryName: "家庭照片", directoryId: "dir-story", name: "海边.jpg", relativePath: "旅行/海边.jpg", kind: "image", mimeType: "image/jpeg", sizeBytes: 1000, modifiedAt: "2026-09-01T00:00:00Z", width: 1200, height: 800, durationMs: null, probeStatus: "ready", playbackStatus: "not_applicable", sourceAvailability: "available", thumbnail: { status: "unavailable", url: null, errorCode: null } } }], nextCursor: null, coverage: { eligible: 10, completed: 8, degraded: 1, failed: 0, stale: 1, complete: false, revision: 2 } });
      if (url.includes("/curation")) return json({ assetId: "asset-story", favorite: false, favoritedAt: null, revision: 1, tags: [] }, { ETag: '"curation-r1"' });
      return originalFetch(input);
    };
    return () => { globalThis.fetch = originalFetch; };
  }, [originalFetch]);
  return <ThemeProvider><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><ToastProvider><MemoryRouter initialEntries={["/intelligence?view=tags&libraryId=lib-story"]}>{children}</MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}

function json(body: unknown, headers: HeadersInit = {}) { return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json", ...headers } })); }
