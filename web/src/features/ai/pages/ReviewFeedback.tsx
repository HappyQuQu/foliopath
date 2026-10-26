import { useSearchParams } from "react-router-dom";
import { Button, ErrorState } from "../../../components/ui";
import { ApiError } from "../../../lib/api/errors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import styles from "./AIReviewPage.module.css";

export function ReviewPagination({ nextCursor, cursorParam = "cursor" }: { nextCursor: string | null | undefined; cursorParam?: "cursor" | "peopleCursor" | "groupsCursor" }) {
 const { t } = useLocale(); const [params, setParams] = useSearchParams();
 if (!nextCursor && !params.has(cursorParam)) return null;
 function go(cursor?: string) { const next = new URLSearchParams(params); if (cursor) next.set(cursorParam, cursor); else next.delete(cursorParam); setParams(next); }
 return <div className={styles.actions}>
  <Button variant="quiet" disabled={!params.has(cursorParam)} onClick={() => go()}>{t("intelligence.firstPage")}</Button>
  <Button disabled={!nextCursor} onClick={() => go(nextCursor!)}>{t("intelligence.nextPage")}</Button>
 </div>;
}

export function ReviewErrorState({ error, onRetry, cursorParam = "cursor" }: { error: unknown; onRetry: () => void; cursorParam?: "cursor" | "peopleCursor" | "groupsCursor" }) {
 const { t } = useLocale();
 const [params, setParams] = useSearchParams();
 const code = error instanceof ApiError ? error.code : "";
 const message = code === "face_not_ready" || code === "face_disabled" ? t("intelligence.notReady")
   : code === "library_offline" ? t("intelligence.offline")
   : code === "precondition_failed" || code === "face_constraint_conflict" || code === "invalid_cursor" || code === "face_cursor_stale" || code === "suggestion_cursor_stale" ? t("intelligence.revisionConflict")
   : t("intelligence.loadFailed");
 return <ErrorState message={message} onRetry={() => {
  if ((code === "invalid_cursor" || code === "face_cursor_stale" || code === "suggestion_cursor_stale") && params.has(cursorParam)) { const next = new URLSearchParams(params); next.delete(cursorParam); setParams(next); }
  else onRetry();
 }} />;
}
