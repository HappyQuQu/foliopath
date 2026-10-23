import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { curationKeys } from "../../curation";
import { aiKeys } from "../queries";
import { AISettingsPage } from "./AISettingsPage";
import { ApiError } from "../../../lib/api/errors";
import { readAIOperationIds, writeAIOperationIds } from "../../../lib/storage/preferences";

const meta = {
  title: "Features/Intelligence/Settings",
  component: AISettingsPage,
  args: { session: { administrator: { displayName: "管理员", id: "adm-story", username: "admin" }, csrfToken: "storybook-csrf-token-that-is-long-enough", expiresAt: "2099-01-01T00:00:00Z" } },
  decorators: [(Story, context) => <SettingsStory route={context.parameters.route ?? "/settings/ai"} configured={context.parameters.configured ?? true} automaticState={context.parameters.automaticState ?? "ready"} taskPrototype={context.parameters.taskPrototype ?? false} taskStatusUnavailable={context.parameters.taskStatusUnavailable ?? false} faceFailures={context.parameters.faceFailures ?? false} historicalFaces={context.parameters.historicalFaces ?? false}><Story /></SettingsStory>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Contract-backed UI states using synthetic model and library metadata. These are visual fixtures, not available production model sources." } } },
} satisfies Meta<typeof AISettingsPage>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Overview: Story = {};
export const ServiceUnavailable: Story = { parameters: { configured: false } };
export const Libraries: Story = { parameters: { route: "/settings/ai?view=libraries", configured: true } };
export const ProcessingJobs: Story = { parameters: { route: "/settings/ai?view=tasks&library=lib_travel" } };
export const TaskWorkspacePrototype: Story = { parameters: { route: "/settings/ai?view=tasks&library=lib_4", taskPrototype: true, docs: { description: { story: "Accepted task-list direction. Production component, read-only synthetic contract data; no production requests or simulated successful writes." } } } };
export const TaskStatusUnavailable: Story = { parameters: { route: "/settings/ai?view=tasks&library=lib_4", taskPrototype: true, taskStatusUnavailable: true, docs: { description: { story: "Synthetic status refresh failures. Stale progress is hidden, writes stay disabled, and retry keeps the explicit failure fixture." } } } };
export const FaceFailureDetails: Story = { parameters: { route: "/settings/ai?view=tasks&library=lib_4", taskPrototype: true, faceFailures: true, docs: { description: { story: "Production diagnostic component, synthetic interrupted task plus one recorded and one historical unrecorded failure. Read-only fixture, not actual model results." } } } };
export const HistoricalFaceResults: Story = { parameters: { route: "/settings/ai?view=tasks&library=lib_4", taskPrototype: true, historicalFaces: true, docs: { description: { story: "Production component with synthetic historical generation coverage. Counts are not presented as current-model completion; no writes are passed to production." } } } };
export const ServiceAndModels: Story = { parameters: { route: "/settings/ai?view=service" } };
export const UnknownLibrary: Story = { parameters: { route: "/settings/ai?library=lib_missing" } };
export const AutomaticTagsProcessing: Story = { parameters: { automaticState: "building" } };
export const AutomaticTagsAwaitingModel: Story = { parameters: { automaticState: "awaiting_model" } };
export const AutomaticTagsStopped: Story = { parameters: { automaticState: "degraded" } };
export const ConfigurationPrototype: Story = {
  parameters: {
    docs: {
      description: {
        story: "Accepted configuration prototype. It renders the production component with synthetic contract data so the prototype and shipped UI cannot drift.",
      },
    },
  },
};

function SettingsStory({ children, route, configured, automaticState, taskPrototype, taskStatusUnavailable, faceFailures, historicalFaces }: { children: ReactNode; route: string; configured: boolean; automaticState: "ready" | "building" | "awaiting_model" | "degraded"; taskPrototype: boolean; taskStatusUnavailable: boolean; faceFailures: boolean; historicalFaces: boolean }) {
  const [originalRecords] = useState(readAIOperationIds);
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { enabled: false, retry: false, staleTime: Infinity } } });
    const libraries = taskPrototype ? [{ id: "lib_4", name: "pixiv", status: "ready" }, { id: "lib_travel", name: "旅行记录", status: "ready" }] : [{ id: "lib_family", name: "家庭影像", status: "ready" }, { id: "lib_travel", name: "旅行记录", status: "ready" }];
    if (taskPrototype) {
      writeAIOperationIds(["aio_story_missing_1", "aio_story_missing_2"]);
      for (const id of ["aio_story_missing_1", "aio_story_missing_2"]) client.getQueryCache().build(client, { queryKey: aiKeys.operation(id) }).setState({ status: "error", fetchStatus: "idle", error: new ApiError({ status: 404, code: "ai_operation_not_found", message: "Synthetic missing task", requestId: undefined }) });
    }
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
    const tags = [{ id: "tag_1", name: "旅行", assetCount: 12 }, { id: "tag_2", name: "风景", assetCount: 24 }];
    client.setQueryData(aiKeys.tagVocabulary, { revision: 1, entries: tags.map(tag => ({ tagId: tag.id, name: tag.name })), publishedAt: "2026-10-22T00:00:00Z", etag: '"ai-tag-vocabulary-r1"' });
    client.setQueryData(curationKeys.tags, { pages: [{ items: tags, nextCursor: null }], pageParams: [undefined] });
    for (const [index, library] of libraries.entries()) {
      const data = { libraryId: library.id, enabled: configured, state: configured ? "building" : "disabled", revision: 1, activeGenerationId: configured ? `generation_${library.id}` : null, etag: '"semantic-r1"', coverage: { completed: configured ? 1240 + index * 120 : 0, eligible: 2490 + index * 150, failed: 0, stale: 0, complete: false, revision: 1 } };
      client.setQueryData(aiKeys.semantic(library.id), { ...data, coverageRecorded: configured && !taskPrototype });
      client.setQueryData(aiKeys.videoStatus(library.id), { libraryId: library.id, enabled: configured, offline: false, activeGenerationId: null, coverage: null });
      client.setQueryData(aiKeys.faces(library.id), { ...data, enabled: configured, generationCurrent: !historicalFaces, state: taskPrototype ? "degraded" : configured ? "ready" : "disabled", ...(taskPrototype ? { coverage: { ...data.coverage, completed: 4, eligible: 999, failed: faceFailures ? 2 : 0 }, diagnostics: {
        latestTask: { operationId: "aio_story_interrupted", state: "failed", errorCode: "library_not_ready", updatedAt: "2026-10-23T00:00:00Z" },
        recordedFailures: faceFailures ? 1 : 0, unrecordedFailures: faceFailures ? 1 : 0,
        reasons: faceFailures ? [{ code: "invalid_media", count: 1 }] : [],
        samples: faceFailures ? [{ assetId: "ast_fixture", name: "synthetic-empty-image.jpg", code: "invalid_media", sizeBytes: 0, updatedAt: "2026-10-23T00:00:00Z" }] : [],
      } } : {}) });
      client.setQueryData(aiKeys.automaticTags(library.id), { libraryId: library.id, enabled: configured && !taskPrototype, state: configured && !taskPrototype ? automaticState : "disabled", revision: 1, offline: false, activeOperationId: !taskPrototype && automaticState === "building" ? "op_automatic_story" : null,
        model: configured && automaticState !== "awaiting_model" ? { identity: "a".repeat(64), displayName: "JoyTag", protocol: "foliopath-tag-v1" } : null,
        etag: '"autotag-library-1-r1"', coverage: { completed: taskPrototype || !configured || automaticState === "awaiting_model" ? 0 : automaticState === "ready" ? 24 : 12, eligible: taskPrototype ? 999 : 24, failed: !taskPrototype && automaticState === "degraded" ? 1 : 0, stale: 0, revision: 1 } });
      client.setQueryData(aiKeys.suggestions(library.id, "pending"), { items: [], nextCursor: null, coverage: { ...data.coverage, degraded: 0, completed: 0 } });
      if (taskStatusUnavailable) for (const queryKey of [aiKeys.semantic(library.id), aiKeys.faces(library.id), aiKeys.videoStatus(library.id), aiKeys.automaticTags(library.id)]) {
        client.getQueryCache().find({ queryKey, exact: true })!.setState({ status: "error", fetchStatus: "idle", error: new ApiError({ status: 503, code: "internal_error", message: "Synthetic status failure", requestId: "story-fixture" }) });
      }
    }
    client.setQueryData(aiKeys.downloads, { runtimeAvailable: configured, items: [] });
    return client;
  });
  useEffect(() => {
    const original = window.fetch;
    window.fetch = async (input, init) => {
      const url = new URL(input instanceof Request ? input.url : String(input), window.location.origin);
      if (!url.pathname.startsWith("/api/v1/")) return original(input, init);
      const match = /^\/api\/v1\/libraries\/([^/]+)\/ai\/(automatic-tags|semantic|faces|video-semantic)$/.exec(url.pathname);
      const method = init?.method ?? (input instanceof Request ? input.method : "GET");
      if (method === "GET" && match && taskStatusUnavailable) return Response.json({ error: { code: "internal_error", message: "Synthetic status failure", requestId: "story-fixture" } }, { status: 503 });
      const key = match ? match[2] === "semantic" ? aiKeys.semantic(match[1]!) : match[2] === "faces" ? aiKeys.faces(match[1]!) : match[2] === "video-semantic" ? aiKeys.videoStatus(match[1]!) : aiKeys.automaticTags(match[1]!) : url.pathname === "/api/v1/ai/models" ? aiKeys.models : url.pathname === "/api/v1/ai/model-downloads" ? aiKeys.downloads : null;
      const value = key && client.getQueryData<Record<string, unknown>>(key);
      if (method === "GET" && value) {
        const { etag, ...body } = value;
        return Response.json(body, typeof etag === "string" ? { headers: { ETag: etag } } : {});
      }
      if (method === "GET" && url.pathname.startsWith("/api/v1/ai/operations/")) return Response.json({ error: { code: "ai_operation_not_found", message: "Synthetic unavailable task", requestId: "story-fixture" } }, { status: 404 });
      // Keep polling on a stable contract fixture; never pass writes to a real API.
      return Response.json({ error: { code: "precondition_failed", message: "Read-only synthetic snapshot", requestId: "story-fixture" } }, { status: 412 });
    };
    return () => { window.fetch = original; if (taskPrototype) writeAIOperationIds(originalRecords); client.clear(); };
  }, [client, originalRecords, taskPrototype, taskStatusUnavailable]);
  return <ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[route]}>{children}<aside role="note" style={{ background: "var(--color-surface)", color: "var(--color-text-muted)", padding: "var(--space-2)", fontSize: "var(--font-size-2)" }}>只读组件预览 · 合成数据 / Read-only component preview · synthetic data</aside></MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider>;
}
