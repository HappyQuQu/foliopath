import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../../components/ui";
import type { AIModelDownloadPage } from "../../../lib/api/ai";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { aiKeys } from "../queries";
import { AISettingsPage } from "./AISettingsPage";

const meta = {
  title: "Features/Intelligence/Settings",
  component: AISettingsPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-01-01T00:00:00Z" } },
  decorators: [(Story, context) => <SettingsStory route={context.parameters.route ?? "/settings/ai"} configured={context.parameters.configured ?? false}><Story /></SettingsStory>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Contract-backed UI states using synthetic model and library metadata. These are visual fixtures, not available production model sources." } } },
} satisfies Meta<typeof AISettingsPage>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Overview: Story = {};
export const NoDownloadSource: Story = { parameters: { route: "/settings/ai?view=models" } };
export const DownloadInProgress: Story = { parameters: { route: "/settings/ai?view=models", configured: true } };
export const Libraries: Story = { parameters: { route: "/settings/ai?view=libraries", configured: true } };

function SettingsStory({ children, route, configured }: { children: ReactNode; route: string; configured: boolean }) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity } } });
    const libraries = [{ id: "lib_family", name: "家庭影像", status: "ready" }, { id: "lib_travel", name: "旅行记录", status: "ready" }];
    client.setQueryData(["libraries", "list"], { pages: [{ items: libraries, nextCursor: null }], pageParams: [undefined] });
    client.setQueryData(aiKeys.models, { items: [], activeModelId: null, activeFaceModelId: null, revision: 1 });
    client.setQueryData(aiKeys.proxy, { mode: "inherit", proxyUrl: "", revision: 1, deploymentConfigured: true, effectiveMode: "proxy" });
    for (const [index, library] of libraries.entries()) {
      const data = { libraryId: library.id, enabled: configured, state: configured ? "building" : "disabled", revision: 1, activeGenerationId: null, etag: '"semantic-r1"', coverage: { completed: configured ? 1240 + index * 120 : 0, eligible: 2490 + index * 150, failed: 0, stale: 0, complete: false, revision: 1 } };
      client.setQueryData(aiKeys.semantic(library.id), { ...data, coverageRecorded: configured });
      client.setQueryData(aiKeys.videoStatus(library.id), { libraryId: library.id, enabled: configured, offline: false, activeGenerationId: null, coverage: null });
      client.setQueryData(aiKeys.faces(library.id), { ...data, enabled: false, state: "disabled" });
    }
    const base = { sourceLabel: "示例模型来源", purpose: "semantic_image_text", version: "1.0", licenseId: "Apache-2.0", packageSizeBytes: 768 * 1024 ** 2, compatibility: "compatible", installedModelId: null };
    client.setQueryData(aiKeys.downloads, { runtimeAvailable: configured, items: configured ? [{ ...base, id: "download_vision", name: "视觉理解模型", operation: { id: "aio_visual_fixture", kind: "model_install", state: "running", phase: "copying", revision: 3, completedItems: 326 * 1024 ** 2, totalItems: base.packageSizeBytes, errorCode: null, createdAt: "2026-09-08T00:00:00Z", updatedAt: "2026-09-08T00:01:00Z", etag: '"aio_visual_fixture-r3"' } }, { ...base, id: "download_vision_alt", name: "视觉理解模型 · 备用版本", operation: null }] : [] });
    for (const item of client.getQueryData<AIModelDownloadPage>(aiKeys.downloads)?.items ?? []) {
      if (item.operation) client.setQueryData(aiKeys.operation(item.operation.id), item.operation);
    }
    // Story-only response fixture; production presets are owned by the backend.
    client.setQueryData<AIModelDownloadPage>(aiKeys.downloads, (page) => page && ({ ...page, presets: [
      { id: "preset_siglip1", name: "SigLIP Base Patch16 224", sourceLabel: "Google · Hugging Face", purpose: "semantic_image_text", readiness: "package_pending" },
      { id: "preset_siglip2", name: "SigLIP 2 Base Patch16 224", sourceLabel: "Google · Hugging Face", purpose: "semantic_image_text", readiness: "adapter_pending" },
      { id: "preset_buffalo_l", name: "buffalo_l", sourceLabel: "InsightFace · Immich", purpose: "face_detection_embedding", readiness: "adapter_pending" },
    ] }));
    return client;
  });
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[route]}>{children}<aside role="note" style={{ background: "var(--color-surface)", color: "var(--color-text-muted)", padding: "var(--space-2)", fontSize: "var(--font-size-2)" }}>组件预览 · 合成数据 / Component preview · synthetic data</aside></MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
