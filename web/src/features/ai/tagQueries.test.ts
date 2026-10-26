import { beforeEach, expect, it, vi } from "vitest";

import { getAITagVocabulary, replaceAITagVocabulary, requestAITagSuggestionJob, type AIOperationSnapshot, type AITagVocabularySnapshot } from "../../lib/api/ai";
import { aiKeys } from "./queries";
import { useAITagVocabularyQuery, useReplaceAITagVocabularyMutation, useRequestAITagSuggestionJobMutation } from "./tagQueries";

const cache = vi.hoisted(() => ({ setQueryData: vi.fn(), invalidateQueries: vi.fn() }));
vi.mock("@tanstack/react-query", () => ({
  skipToken: Symbol("skipToken"),
  useQuery: (options: unknown) => options,
  useQueries: (options: unknown) => options,
  useMutation: (options: unknown) => options,
  useQueryClient: () => cache,
}));

beforeEach(() => vi.clearAllMocks());

it("reads the shared vocabulary through its canonical key and domain adapter", () => {
  expect(useAITagVocabularyQuery()).toMatchObject({ queryKey: aiKeys.tagVocabulary, queryFn: getAITagVocabulary, staleTime: 30_000 });
});

it("publishes the returned revision and invalidates suggestions across all libraries", () => {
  const options = useReplaceAITagVocabularyMutation() as unknown as { mutationFn: unknown; onSuccess: (value: AITagVocabularySnapshot) => void };
  const snapshot: AITagVocabularySnapshot = { revision: 2, etag: '"ai-tag-vocabulary-r2"', publishedAt: "2026-10-22T00:00:00Z", entries: [{ tagId: "tag_1", name: "旅行" }] };
  expect(options.mutationFn).toBe(replaceAITagVocabulary);
  options.onSuccess(snapshot);
  expect(cache.setQueryData).toHaveBeenCalledWith(aiKeys.tagVocabulary, snapshot);
  expect(cache.invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: aiKeys.suggestionsRoot });
});

it("seeds the server operation so the shared operation tracker owns progress", () => {
  const options = useRequestAITagSuggestionJobMutation() as unknown as { mutationFn: unknown; onSuccess: (value: AIOperationSnapshot) => void };
  const operation: AIOperationSnapshot = { id: "aio_tags", kind: "tag_suggestion_missing", state: "queued", phase: "queued", revision: 1, completedItems: 0, totalItems: null, errorCode: null, createdAt: "2026-10-22T00:00:00Z", updatedAt: "2026-10-22T00:00:00Z", etag: '"aio_tags-r1"' };
  expect(options.mutationFn).toBe(requestAITagSuggestionJob);
  options.onSuccess(operation);
  expect(cache.setQueryData).toHaveBeenCalledWith(aiKeys.operation(operation.id), operation);
  expect(cache.invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: aiKeys.suggestionsRoot });
});
