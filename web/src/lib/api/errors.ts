import type { components } from "./generated/schema";

type WireErrorResponse = components["schemas"]["ErrorResponse"];

export type ApiErrorCode = components["schemas"]["ErrorCode"] | "network_error";

export class ApiError extends Error {
  readonly code: ApiErrorCode;
  readonly requestId: string | undefined;
  readonly status: number;

  constructor({
    code,
    message,
    requestId,
    status,
  }: {
    code: ApiErrorCode;
    message: string;
    requestId: string | undefined;
    status: number;
  }) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.requestId = requestId;
    this.status = status;
  }
}

export function createApiError(error: unknown, response?: Response): ApiError {
  if (error instanceof ApiError) return error;

  const wireError = isWireError(error) ? error.error : undefined;

  return new ApiError({
    code: wireError?.code ?? "network_error",
    message: wireError?.message ?? "暂时无法连接 FolioPath，请稍后重试。",
    requestId: wireError?.requestId,
    status: response?.status ?? 0,
  });
}

function isWireError(value: unknown): value is WireErrorResponse {
  if (typeof value !== "object" || value === null || !("error" in value)) return false;
  const error = value.error;
  return (
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    typeof error.code === "string" &&
    "message" in error &&
    typeof error.message === "string"
  );
}

export function isAuthenticationError(error: unknown): error is ApiError {
  return (
    error instanceof ApiError &&
    (error.code === "authentication_required" || error.code === "session_expired")
  );
}
