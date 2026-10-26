import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState, type ReactNode } from "react";
import type { AssetCuration } from "../../../lib/api/curation";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { curationKeys } from "../queries";
import { AssetCurationControls } from "./AssetCurationControls";

const meta = {
  title: "Features/Curation/Photo tag ownership",
  component: AssetCurationControls,
  args: { assetId: "ast_synthetic", csrfToken: "synthetic-story-token-not-a-session" },
  decorators: [(Story, context) => <OwnershipFixture full={context.parameters.full ?? false} longTag={context.parameters.longTag ?? false}><Story /></OwnershipFixture>],
  parameters: { docs: { description: { component: "Production photo editor, synthetic manual/AI source snapshots. Read-only preview: writes return a synthetic conflict instead of reaching an application. Real mutation semantics are covered by component, query, database and local Docker tests." } } },
} satisfies Meta<typeof AssetCurationControls>;
export default meta;
type Story = StoryObj<typeof meta>;
export const AIAndManual: Story = {};
export const AtTagLimit: Story = { parameters: { full: true } };
export const LongManualName: Story = { parameters: { longTag: true } };

function OwnershipFixture({ children, full, longTag }: { children: ReactNode; full: boolean; longTag: boolean }) {
  const [client] = useState(() => {
    const result = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity }, mutations: { retry: false } } });
    const timestamps = { createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z" };
    const state: AssetCuration = { assetId: "ast_synthetic", favorite: false, favoritedAt: null, revision: 1, tags: [
      { ...timestamps, id: "tag_manual", name: longTag ? "保留完整的长标签名称且不遮挡人工来源和删除按钮".repeat(2).slice(0, 32) : "精选", assetCount: 1, source: "manual" },
      { ...timestamps, id: "tag_ai", name: "sky", assetCount: 1, source: "ai", localization: { key: "sky", zhCN: "天空", en: "Sky" } },
      ...(full ? Array.from({ length: 18 }, (_, i) => ({ ...timestamps, id: `tag_${i}`, name: `样例 ${i + 1}`, assetCount: 1, source: "manual" as const })) : []),
    ] };
    result.setQueryData(curationKeys.asset(state.assetId), state);
    result.setQueryData(curationKeys.tags, { pages: [{ items: state.tags, nextCursor: null }], pageParams: [undefined] });
    return result;
  });
  useEffect(() => {
    const original = window.fetch;
    window.fetch = async (input, init) => {
      const url = new URL(input instanceof Request ? input.url : String(input), window.location.origin);
      if (!url.pathname.startsWith("/api/v1/")) return original(input, init);
      return Response.json({ error: { code: "precondition_failed", message: "Read-only synthetic snapshot", requestId: "story-fixture" } }, { status: 412 });
    };
    return () => { window.fetch = original; client.clear(); };
  }, [client]);
  return <ThemeProvider><QueryClientProvider client={client}>
    <p role="note">只读组件预览 · 合成数据 / Read-only preview · synthetic data</p>{children}
  </QueryClientProvider></ThemeProvider>;
}
