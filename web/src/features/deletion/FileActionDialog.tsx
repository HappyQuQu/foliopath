import { useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Button, Dialog, InlineStatus, useToast } from "../../components/ui";
import type { AuthenticatedSession } from "../../lib/api/auth";
import { changeDeletionOperation, type DeletionOperation } from "../../lib/api/deletion";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../lib/requestKey";
import { paths } from "../../routes/paths";
import { deletionErrorMessage, deletionSubmissionUncertain } from "./presentation";
import { refreshDeletionState } from "./queries";
import styles from "./Deletion.module.css";

export type FileActionIntent = { kind: "stop" | "resume"; operation: DeletionOperation };
export function FileActionDialog({ intent, session, onClose, onAccepted }: { intent: FileActionIntent; session: AuthenticatedSession; onClose: () => void; onAccepted: () => void }) {
  const { t } = useLocale();
  const client = useQueryClient();
  const navigate = useNavigate();
  const toast = useToast();
  const [key] = useState(createRequestKey);
  const busy = useRef(false);
  const fallbackFocusRef = useRef<HTMLElement | null>(document.getElementById("main"));
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<unknown>();
  const [uncertain, setUncertain] = useState(false);
  const permanent = intent.kind === "resume" && intent.operation.kind === "permanent";
  const title = intent.kind === "resume" ? "deletion.resumeTitle" : "deletion.pause";
  async function submit() {
    if (busy.current) return;
    busy.current = true; setPending(true); setError(undefined);
    try {
      await changeDeletionOperation(intent.operation, intent.kind, session.csrfToken, key);
      toast.show({ message: t(intent.kind === "stop" ? "deletion.pauseHint" : "deletion.queued"), tone: "success" });
      onAccepted(); void refreshDeletionState(client); onClose();
    } catch (error) { setError(error); setUncertain(deletionSubmissionUncertain(error)); }
    finally { busy.current = false; setPending(false); }
  }
  function close() { if (!busy.current && !uncertain) onClose(); }
  return <Dialog fallbackFocusRef={fallbackFocusRef} open onOpenChange={close} title={t(title)} actions={<><Button disabled={pending || uncertain} onClick={close}>{t("deletion.cancel")}</Button><Button variant={permanent ? "danger" : "primary"} loading={pending} onClick={() => void submit()}>{t(uncertain ? "deletion.retry" : intent.kind === "resume" ? "deletion.resume" : "deletion.pause")}</Button></>}>
    <div className={styles.stack}>
      <p>{t(intent.kind === "resume" ? "deletion.resumeHint" : "deletion.pauseHint")}</p>
      {error !== undefined && <InlineStatus tone="danger">{t(deletionErrorMessage(error))}</InlineStatus>}
      {uncertain && <Button onClick={() => navigate(paths.fileTasks)}>{t("deletion.openTasks")}</Button>}
    </div>
  </Dialog>;
}
