import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  getSettings,
  updateSettings,
} from "../../lib/api/settings";

export const settingsKeys = {
  all: ["settings"] as const,
  detail: ["settings", "detail"] as const,
};

export function useSettingsQuery() {
  return useQuery({
    queryKey: settingsKeys.detail,
    queryFn: getSettings,
    staleTime: 15_000,
  });
}

export function useUpdateSettingsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateSettings,
    onSuccess: (settings) => {
      queryClient.setQueryData(settingsKeys.detail, settings);
    },
  });
}
