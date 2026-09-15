import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "./client";
import { getLibraryVideoSemanticStatus } from "./ai";
import { requestLibraryVideoSemanticJob } from "./ai";
import { applyFaceReview, downloadAIModel, listAIModelDownloads, getLibraryFaceSettings, listAITagSuggestions, listFaceClusters, reviewAITagSuggestions } from "./ai";

vi.mock("./client", () => ({ apiClient: { GET: vi.fn(), POST: vi.fn(), PATCH: vi.fn() } }));

describe("intelligence API adapters", () => {
  beforeEach(() => { vi.mocked(apiClient.GET).mockReset(); vi.mocked(apiClient.POST).mockReset(); });

  it("requests video work by library ID with CSRF and idempotency", async () => {
    vi.mocked(apiClient.POST).mockResolvedValue({ data: { id: "aio_video", revision: 2 }, response: new Response(null, { headers: { ETag: '"aio_video-r2"' } }) } as never);
    await expect(requestLibraryVideoSemanticJob({ libraryId: "lib_1", mode: "all", csrfToken: "csrf", idempotencyKey: "video-key" })).resolves.toMatchObject({ id: "aio_video" });
    expect(apiClient.POST).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/video-semantic/jobs", { body: { mode: "all" }, headers: { "X-CSRF-Token": "csrf" }, params: { header: { "Idempotency-Key": "video-key" }, path: { libraryId: "lib_1" } } });
  });

  it("reads video progress without a query or inference request and preserves unknown coverage", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({ data: { libraryId: "lib_1", enabled: false, offline: true, activeGenerationId: null, coverage: null }, response: new Response() } as never);
    await expect(getLibraryVideoSemanticStatus("lib_1")).resolves.toMatchObject({ coverage: null, offline: true });
    expect(apiClient.GET).toHaveBeenCalledWith("/api/v1/libraries/{libraryId}/ai/video-semantic", { params: { path: { libraryId: "lib_1" } } });
    expect(apiClient.POST).not.toHaveBeenCalled();
  });

  it("downloads by ID without sending a URL or body and preserves the operation validator", async () => {
    vi.mocked(apiClient.POST).mockResolvedValue({ data: { id: "aio_download", revision: 3 }, response: new Response(null, { headers: { ETag: '\"aio_download-r3\"' } }) } as never);
    await expect(downloadAIModel({ downloadId: "download_vision", csrfToken: "csrf", idempotencyKey: "download_key" })).resolves.toMatchObject({ etag: '\"aio_download-r3\"' });
    expect(apiClient.POST).toHaveBeenCalledWith("/api/v1/ai/model-downloads/{downloadId}/install", { headers: { "X-CSRF-Token": "csrf" }, params: { header: { "Idempotency-Key": "download_key" }, path: { downloadId: "download_vision" } } });
  });

  it("restores a cancellable snapshot from server download history", async () => {
    vi.mocked(apiClient.GET).mockResolvedValue({ data: { runtimeAvailable: false, items: [{ id: "download_vision", operation: { id: "aio_download", revision: 7 } }, { id: "download_second", operation: null }] }, response: new Response() } as never);
    await expect(listAIModelDownloads()).resolves.toMatchObject({ runtimeAvailable: false, items: [{ operation: { etag: '\"aio_download-r7\"' } }, { operation: null }] });
  });

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
