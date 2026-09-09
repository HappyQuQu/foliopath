import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AIReviewPage } from "./AIReviewPage";
import { aiKeys } from "../queries";

const meta = {
  title: "Features/Intelligence/Review",
  component: AIReviewPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-09-03T00:00:00Z" } },
  decorators: [(Story) => <StoryBoundary><Story /></StoryBoundary>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Synthetic UI fixtures, not live model analysis." } } },
} satisfies Meta<typeof AIReviewPage>;
export default meta;
type Story = StoryObj<typeof meta>;

export const PendingSuggestions: Story = {};

function StoryBoundary({ children }: { children: ReactNode }) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity } } });
    client.setQueryData(["libraries", "list"], { pages: [{ items: [{ id: "lib-story", name: "界面测试媒体库", status: "ready" }], nextCursor: null }], pageParams: [undefined] });
    const coverage = { eligible: 10, completed: 8, degraded: 1, failed: 0, stale: 1, complete: false, revision: 2 };
    client.setQueryData(aiKeys.suggestions("lib-story", "pending"), { items: [{
      id: "suggestion-story", libraryId: "lib-story", generationId: "gen-story", status: "pending", confidence: .93,
      revision: 2, vocabularyRevision: 1, reviewedAt: null, tag: { tagId: "tag-story", name: "蓝色" },
      asset: { id: "asset-story", libraryId: "lib-story", libraryName: "界面测试媒体库", directoryId: "dir-story",
        name: "合成色块.png", relativePath: "合成色块.png", kind: "image", mimeType: "image/png", sizeBytes: 1000,
        modifiedAt: "2026-09-01T00:00:00Z", width: 1200, height: 800, durationMs: null, probeStatus: "ready",
        playbackStatus: "not_applicable", sourceAvailability: "available",
        thumbnail: { status: "ready", url: "/test-media/viewer-blue-violet.png", errorCode: null } },
    }], nextCursor: null, coverage });
    for (const status of ["accepted", "dismissed"]) client.setQueryData(aiKeys.suggestions("lib-story", status), { items: [], nextCursor: null, coverage });
    client.setQueryData(["curation", "asset", "asset-story"], { assetId: "asset-story", favorite: false, favoritedAt: null, revision: 1, tags: [], etag: '"curation-r1"' });
    client.setQueryData(aiKeys.people(""), { items: [], nextCursor: null });
    client.setQueryData(aiKeys.clusters("lib-story", "core"), { items: [], nextCursor: null, coverage, groupAssignmentAllowed: false });
    return client;
  });
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={["/intelligence?view=tags&libraryId=lib-story"]}>{children}</MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
