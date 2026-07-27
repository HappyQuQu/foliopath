import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getAuthenticationStatus,
  getSession,
  login,
  logout,
  setupAdministrator,
  type LoginInput,
  type SetupAdministratorInput,
} from "../../lib/api/auth";

export const authKeys = {
  all: ["auth"] as const,
  session: ["auth", "session"] as const,
  status: ["auth", "status"] as const,
};

export function useAuthenticationStatusQuery() {
  return useQuery({
    queryKey: authKeys.status,
    queryFn: getAuthenticationStatus,
    staleTime: 0,
  });
}

export function useSessionQuery({ enabled = true }: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: authKeys.session,
    queryFn: getSession,
    enabled,
    staleTime: 15_000,
  });
}

export function useLoginMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: LoginInput) => login(input),
    onSuccess: (session) => {
      queryClient.setQueryData(authKeys.session, session);
      void queryClient.invalidateQueries({ queryKey: authKeys.status });
    },
  });
}

export function useSetupAdministratorMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: SetupAdministratorInput) => setupAdministrator(input),
    onSuccess: (session) => {
      queryClient.setQueryData(authKeys.session, session);
      queryClient.setQueryData(authKeys.status, {
        setupRequired: false,
      });
    },
  });
}

export function useLogoutMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (csrfToken: string) => logout(csrfToken),
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: authKeys.session });
    },
  });
}
