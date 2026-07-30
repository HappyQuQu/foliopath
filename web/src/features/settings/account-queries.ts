import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  changeAccountPassword,
  getAccount,
  updateAccount,
} from "../../lib/api/account";
import type { AuthenticatedSession } from "../../lib/api/auth";
import { authKeys } from "../auth";

export const accountKeys = {
  detail: ["account", "detail"] as const,
};

export function useAccountQuery() {
  return useQuery({
    queryKey: accountKeys.detail,
    queryFn: getAccount,
    staleTime: 15_000,
  });
}

export function useUpdateAccountMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: updateAccount,
    onSuccess: (account) => {
      queryClient.setQueryData(accountKeys.detail, account);
      queryClient.setQueryData<AuthenticatedSession>(
        authKeys.session,
        (current) =>
          current
            ? {
                ...current,
                administrator: {
                  ...current.administrator,
                  displayName: account.displayName,
                },
              }
            : current,
      );
    },
  });
}

export function useChangePasswordMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: changeAccountPassword,
    onSuccess: (etag) => {
      queryClient.setQueryData(accountKeys.detail, (current: unknown) =>
        typeof current === "object" && current !== null
          ? { ...current, etag }
          : current,
      );
    },
  });
}
