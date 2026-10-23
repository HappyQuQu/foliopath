import { beforeEach, expect, it, vi } from "vitest";
import { getAutomaticTagSettings, requestAutomaticTagJob, updateAutomaticTagSettings } from "./ai";
import { addManualAssetTag } from "./curation";
import { apiClient } from "./client";

vi.mock("./client", () => ({ apiClient: { GET: vi.fn(), PUT: vi.fn(), POST: vi.fn() } }));
beforeEach(() => vi.clearAllMocks());

it("scopes automatic settings to a library and preserves the exact ETag", async () => {
  vi.mocked(apiClient.GET).mockResolvedValue({ data: { libraryId: "lib_2", enabled: true }, response: new Response(null, { headers: { ETag: '"autotag-library-2-r3"' } }) } as never);
  await expect(getAutomaticTagSettings("lib_2")).resolves.toMatchObject({ libraryId: "lib_2", etag: '"autotag-library-2-r3"' });
  expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/automatic-tags", { params: { path: { libraryId: "lib_2" } } });
});

it("does not silently retry a conflicting enable/disable update", async () => {
  vi.mocked(apiClient.PUT).mockResolvedValue({ error: { error: { code: "precondition_failed", message: "Changed" } }, response: new Response(null, { status: 412 }) } as never);
  await expect(updateAutomaticTagSettings({ libraryId: "lib_2", enabled: false, etag: '"autotag-library-2-r3"', csrfToken: "csrf" })).rejects.toMatchObject({ code: "precondition_failed" });
  expect(apiClient.PUT).toHaveBeenCalledOnce();
  expect(apiClient.PUT).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/automatic-tags", { body: { enabled: false }, headers: { "X-CSRF-Token": "csrf" }, params: { path: { libraryId: "lib_2" }, header: { "If-Match": '"autotag-library-2-r3"' } } });
});

it.each(["missing", "all"] as const)("requests %s with explicit full-run confirmation and no candidate vocabulary", async mode => {
  vi.mocked(apiClient.POST).mockResolvedValue({ data: { id: "aio_1" }, response: new Response(null, { headers: { ETag: '"aio_1-r1"' } }) } as never);
  await requestAutomaticTagJob({ libraryId: "lib_2", mode, csrfToken: "csrf", idempotencyKey: "unique-key" });
  expect(apiClient.POST).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/automatic-tags/jobs", { body: { mode, ...(mode === "all" ? { confirmation: "reprocess_automatic_tags" } : {}) }, headers: { "X-CSRF-Token": "csrf" }, params: { path: { libraryId: "lib_2" }, header: { "Idempotency-Key": "unique-key" } } });
});

it("explicit manual ownership uses a no-body CAS endpoint", async () => {
  vi.mocked(apiClient.PUT).mockResolvedValue({ data: { assetId: "ast_1", tags: [{ id: "tag_2", source: "manual" }] }, response: new Response(null) } as never);
  await expect(addManualAssetTag({ assetId: "ast_1", tagId: "tag_2", revision: 4, csrfToken: "csrf" })).resolves.toMatchObject({ tags: [{ source: "manual" }] });
  expect(apiClient.PUT).toHaveBeenCalledWith("/api/v1/assets/{assetId}/tags/{tagId}", { headers: { "X-CSRF-Token": "csrf" }, params: { path: { assetId: "ast_1", tagId: "tag_2" }, header: { "If-Match": '"curation-r4"' } } });
});
