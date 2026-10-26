import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { getAutomaticTagSettings, requestAutomaticTagJob, updateAutomaticTagSettings, type AutomaticTagSnapshot } from "../../lib/api/ai";
import { curationKeys } from "../curation";
import { aiKeys } from "./queries";

export function useAutomaticTagSettings(libraryId: string) {
  const client = useQueryClient();
  const previous = useRef<{ libraryId: string; revision: number } | null>(null);
  const query = useQuery({
    queryKey: aiKeys.automaticTags(libraryId), queryFn: () => getAutomaticTagSettings(libraryId), enabled: Boolean(libraryId),
    refetchInterval: q => q.state.data?.activeOperationId ? 2_000 : 10_000,
  });
  const revision = query.data?.coverage.revision;
  useEffect(() => {
    if (revision === undefined) return;
    if (previous.current?.libraryId === libraryId && previous.current.revision !== revision) {
      void client.invalidateQueries({ queryKey: curationKeys.all });
    }
    previous.current = { libraryId, revision };
  }, [client, libraryId, revision]);
  return query;
}
export function useUpdateAutomaticTags() {
  const client = useQueryClient();
  return useMutation({ mutationFn: updateAutomaticTagSettings,
    onSuccess: async value => {
      await client.cancelQueries({ queryKey: aiKeys.automaticTags(value.libraryId), exact: true });
      client.setQueryData<AutomaticTagSnapshot>(aiKeys.automaticTags(value.libraryId), current => {
        // A slower mutation response must not replace a newer settings/coverage
        // snapshot already returned by polling or another tab's update.
        if (current && (current.revision > value.revision || current.revision === value.revision && current.coverage.revision > value.coverage.revision)) return current;
        return value;
      });
    },
    onError: (_error, input) => { void client.invalidateQueries({ queryKey: aiKeys.automaticTags(input.libraryId) }); },
  });
}
export function useRequestAutomaticTags() {
  const client = useQueryClient();
  return useMutation({ mutationFn: requestAutomaticTagJob,
    onSuccess: (value, input) => {
      client.setQueryData(aiKeys.operation(value.id), value);
      void client.invalidateQueries({ queryKey: aiKeys.automaticTags(input.libraryId) });
    },
    // The server may have admitted work before a transport failure. Read the
    // authoritative operation reference instead of assuming nothing happened.
    onError: (_error, input) => { void client.invalidateQueries({ queryKey: aiKeys.automaticTags(input.libraryId) }); },
  });
}
