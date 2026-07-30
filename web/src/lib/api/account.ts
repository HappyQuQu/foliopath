import type { components } from "./generated/schema";
import { apiClient } from "./client";
import { createApiError } from "./errors";

export type Account = components["schemas"]["Account"] & {
  etag: string;
};

export async function getAccount(): Promise<Account> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/account");
    if (data) return { ...data, etag: requireEtag(response) };
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function updateAccount(input: {
  csrfToken: string;
  displayName: string;
  etag: string;
}): Promise<Account> {
  try {
    const { data, error, response } = await apiClient.PATCH("/api/v1/account", {
      body: { displayName: input.displayName },
      headers: { "X-CSRF-Token": input.csrfToken },
      params: { header: { "If-Match": input.etag } },
    });
    if (data) return { ...data, etag: requireEtag(response) };
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function changeAccountPassword(input: {
  csrfToken: string;
  currentPassword: string;
  etag: string;
  newPassword: string;
}): Promise<string> {
  try {
    const { error, response } = await apiClient.POST(
      "/api/v1/account/password",
      {
        body: {
          currentPassword: input.currentPassword,
          newPassword: input.newPassword,
        },
        headers: { "X-CSRF-Token": input.csrfToken },
        params: { header: { "If-Match": input.etag } },
      },
    );
    if (response.status === 204) return requireEtag(response);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function requireEtag(response: Response): string {
  const etag = response.headers.get("ETag");
  if (!etag) throw new Error("Required representation validator was not returned.");
  return etag;
}
