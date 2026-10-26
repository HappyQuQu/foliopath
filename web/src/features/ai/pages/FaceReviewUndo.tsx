import { useRef, useState, type RefObject } from "react";
import { Button, Dialog, DialogCloseButton, InlineStatus } from "../../../components/ui";
import type { FaceReviewResult } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { useUndoFaceReviewMutation } from "../queries";
import { isFaceUndoConflict } from "../reviewErrors";
import styles from "./FaceReviewUndo.module.css";

export type FaceReviewReceipt = { result: FaceReviewResult; subject: string; source?: string | undefined; target?: string | undefined };
export type FaceReviewCallbacks = {
  onReviewStart?: () => void;
  onReviewCommitted?: (receipt: FaceReviewReceipt) => void;
};

// Exactly one confirmed receipt per mounted workspace, never an inferred history.
export function FaceReviewUndo({ receipt: initialReceipt, session, onDismiss, fallbackFocusRef, onStateResolved }: {
  receipt: FaceReviewReceipt; session: AuthenticatedSession; onDismiss: () => void;
  fallbackFocusRef: RefObject<HTMLElement | null>; onStateResolved?: () => void;
}) {
  const { t } = useLocale();
  const [receipt] = useState(() => ({ ...initialReceipt, result: { ...initialReceipt.result } }));
  const mutation = useUndoFaceReviewMutation();
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState<"ready" | "uncertain" | "conflict" | "done">("ready");
  const [saving, setSaving] = useState(false);
  const busy = useRef(false);
  const [key] = useState(createRequestKey);
  const terminal = status === "conflict" || status === "done";
  const { result } = receipt;
  async function submit() {
    if (busy.current || terminal) return;
    busy.current = true; setSaving(true);
    try {
      await mutation.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: key,
        reviewId: result.reviewId, revision: result.revision });
      setStatus("done"); setOpen(false); onStateResolved?.();
    } catch (error) {
      if (isFaceUndoConflict(error)) { setStatus("conflict"); setOpen(false); onStateResolved?.(); }
      else setStatus("uncertain");
    } finally { busy.current = false; setSaving(false); }
  }
  return <section className={styles.notice} aria-label={t("faceUndo.title")}>
    <InlineStatus tone={status === "done" ? "success" : status === "ready" ? "info" : "warning"}>
      {t(`faceUndo.${status}`)} {t(`faceUndo.action.${result.action}`)} · {receipt.subject}
    </InlineStatus>
    {!terminal && <Button variant="secondary" onClick={() => setOpen(true)}>{t("faceUndo.open")}</Button>}
    <Button variant="quiet" disabled={saving} onClick={onDismiss}>{t("faceUndo.dismiss")}</Button>
    <Dialog open={open} fallbackFocusRef={fallbackFocusRef} onOpenChange={value => { if (!busy.current) setOpen(value); }}
      title={t("faceUndo.confirm")} description={t("faceUndo.safety")}
      actions={<><DialogCloseButton onClick={() => { if (!busy.current) setOpen(false); }} /><Button disabled={saving} onClick={() => void submit()}>{t("faceUndo.submit")}</Button></>}>
      <div className={styles.summary}><strong>{t(`faceUndo.action.${result.action}`)}</strong><p>{receipt.subject}</p>
        {receipt.source && <p>{t("photoFaces.source")} {receipt.source}</p>}
        {receipt.target && <p>{t("photoFaces.destination")} {receipt.target}</p>}
        {status === "uncertain" && <InlineStatus tone="warning">{t("faceUndo.uncertainDetail")}</InlineStatus>}
      </div>
    </Dialog>
  </section>;
}
