import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { curationKeys } from "../../curation";
import { AIReviewPage } from "./AIReviewPage";
import { aiKeys } from "../queries";

const meta = {
  title: "Features/Intelligence/Review",
  component: AIReviewPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-09-03T00:00:00Z" } },
  decorators: [(Story, context) => <StoryBoundary status={context.parameters.tagStatus ?? "pending"} empty={context.parameters.emptySuggestions ?? false} view={context.parameters.view ?? "tags"}><Story /></StoryBoundary>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Synthetic UI fixtures, not live model analysis." } } },
} satisfies Meta<typeof AIReviewPage>;
export default meta;
type Story = StoryObj<typeof meta>;

export const PeopleSidebar: Story = { parameters: { view: "people" } };
export const UnnamedGroupsSidebar: Story = { parameters: { view: "clusters" } };
export const PendingSuggestions: Story = {};
export const AcceptedSuggestions: Story = { parameters: { tagStatus: "accepted" } };
export const DismissedSuggestions: Story = { parameters: { tagStatus: "dismissed" } };
export const NoPendingSuggestions: Story = { parameters: { emptySuggestions: true } };

function StoryBoundary({ children, status, empty, view }: { view: string; children: ReactNode; status: "pending" | "accepted" | "dismissed"; empty: boolean }) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity } } });
    client.setQueryData(["libraries", "list"], { pages: [{ items: [{ id: "lib-story", name: "界面测试媒体库", status: "ready", assetCount: 10, directoryCount: 1 }], nextCursor: null }], pageParams: [undefined] });
    const coverage = { eligible: 10, completed: 8, degraded: 1, failed: 0, stale: 1, complete: false, revision: 2 };
    const sample = {
      id: "suggestion-story", libraryId: "lib-story", generationId: "gen-story", status: "pending", confidence: .93,
      revision: 2, vocabularyRevision: 1, reviewedAt: null, tag: { tagId: "tag-story", name: "蓝色" },
      asset: { id: "asset-story", libraryId: "lib-story", libraryName: "界面测试媒体库", directoryId: "dir-story",
        name: "合成色块.png", relativePath: "合成色块.png", kind: "image", mimeType: "image/png", sizeBytes: 1000,
        modifiedAt: "2026-09-01T00:00:00Z", width: 1200, height: 800, durationMs: null, probeStatus: "ready",
        playbackStatus: "not_applicable", sourceAvailability: "available",
        thumbnail: { status: "ready", url: "/test-media/viewer-blue-violet.png", errorCode: null } },
    };
    const items = [sample, { ...sample, id: "suggestion-violet", confidence: .81, tag: { tagId: "tag-violet", name: "紫色" } }, { ...sample, id: "suggestion-second", confidence: .76, asset: { ...sample.asset, id: "asset-second", name: "测试色块-02.png" } }];
    for (const filter of ["pending", "accepted", "dismissed"] as const) client.setQueryData(aiKeys.suggestions("lib-story", filter), { items: !empty && filter === status ? items.map(item => ({ ...item, status: filter, reviewedAt: filter === "pending" ? null : "2026-10-22T00:00:00Z" })) : [], nextCursor: null, coverage });
    client.setQueryData(curationKeys.asset("asset-story"), { assetId: "asset-story", favorite: false, favoritedAt: null, revision: 1, tags: [], etag: '"curation-r1"' });
    client.setQueryData(curationKeys.asset("asset-second"), { assetId: "asset-second", favorite: false, favoritedAt: null, revision: 1, tags: [], etag: '"curation-r1"' });
    client.setQueryData(aiKeys.people(""), { items: [{ id: "person-story", name: "青岚（合成）", revision: 1, confirmedFaceCount: 4, assetCount: 3 }], nextCursor: null });
    client.setQueryData(aiKeys.clusters("lib-story", "core"), { items: [], nextCursor: null, coverage, groupAssignmentAllowed: false });
    return client;
  });
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[`/intelligence?view=${view}&libraryId=lib-story&status=${status}`]}>{children}<aside role="note" style={{ padding: "var(--space-3)", color: "var(--color-text-muted)" }}>组件预览 · 合成数据，不是真实识别结果 / Synthetic preview, not model output</aside></MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
