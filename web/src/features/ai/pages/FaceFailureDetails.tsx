import { Link } from "react-router-dom";
import { Button, InlineStatus } from "../../../components/ui";
import type { FaceSettingsSnapshot } from "../../../lib/api/ai";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import styles from "./FaceFailureDetails.module.css";

const taskReasons: Record<string, MessageKey> = {
  library_not_ready: "ai.faceDiagnostics.library_not_ready",
  library_scanning: "ai.faceDiagnostics.library_scanning",
  library_offline: "ai.faceDiagnostics.library_offline",
  model_unavailable: "ai.faceDiagnostics.model_unavailable",
  face_disabled: "ai.faceDiagnostics.face_disabled",
  operation_interrupted: "ai.faceDiagnostics.operation_interrupted",
  cancelled: "ai.faceDiagnostics.cancelledReason",
};
export function FaceAnalysisStatus({ settings, library, plain = false }: { settings: FaceSettingsSnapshot; library: Pick<LibrarySummary, "id" | "status">; plain?: boolean }) {
  const { t } = useLocale();
  const task = settings.diagnostics?.latestTask;
  let message: string = t(`ai.state.${settings.state}`);
  if (settings.enabled && settings.state !== "clearing" && settings.generationCurrent === false) message = t("ai.faceDiagnostics.historical");
  else if (settings.state === "degraded") {
    if (task?.state === "failed") message = t(settings.coverage.failed > 0 ? "ai.faceDiagnostics.partialInterrupted" : "ai.faceDiagnostics.interrupted");
    else if (task?.state === "cancelled") message = t("ai.faceDiagnostics.cancelled");
    else if (settings.coverage.failed === 0) message = t("ai.faceDiagnostics.incomplete");
  } else if (settings.state === "building" && library.status === "scanning" && (task?.state === "running" || task?.state === "queued")) message = t("ai.faceDiagnostics.waitingScan");
  return <InlineStatus appearance={plain ? "plain" : "surface"} tone={settings.state === "degraded" || settings.generationCurrent === false ? "warning" : settings.state === "ready" ? "success" : "info"}>{message}</InlineStatus>;
}

export function FaceGenerationHint({ settings }: { settings: FaceSettingsSnapshot }) {
  const { t } = useLocale();
  return settings.generationCurrent === false ? <p>{t("ai.faceDiagnostics.historicalHint")}</p> : null;
}

export function FaceFailureDetails({ settings, library, disabled, onRetry }: { settings: FaceSettingsSnapshot; library: Pick<LibrarySummary, "id" | "status">; disabled: boolean; onRetry: () => void }) {
  const { locale, t } = useLocale();
  const diagnostics = settings.diagnostics;
  const task = diagnostics?.latestTask;
  const number = new Intl.NumberFormat(locale);
  if (settings.state !== "degraded" && settings.coverage.failed === 0 && !(task?.state === "failed" || task?.state === "cancelled")) return null;
  return <details className={styles.details}>
    <summary>{t("ai.faceDiagnostics.details")}</summary>
    {diagnostics ? <>
      {task && <div className={styles.task}>
        <strong>{t("ai.faceDiagnostics.latest")} · {t(`ai.operationState.${task.state}`)}</strong>
        <time dateTime={task.updatedAt}>{new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(new Date(task.updatedAt))}</time>
        {(task.state === "failed" || task.state === "cancelled") && <p>{t(taskReasons[task.errorCode ?? ""] ?? "ai.faceDiagnostics.unknownTaskReason")}</p>}
        {task.errorCode && taskReasons[task.errorCode] && <code>{task.errorCode}</code>}
      </div>}
      <p>{t("ai.faceDiagnostics.recorded").replace("{count}", number.format(diagnostics.recordedFailures))}</p>
      {diagnostics.unrecordedFailures > 0 && <InlineStatus tone="warning">{t("ai.faceDiagnostics.legacy").replace("{count}", number.format(diagnostics.unrecordedFailures))}</InlineStatus>}
      {diagnostics.reasons.length > 0 && <ul>{diagnostics.reasons.map(reason => <li key={reason.code}>{t(`ai.faceDiagnostics.${reason.code}`)} · {number.format(reason.count)}</li>)}</ul>}
      {diagnostics.samples.length > 0 && <>
        <p>{t("ai.faceDiagnostics.sample")}</p>
        <ul className={styles.samples}>{diagnostics.samples.map(sample => <li key={sample.assetId}>
          <Link to={paths.media(library.id, sample.assetId, `${paths.aiSettings}?${new URLSearchParams({ library: library.id, view: "tasks" })}`)}>{sample.name}</Link>
          <span>{t(sample.code === "invalid_media" && sample.sizeBytes === 0 ? "ai.faceDiagnostics.emptySource" : `ai.faceDiagnostics.${sample.code}`)}</span>
          <time dateTime={sample.updatedAt}>{new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(new Date(sample.updatedAt))}</time>
        </li>)}</ul>
      </>}
      {diagnostics.reasons.some(reason => reason.code === "invalid_media") && <p>{t("ai.faceDiagnostics.sourceRepairHint")}</p>}
    </> : <p>{t("ai.faceDiagnostics.notRecorded")}</p>}
    <p>{t("ai.faceDiagnostics.retryHint")}</p>
    <Button size="small" disabled={disabled} onClick={onRetry}>{t("ai.faceDiagnostics.retry")}</Button>
  </details>;
}
