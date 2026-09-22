import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { aiKeys } from "../queries";
import { AISettingsPage } from "./AISettingsPage";

const meta = {
  title: "Features/Intelligence/Settings",
  component: AISettingsPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-01-01T00:00:00Z" } },
  decorators: [(Story, context) => <SettingsStory route={context.parameters.route ?? "/settings/ai"} configured={context.parameters.configured ?? true}><Story /></SettingsStory>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Contract-backed UI states using synthetic model and library metadata. These are visual fixtures, not available production model sources." } } },
} satisfies Meta<typeof AISettingsPage>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Overview: Story = {};
export const ServiceUnavailable: Story = { parameters: { configured: false } };
export const Libraries: Story = { parameters: { route: "/settings/ai?view=libraries", configured: true } };
export const ConfigurationPrototype: Story = {
  parameters: {
    docs: {
      description: {
        story: "Accepted configuration prototype. It renders the production component with synthetic contract data so the prototype and shipped UI cannot drift.",
      },
    },
  },
};

function SettingsStory({ children, route, configured }: { children: ReactNode; route: string; configured: boolean }) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity } } });
    const libraries = [{ id: "lib_family", name: "家庭影像", status: "ready" }, { id: "lib_travel", name: "旅行记录", status: "ready" }];
    client.setQueryData(["libraries", "list"], { pages: [{ items: libraries, nextCursor: null }], pageParams: [undefined] });
    client.setQueryData(aiKeys.models, configured ? {
      items: [
        { id: "model_semantic_service", active: true, state: "available", storageMode: "service", purpose: "semantic_image_text", version: "foliopath-ml-v1", architecture: "amd64 / arm64", packageSizeBytes: 640_000_000, licenseId: "MIT", availabilityRevision: 1 },
        { id: "model_face_service", active: true, state: "available", storageMode: "service", purpose: "face_detection_embedding", version: "foliopath-face-v1", architecture: "amd64 / arm64", packageSizeBytes: 326_000_000, licenseId: "MIT", availabilityRevision: 1 },
      ],
      activeModelId: "model_semantic_service",
      activeFaceModelId: "model_face_service",
      revision: 1,
    } : { items: [], activeModelId: null, activeFaceModelId: null, revision: 1 });
    client.setQueryData(aiKeys.proxy, { mode: "inherit", proxyUrl: "", revision: 1, deploymentConfigured: true, effectiveMode: "proxy" });
    for (const [index, library] of libraries.entries()) {
      const data = { libraryId: library.id, enabled: configured, state: configured ? "building" : "disabled", revision: 1, activeGenerationId: null, etag: '"semantic-r1"', coverage: { completed: configured ? 1240 + index * 120 : 0, eligible: 2490 + index * 150, failed: 0, stale: 0, complete: false, revision: 1 } };
      client.setQueryData(aiKeys.semantic(library.id), { ...data, coverageRecorded: configured });
      client.setQueryData(aiKeys.videoStatus(library.id), { libraryId: library.id, enabled: configured, offline: false, activeGenerationId: null, coverage: null });
      client.setQueryData(aiKeys.faces(library.id), { ...data, enabled: configured, state: configured ? "ready" : "disabled" });
    }
    client.setQueryData(aiKeys.downloads, { runtimeAvailable: configured, items: [] });
    return client;
  });
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[route]}>{children}<aside role="note" style={{ background: "var(--color-surface)", color: "var(--color-text-muted)", padding: "var(--space-2)", fontSize: "var(--font-size-2)" }}>组件预览 · 合成数据 / Component preview · synthetic data</aside></MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
