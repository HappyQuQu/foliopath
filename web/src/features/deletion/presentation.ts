import { ApiError } from "../../lib/api/errors";
import type { DeletionOperation, DeletionSettings } from "../../lib/api/deletion";
import type { MessageKey } from "../../lib/i18n/LocaleProvider";

export function deletionOperationIsActive(state: DeletionOperation["state"]): boolean {
  return state === "queued" || state === "running" || state === "reconciling";
}

export function deletionErrorMessage(error: unknown, unknownMessage: MessageKey = "deletion.unknown"): MessageKey {
  if (!(error instanceof ApiError)) return unknownMessage;
  switch (error.code) {
    case "deletion_disabled": return "deletion.disabled";
    case "deletion_unavailable": case "deletion_permission_denied": return "deletion.unavailable";
    case "deletion_busy": case "rate_limited": return "deletion.busy";
    case "deletion_confirmation_expired": case "deletion_source_changed": case "precondition_failed": case "deletion_conflict": case "deletion_not_found": return "deletion.changed";
    default: return error.status === 0 || error.status >= 500 ? unknownMessage : "deletion.failed";
  }
}
export function deletionUnavailableMessage(settings: DeletionSettings): MessageKey {
  switch (settings.unavailableReason) {
    case "not_configured": return "deletion.notConfigured";
    case "read_only": return "deletion.readOnly";
    case "unsupported": return "deletion.unsupported";
    default: return "deletion.unavailable";
  }
}
export function deletionItemErrorMessage(code: string): MessageKey {
  switch (code) {
    case "source_changed": case "source_missing": case "permission_denied": case "write_unavailable": case "interrupted": return `deletion.error.${code}`;
    case "destination_exists": return "deletion.error.conflict";
    default: return "deletion.error.io_error";
  }
}
export function deletionSubmissionUncertain(error: unknown) {
  return !(error instanceof ApiError) || error.status === 0 || error.status >= 500;
}
