import { apiClient } from "./client";
import { createApiError } from "./errors";

export interface AuthenticationStatus {
  setupRequired: boolean;
}

export interface Administrator {
  displayName: string;
  id: string;
  username: string;
}

export interface AuthenticatedSession {
  administrator: Administrator;
  csrfToken: string;
  expiresAt: string;
}

export interface LoginInput {
  password: string;
  username: string;
}

export interface SetupAdministratorInput extends LoginInput {
  displayName: string;
}

function currentOrigin(): string {
  return window.location.origin;
}

export async function getAuthenticationStatus(): Promise<AuthenticationStatus> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/auth/status");
    if (data) return { setupRequired: data.setupRequired };
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function getSession(): Promise<AuthenticatedSession> {
  try {
    const { data, error, response } = await apiClient.GET("/api/v1/auth/session");
    if (data) return mapSession(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function login(input: LoginInput): Promise<AuthenticatedSession> {
  try {
    const { data, error, response } = await apiClient.POST("/api/v1/auth/login", {
      body: input,
      params: { header: { Origin: currentOrigin() } },
    });
    if (data) return mapSession(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function setupAdministrator(
  input: SetupAdministratorInput,
): Promise<AuthenticatedSession> {
  try {
    const { data, error, response } = await apiClient.POST("/api/v1/auth/setup", {
      body: input,
      params: { header: { Origin: currentOrigin() } },
    });
    if (data) return mapSession(data);
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

export async function logout(csrfToken: string): Promise<void> {
  try {
    const { error, response } = await apiClient.POST("/api/v1/auth/logout", {
      headers: { "X-CSRF-Token": csrfToken },
    });
    if (response.ok) return;
    throw createApiError(error, response);
  } catch (error) {
    throw createApiError(error);
  }
}

function mapSession(session: {
  administrator: Administrator;
  csrfToken: string;
  expiresAt: string;
}): AuthenticatedSession {
  return {
    administrator: { ...session.administrator },
    csrfToken: session.csrfToken,
    expiresAt: session.expiresAt,
  };
}
