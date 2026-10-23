import { beforeEach, expect, it, vi } from "vitest";

import { getAITagVocabulary, replaceAITagVocabulary, requestAITagSuggestionJob } from "./ai";
import { apiClient } from "./client";

vi.mock("./client", () => ({ apiClient: { GET: vi.fn(), PUT: vi.fn(), POST: vi.fn() } }));

beforeEach(() => vi.clearAllMocks());

it("preserves the server validator for the shared tag vocabulary", async () => {
  vi.mocked(apiClient.GET).mockResolvedValue({ data: { revision: 4, entries: [{ tagId: "tag_1", name: "旅行" }], publishedAt: "2026-10-22T00:00:00Z" }, response: new Response(null, { headers: { ETag: '"ai-tag-vocabulary-r4"' } }) } as never);
  await expect(getAITagVocabulary()).resolves.toMatchObject({ etag: '"ai-tag-vocabulary-r4"', revision: 4 });
  expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/ai/tag-vocabulary");
});

it("publishes only opaque existing tag IDs with CSRF and the exact vocabulary revision", async () => {
  vi.mocked(apiClient.PUT).mockResolvedValue({ data: { revision: 5, entries: [{ tagId: "tag_2", name: "风景" }], publishedAt: "2026-10-22T00:01:00Z" }, response: new Response(null, { headers: { ETag: '"ai-tag-vocabulary-r5"' } }) } as never);
  await expect(replaceAITagVocabulary({ csrfToken: "csrf", etag: '"ai-tag-vocabulary-r4"', tagIds: ["tag_2"] })).resolves.toMatchObject({ etag: '"ai-tag-vocabulary-r5"' });
  expect(apiClient.PUT).toHaveBeenCalledWith("/api/v1/ai/tag-vocabulary", {
    body: { tagIds: ["tag_2"] },
    headers: { "X-CSRF-Token": "csrf" },
    params: { header: { "If-Match": '"ai-tag-vocabulary-r4"' } },
  });
  expect(apiClient.POST).not.toHaveBeenCalled();
});

it("keeps revision conflicts as failures without replaying an unconditional save", async () => {
  vi.mocked(apiClient.PUT).mockResolvedValue({ error: { error: { code: "precondition_failed", message: "Changed" } }, response: new Response(null, { status: 412 }) } as never);
  await expect(replaceAITagVocabulary({ csrfToken: "csrf", etag: '"ai-tag-vocabulary-r4"', tagIds: [] })).rejects.toMatchObject({ code: "precondition_failed", status: 412 });
  expect(apiClient.PUT).toHaveBeenCalledOnce();
});

it.each(["missing", "all"] as const)("requests %s suggestions only for the selected library", async (mode) => {
  vi.mocked(apiClient.POST).mockResolvedValue({ data: { id: "aio_tags", revision: 1, kind: mode === "all" ? "tag_suggestion_rebuild" : "tag_suggestion_missing" }, response: new Response(null, { headers: { ETag: '"aio_tags-r1"' } }) } as never);
  await expect(requestAITagSuggestionJob({ csrfToken: "csrf", idempotencyKey: "tag-job-key", libraryId: "lib_2", mode })).resolves.toMatchObject({ id: "aio_tags", etag: '"aio_tags-r1"' });
  expect(apiClient.POST).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/tag-suggestions/jobs", {
    body: { mode }, headers: { "X-CSRF-Token": "csrf" }, params: { header: { "Idempotency-Key": "tag-job-key" }, path: { libraryId: "lib_2" } },
  });
});
