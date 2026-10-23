import { skipToken, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import {
  getAITagVocabulary,
  replaceAITagVocabulary,
  requestAITagSuggestionJob,
  type AIOperationSnapshot,
} from "../../lib/api/ai";
import { aiKeys, operationIsActive } from "./queries";

export function useAITagVocabularyQuery() {
  return useQuery({
    queryKey: aiKeys.tagVocabulary,
    queryFn: getAITagVocabulary,
    staleTime: 30_000,
  });
}

export function useReplaceAITagVocabularyMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: replaceAITagVocabulary,
    onSuccess: (snapshot) => {
      client.setQueryData(aiKeys.tagVocabulary, snapshot);
      void client.invalidateQueries({ queryKey: aiKeys.suggestionsRoot });
    },
  });
}

export function useRequestAITagSuggestionJobMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requestAITagSuggestionJob,
    onSuccess: (operation) => {
      client.setQueryData(aiKeys.operation(operation.id), operation);
      void client.invalidateQueries({ queryKey: aiKeys.suggestionsRoot });
    },
  });
}

// The shared operation tracker owns network polling. This passive observer only
// refreshes the mounted task card's coverage while its submitted job is active.
export function useAITagJobCoverageRefresh(libraryId: string, operationId: string | null) {
  const client = useQueryClient();
  const operation = useQuery<AIOperationSnapshot>({
    queryKey: aiKeys.operation(operationId ?? ""),
    queryFn: skipToken,
  });
  const active = Boolean(operationId && !operation.isError && operation.data && operationIsActive(operation.data));
  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState === "hidden") return;
      void client.invalidateQueries({ queryKey: aiKeys.suggestions(libraryId, "pending"), exact: true, refetchType: "active" }, { cancelRefetch: false });
    }, 1_500);
    return () => window.clearInterval(timer);
  }, [active, client, libraryId]);
}
