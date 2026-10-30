import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { Button, ToastProvider } from "../../components/ui";
import type { DeletionOperation, DeletionPreflight, DeletionSettings } from "../../lib/api/deletion";
import { useDeletionFlow } from "./useDeletionFlow";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { FileManagementPage } from "./FileManagementPage";

const session = { administrator: { id: "adm_story", username: "synthetic", displayName: "合成管理员" }, csrfToken: "synthetic-story-csrf-not-a-real-credential", expiresAt: "2099-01-01T00:00:00Z" };
const settings: DeletionSettings = { enabled: true, available: true, unavailableReason: null, recycleBinEnabled: false, retentionDays: null, revision: 1 };
const operation: DeletionOperation = { id: "deletion_synthetic", kind: "permanent", state: "partial", createdAt: "2026-10-30T01:30:00Z", updatedAt: "2026-10-30T01:32:00Z", revision: 3, canStop: false, canResume: true, totalItems: 2, completedItems: 1, failedItems: 1, itemsNextCursor: null, items: [{ id: "deletion_item_synthetic", libraryId: "lib_story", libraryName: "合成旅行库", name: "京都相册", relativePath: "旅行/日本京都/京都相册", kind: "directory", state: "partial", removedEntries: 256, errorCode: "permission_denied", trashEntryId: null }] };
operation.items.push({ ...operation.items[0]!, id: "deletion_item_completed", name: "海边相册", relativePath: "旅行/海边相册", state: "deleted", removedEntries: 4, errorCode: null });
const originalFetch = globalThis.fetch.bind(globalThis);
function FixtureBoundary({ children, unavailable, empty, mode, blocked }: { children: ReactNode; unavailable: boolean; empty: boolean; mode: "permanent"; blocked: boolean }) {
  const [fixture] = useState(() => ({ policy: { ...settings, recycleBinEnabled: false }, task: { ...operation } }));
  globalThis.fetch = async (input, init) => {
    const request = input instanceof Request ? input : new Request(new URL(String(input), location.origin), init);
    const path = new URL(request.url).pathname;
    if (path === "/api/v1/deletion/settings") {
      if (request.method === "PATCH") { const body = await request.json(); fixture.policy = { ...fixture.policy, enabled: body.enabled, recycleBinEnabled: body.recycleBinEnabled, retentionDays: body.retentionDays, revision: fixture.policy.revision + 1 }; }
      return Response.json(unavailable ? { ...fixture.policy, enabled: false, available: false, unavailableReason: "unsupported" } : fixture.policy);
    }
    if (path === "/api/v1/deletion/preflight") {
      const preview: DeletionPreflight = { id: blocked ? null : "synthetic-preview", mode, retentionDays: null, canSubmit: !blocked, expiresAt: blocked ? null : "2099-01-01T00:00:00Z", items: [{ target: { type: "directory", id: "dir_story" }, libraryId: "lib_story", libraryName: "合成旅行库", name: "京都相册", relativePath: "旅行/京都相册", kind: "directory", blockedReason: blocked ? "busy" : null }] };
      return Response.json(preview);
    }
    if (path === "/api/v1/deletion/operations") {
      if (request.method === "POST") return Response.json({ ...fixture.task, kind: mode, state: "queued", canResume: false, canStop: true }, { status: 202 });
      return Response.json({ items: empty ? [] : [fixture.task], nextCursor: null });
    }
    if (path.startsWith("/api/v1/deletion/operations/")) {
      if (request.method === "POST") fixture.task = { ...fixture.task, revision: fixture.task.revision + 1, state: path.endsWith("stop") ? "stopped" : "queued", canStop: path.endsWith("resume"), canResume: path.endsWith("stop") };
      return Response.json(fixture.task);
    }
    if (path === "/api/v1/libraries") return Response.json({ items: [{ id: "lib_story", name: "合成旅行库", status: "ready", assetCount: 0, directoryCount: 0, automaticDiscoveryErrorCode: null, automaticDiscoveryStatus: "disabled", contentRevision: 1, displayPath: "/synthetic", lastSuccessfulScanAt: null, latestScanId: null, lastAutomaticDiscoveryAt: null }], nextCursor: null });
    if (path === "/api/v1/releases") return Response.json({ currentVersion: "synthetic", latestVersion: null, releases: [], checkedAt: "2026-10-30T00:00:00Z", updateAvailable: false });
    if (path === "/api/v1/deletion/settings/retention-preview") return Response.json({ expiredEntries: 1 });
    return originalFetch(input, init);
  };
  useEffect(() => () => { globalThis.fetch = originalFetch; }, []);
  return children;
}
function ConfirmationExample() {
  const flow = useDeletionFlow(session);
  useEffect(() => { flow.start([{ type: "directory", id: "dir_story" }]); }, []);
  return <><Button onClick={() => flow.start([{ type: "directory", id: "dir_story" }])}>打开删除确认（合成合同夹具）</Button>{flow.dialog}</>;
}
function Example({ view = "tasks", unavailable = false, empty = false, confirmation = false, mode = "permanent", blocked = false }: { view?: string; unavailable?: boolean; empty?: boolean; confirmation?: boolean; mode?: "permanent"; blocked?: boolean }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  return <FixtureBoundary unavailable={unavailable} empty={empty} mode={mode} blocked={blocked}><ThemeProvider><QueryClientProvider client={client}><ToastProvider><MemoryRouter initialEntries={[`/files?view=${view}`]}>{confirmation ? <ConfirmationExample /> : <FileManagementPage session={session} />}</MemoryRouter></ToastProvider></QueryClientProvider></ThemeProvider></FixtureBoundary>;
}
const meta = { title: "Features/Deletion/File management", component: Example, parameters: { layout: "fullscreen", docs: { description: { component: "Production components with synthetic API contract fixtures only. No media or deployment is modified. Permanent deletion, restoration conflicts, bounded pages and task states consume the accepted backend contract." } } } } satisfies Meta<typeof Example>;
export default meta;
type Story = StoryObj<typeof meta>;
export const EmptyTasks: Story = { args: { view: "tasks", empty: true } };
export const PartialTask: Story = { args: { view: "tasks" } };
export const Settings: Story = { args: { view: "settings" } };
export const UnsupportedSystem: Story = { args: { view: "settings", unavailable: true } };

export const PermanentConfirmation: Story = { args: { confirmation: true, mode: "permanent" } };
export const BlockedSelection: Story = { args: { confirmation: true, blocked: true } };

export const AcceptedTask: Story = {
  args: { confirmation: true },
  play: async ({ canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const submit = await body.findByRole("button", { name: /^(永久删除|Delete permanently)$/ });
    await waitFor(() => expect(submit).toBeEnabled());
    await userEvent.click(submit);
    await expect(await body.findByRole("button", { name: /^(查看文件任务|View file tasks)$/ })).toBeVisible();
  },
};
