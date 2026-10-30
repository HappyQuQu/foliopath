import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { userEvent, within } from "storybook/test";
import type { DeletionOperation } from "../../lib/api/deletion";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { NotificationCenter } from "./NotificationCenter";

const task: DeletionOperation = { id: "deletion_1", kind: "permanent", state: "succeeded", createdAt: "2026-10-30T01:30:00Z", updatedAt: "2026-10-30T01:31:00Z", revision: 2, canStop: false, canResume: false, totalItems: 1, completedItems: 1, failedItems: 0, itemsNextCursor: null, items: [{ id: "deletion_item_1", libraryId: "lib_story", libraryName: "Synthetic library / 合成媒体库", name: "旅行相册 · Synthetic album", relativePath: "旅行相册", kind: "directory", state: "deleted", removedEntries: 20, errorCode: null, trashEntryId: null }] };
function Example() {
  const [originalFetch] = useState(() => globalThis.fetch);
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  globalThis.fetch = async (input, init) => {
    const request = input instanceof Request ? input : new Request(new URL(String(input), location.origin), init);
    const path = new URL(request.url).pathname;
    if (path === "/api/v1/deletion/operations") return Response.json({ items: [task, { ...task, id: "deletion_2", state: "partial", completedItems: 0, failedItems: 1 }, { ...task, id: "deletion_3", state: "failed", completedItems: 0, failedItems: 1 }, { ...task, id: "deletion_4", state: "running", completedItems: 0, canStop: true }], nextCursor: null });
    if (path === "/api/v1/libraries") return Response.json({ items: [], nextCursor: null });
    if (path === "/api/v1/diagnostics/media-failures") return Response.json({ items: [], nextCursor: null, revision: null });
    if (path === "/api/v1/releases") return Response.json({ currentVersion: "synthetic", latestVersion: null, releases: [], checkedAt: "2026-10-30T00:00:00Z", updateAvailable: false });
    return originalFetch(input, init);
  };
  useEffect(() => () => { globalThis.fetch = originalFetch; client.clear(); }, [client, originalFetch]);
  return <ThemeProvider><QueryClientProvider client={client}><MemoryRouter><div style={{ display: "flex", justifyContent: "flex-end", alignItems: "flex-start", minHeight: "100vh", padding: "var(--space-6)" }}><NotificationCenter /></div></MemoryRouter></QueryClientProvider></ThemeProvider>;
}
const meta = { title: "Features/Notifications/Deletion results", component: Example, globals: { a11y: { manual: true } }, parameters: { layout: "fullscreen", docs: { description: { component: "Production notification center with labelled synthetic deletion API fixtures. No real media, account or file operation is accessed." } } } } satisfies Meta<typeof Example>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Results: Story = { play: async ({ canvasElement }) => { await userEvent.click(await within(canvasElement).findByRole("button", { name: /消息中心|Notifications/ })); } };
