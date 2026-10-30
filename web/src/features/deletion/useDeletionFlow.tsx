import { ListChecks } from "@phosphor-icons/react";
import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Button, Dialog, InlineStatus, LoadingState, useToast } from "../../components/ui";
import type { AuthenticatedSession } from "../../lib/api/auth";
import { preflightDeletion, submitDeletion, type DeletionPreflight, type DeletionTarget } from "../../lib/api/deletion";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../lib/requestKey";
import { paths } from "../../routes/paths";
import { deletionErrorMessage, deletionSubmissionUncertain } from "./presentation";
import { refreshDeletionState, useDeletionOperationQuery, useDeletionSettingsQuery } from "./queries";
import styles from "./Deletion.module.css";

export function useDeletionFlow(session: AuthenticatedSession | undefined) {
  const { t } = useLocale();
  const client = useQueryClient();
  const navigate = useNavigate();
  const toast = useToast();
  const settings = useDeletionSettingsQuery(Boolean(session));
  const request = useRef<{ targets: DeletionTarget[]; key: string; onAccepted?: () => void } | null>(null);
  const busy = useRef(false);
  const fallbackFocusRef = useRef<HTMLElement | null>(null);
  const [open, setOpen] = useState(false);
  const [preview, setPreview] = useState<DeletionPreflight>();
  const [pending, setPending] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<unknown>();
  const [uncertain, setUncertain] = useState(false);
  const [accepted, setAccepted] = useState(false);
  const [submittedId, setSubmittedId] = useState("");
  const operation = useDeletionOperationQuery(submittedId, "", true);
  const phase = operation.data?.state;
  useEffect(() => {
    if (phase) void refreshDeletionState(client);
  }, [client, submittedId, phase]);

  async function check() {
    if (!session || !request.current || busy.current) return;
    busy.current = true; setChecking(true); setError(undefined); setPreview(undefined);
    try { setPreview(await preflightDeletion(request.current.targets, session.csrfToken)); }
    catch (error) { setError(error); }
    finally { busy.current = false; setChecking(false); }
  }
  function start(targets: DeletionTarget[], onAccepted?: () => void) {
    if (!session || busy.current || uncertain || !targets.length) return;
    fallbackFocusRef.current = document.getElementById("main");
    request.current = { targets, key: createRequestKey(), ...(onAccepted ? { onAccepted } : {}) };
    setOpen(true); setAccepted(false); setUncertain(false); void check();
  }
  async function submit() {
    if (!session || !request.current || !preview?.canSubmit || busy.current) return;
    busy.current = true; setPending(true); setError(undefined);
    try {
      const submitted = await submitDeletion(preview, session.csrfToken, request.current.key);
      setSubmittedId(submitted.id);
      setAccepted(true); setUncertain(false);
      toast.show({ message: t("deletion.queued"), tone: "success" });
      request.current.onAccepted?.();
      void refreshDeletionState(client);
    } catch (error) { setError(error); setUncertain(deletionSubmissionUncertain(error)); }
    finally { busy.current = false; setPending(false); }
  }
  const dialog = <Dialog fallbackFocusRef={fallbackFocusRef} open={open} onOpenChange={value => { if (!busy.current && !uncertain) setOpen(value); }}
    title={t(accepted ? "deletion.accepted" : !preview ? "deletion.delete" : "deletion.confirmPermanent")}
    actions={accepted ? <><Button variant="primary" onClick={() => { setOpen(false); navigate(paths.fileTasks); }}><ListChecks size={18} aria-hidden="true" />{t("deletion.openTasks")}</Button><Button onClick={() => setOpen(false)}>{t("deletion.close")}</Button></> : <>
      <Button disabled={pending || uncertain} onClick={() => setOpen(false)}>{t("deletion.cancel")}</Button>
      {(error || (preview && !preview.canSubmit)) && !uncertain && <Button disabled={pending || checking} onClick={() => { request.current!.key = createRequestKey(); void check(); }}>{t("deletion.refresh")}</Button>}
      <Button variant="danger" loading={pending} disabled={checking || !preview?.canSubmit} onClick={() => void submit()}>{t(uncertain ? "deletion.retry" : "deletion.permanent")}</Button>
    </>}>
    <div className={styles.stack}>
      {accepted ? <p>{t("deletion.queued")}</p> : <>
        {checking && <LoadingState label={t("deletion.checking")} />}
        {preview && <>
          <p>{t("deletion.scope")}</p>
          <InlineStatus tone="warning">{t("deletion.irreversible")}</InlineStatus>
          <ul className={styles.previewList}>{preview.items.map(item => <li key={`${item.target.type}:${item.target.id}`}>
            <strong>{item.name ?? t("deletion.block.missing")}</strong><div className={styles.muted}>{item.libraryName} · {item.relativePath}</div>
            {item.blockedReason && <InlineStatus tone="warning">{t(`deletion.block.${item.blockedReason}`)}</InlineStatus>}
          </li>)}</ul>
        </>}
        {error !== undefined && <InlineStatus tone="danger">{t(deletionErrorMessage(error, preview ? "deletion.unknown" : "deletion.loadFailed"))}</InlineStatus>}
        {uncertain && <Button variant="primary" onClick={() => { setOpen(false); navigate(paths.fileTasks); }}><ListChecks size={18} aria-hidden="true" />{t("deletion.openTasks")}</Button>}
      </>}
    </div>
  </Dialog>;
  return { start, dialog, settings, canDelete: Boolean(settings.data?.available && settings.data.enabled) };
}
