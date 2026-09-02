import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "./client";
import { applyFaceReview, getLibraryFaceSettings, listAITagSuggestions, listFaceClusters, reviewAITagSuggestions } from "./ai";

vi.mock("./client", () => ({ apiClient: { GET: vi.fn(), POST: vi.fn(), PATCH: vi.fn() } }));

describe("intelligence API adapters", () => {
  beforeEach(() => { vi.mocked(apiClient.GET).mockReset(); vi.mocked(apiClient.POST).mockReset(); });

  it("binds tag suggestion cursors to library and status", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({ data: { items: [], nextCursor: null, coverage: coverage() }, response: new Response() } as never);
    await listAITagSuggestions({ cursor: "opaque", libraryId: "lib-1", status: "pending" });
    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/tag-suggestions", { params: { path: { libraryId: "lib-1" }, query: { cursor: "opaque", limit: 50, status: "pending" } } });
  });

  it("submits exact suggestion and curation revisions with an idempotency key", async () => {
    vi.mocked(apiClient.POST).mockResolvedValue({ data: { items: [] }, response: new Response() } as never);
    await reviewAITagSuggestions({ action: "accept", csrfToken: "csrf", idempotencyKey: "key", items: [{ curationRevision: 7, suggestion: { id: "s1", revision: 3 } as never }] });
    expect(apiClient.POST).toHaveBeenCalledWith("/api/v1/ai/tag-suggestion-reviews", expect.objectContaining({ body: { items: [{ action: "accept", expectedCurationRevision: 7, expectedSuggestionRevision: 3, suggestionId: "s1" }] }, params: { header: { "Idempotency-Key": "key" } } }));
  });

  it("never sends paths or face geometry in a face review", async () => {
    vi.mocked(apiClient.POST).mockResolvedValue({ data: { action: "exclude_face", affectedPersonIds: [], reviewId: "r1", revision: 1, undoable: true }, response: new Response() } as never);
    await applyFaceReview({ csrfToken: "csrf", idempotencyKey: "key", review: { action: "ExcludeFaceReview", faceId: "f1", expectedFaceRevision: 2 } });
    const body = (vi.mocked(apiClient.POST).mock.calls[0]?.[1] as { body?: unknown } | undefined)?.body;
    expect(body).toEqual({ action: "ExcludeFaceReview", faceId: "f1", expectedFaceRevision: 2 });
    expect(JSON.stringify(body)).not.toMatch(/path|region|embedding|crop/i);
  });

  it("uses privacy-safe anonymous cluster filters", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({ data: { items: [], nextCursor: null, coverage: coverage(), groupAssignmentAllowed: false }, response: new Response() } as never);
    await listFaceClusters({ kind: "edge", libraryId: "lib-1" });
    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/face-clusters", { params: { path: { libraryId: "lib-1" }, query: { kind: "edge", limit: 50 } } });
  });

  it("requires the server validator for face settings", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({ data: { libraryId: "lib-1", enabled: false, state: "disabled", revision: 1, activeGenerationId: null, coverage: coverage() }, response: new Response(null, { headers: { ETag: '"face-r1"' } }) } as never);
    await expect(getLibraryFaceSettings("lib-1")).resolves.toMatchObject({ etag: '"face-r1"' });
  });
});

function coverage() { return { complete: false, completed: 0, degraded: 0, eligible: 1, failed: 0, revision: 1, stale: 1 }; }
