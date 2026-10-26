import { ApiError } from "../../lib/api/errors";

export function isFaceReviewConflict(error: unknown): boolean {
  return error instanceof ApiError && ["precondition_failed", "face_constraint_conflict", "face_cursor_stale", "idempotency_conflict", "person_not_found", "face_not_found"].includes(error.code);
}

export function isFaceUndoConflict(error: unknown): boolean {
  return isFaceReviewConflict(error) || error instanceof ApiError &&
    ["face_review_not_undoable", "face_review_not_found"].includes(error.code);
}
