import { ArrowRight, Tag } from "@phosphor-icons/react";
import { useEffect, useId, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Button, Dialog, ErrorState, InlineStatus, LoadingState, Switch } from "../../../components/ui";
import type { AIModelOperation } from "../../../lib/api/ai";
import { ApiError } from "../../../lib/api/errors";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useAutomaticTagSettings, useRequestAutomaticTags, useUpdateAutomaticTags } from "../automaticTagQueries";
import styles from "./AISettingsOverview.module.css";
import { AIProcessingRow } from "./AIProcessingRow";
import { TagLanguageControl } from "../../curation";

type Props = { libraryId: string; csrfToken: string; onOperation: (operation: Pick<AIModelOperation, "id">) => void; mode?: "settings" | "tasks"; onSettings?: () => void };
export function AutomaticTagSettings(props: Props) {
  return <LibraryAutomaticTagSettings key={`${props.libraryId}-${props.mode ?? "settings"}`} {...props} />;
}

function LibraryAutomaticTagSettings({ libraryId, csrfToken, onOperation, mode = "settings", onSettings }: Props) {
  const { locale, t } = useLocale();
  const heading = useId();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const settings = useAutomaticTagSettings(libraryId);
  const update = useUpdateAutomaticTags();
  const job = useRequestAutomaticTags();
  const [confirm, setConfirm] = useState(false);
  const [failure, setFailure] = useState<MessageKey | null>(null);
  const value = settings.data;
  const remembered = useRef<string | null>(null);
  useEffect(() => {
    const id = value?.activeOperationId;
    if (id && remembered.current !== id) { remembered.current = id; onOperation({ id }); }
  }, [onOperation, value?.activeOperationId]);
  const pending = update.isPending || job.isPending;
  const unavailable = !value || settings.isError || settings.isPending;
  const blocked = unavailable || pending || !value?.enabled || value.offline || !value.model || value.state === "awaiting_model" || Boolean(value.activeOperationId);
  // Degraded also represents the backend's durable failure/cancellation fence.
  // Its public contract intentionally does not expose a separate pause setting;
  // explicit all is the safe recovery action for every degraded reason.
  const missingBlocked = blocked || value?.state === "degraded";
  const number = new Intl.NumberFormat(locale);
  async function toggle(enabled: boolean) {
    if (!value || unavailable || pending) return;
    setFailure(null);
    try { await update.mutateAsync({ libraryId, csrfToken, enabled, etag: value.etag }); }
    catch (error) { setFailure(error instanceof ApiError && error.code === "precondition_failed" ? "autoTags.conflict" : "autoTags.saveFailed"); }
  }
  async function run(mode: "missing" | "all") {
    if (blocked || mode === "missing" && missingBlocked) return;
    setFailure(null);
    try { onOperation(await job.mutateAsync({ libraryId, csrfToken, mode, idempotencyKey: createRequestKey() })); setConfirm(false); }
    catch { setFailure("autoTags.jobFailed"); setConfirm(false); }
  }
  const confirmation = <Dialog fallbackFocusRef={headingRef} open={confirm} onOpenChange={setConfirm} title={t("autoTags.confirmTitle")} actions={<>
    <Button variant="quiet" disabled={pending} onClick={() => setConfirm(false)}>{t("common.cancel")}</Button>
    <Button variant="primary" disabled={blocked} loading={job.isPending} onClick={() => void run("all")}>{t("autoTags.rerun")}</Button>
  </>}><p>{t("autoTags.confirmDescription")}</p></Dialog>;
  if (mode === "tasks") return <>
    <AIProcessingRow headingRef={headingRef} title={t("autoTags.title")} description={t("ai.tasks.tagsDescription")} icon={<Tag aria-hidden="true" size={26} />}
      action={!unavailable && value && !value.enabled ? <Button onClick={onSettings}>{t("ai.tasks.enable")}</Button> : <Button disabled={missingBlocked} loading={job.isPending} onClick={() => void run("missing")}>{t("autoTags.fill")}</Button>}
      maintenance={!unavailable && value ? <>
        <p>{t("curation.automaticProtection")}</p>
        <div className={styles.actions}><Button disabled={blocked} onClick={() => setConfirm(true)}>{t("autoTags.rerun")}</Button>
          <Link className={styles.textLink} to={`${paths.tags}?${new URLSearchParams({ libraryId })}`}>{t("autoTags.view")}<ArrowRight size={16} aria-hidden="true" /></Link></div>
      </> : undefined}>
      {settings.isPending ? <LoadingState density="compact" label={t("ai.tasks.loadingStatus")} /> : settings.isError ? <ErrorState density="compact" message={t("ai.tasks.statusFailed")} onRetry={() => void settings.refetch()} /> : value && <>
        <strong>{t(`autoTags.state.${value.state}`)}</strong>
        <p>{number.format(value.coverage.completed)} / {number.format(value.coverage.eligible)} · {number.format(value.coverage.failed)} {t("ai.failed")} · {number.format(value.coverage.stale)} {t("autoTags.stale")}</p>
        {value.enabled && <progress aria-label={t("autoTags.progress")} max={Math.max(1, value.coverage.eligible)} value={value.coverage.completed} />}
        {value.offline && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
        {!value.model && <p>{t("autoTags.noModel")}</p>}
      </>}
      {failure && <InlineStatus tone="danger">{t(failure)}</InlineStatus>}
    </AIProcessingRow>
    {confirmation}
  </>;
  return <section className={styles.feature} aria-labelledby={heading}>
    <header className={styles.featureHeader}>
      <Tag className={styles.featureIcon} size={24} aria-hidden="true" />
      <div><h2 id={heading} ref={headingRef} tabIndex={-1}>{t("autoTags.title")}</h2><p>{t("autoTags.description")}</p></div>
      {value && <Switch aria-label={t("autoTags.enable")} checked={value.enabled} disabled={unavailable || pending} onChange={event => void toggle(event.currentTarget.checked)} />}
    </header>
    {settings.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : settings.isError ? <ErrorState message={t("ai.statusFailed")} onRetry={() => void settings.refetch()} /> : value && <>
      <InlineStatus tone={value.state === "ready" ? "success" : value.state === "degraded" || value.state === "awaiting_model" ? "warning" : "info"}>{t(`autoTags.state.${value.state}`)}</InlineStatus>
      {value.offline && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
      <div className={styles.coverage}>
        <div><span>{t("autoTags.progress")}</span><strong>{number.format(value.coverage.completed)} / {number.format(value.coverage.eligible)}</strong></div>
        <progress aria-label={t("autoTags.progress")} max={Math.max(1, value.coverage.eligible)} value={value.coverage.completed} />
        <span>{number.format(value.coverage.failed)} {t("ai.failed")} · {number.format(value.coverage.stale)} {t("autoTags.stale")}</span>
      </div>
      <p className={styles.hint}>{t("curation.automaticProtection")}</p>
      <TagLanguageControl />
      <p className={styles.hint}>{t("tagLanguage.hint")}</p>
      {!value.model && <p className={styles.hint}>{t("autoTags.noModel")}</p>}
      <div className={styles.featureActions}>
        <Link className={styles.textLink} to={`${paths.tags}?${new URLSearchParams({ libraryId })}`}>{t("autoTags.view")}<ArrowRight size={16} aria-hidden="true" /></Link>
        <Button variant="quiet" disabled={missingBlocked} loading={job.isPending} onClick={() => void run("missing")}>{t("autoTags.fill")}</Button>
        <Button variant="quiet" disabled={blocked} onClick={() => setConfirm(true)}>{t("autoTags.rerun")}</Button>
      </div>
    </>}
    {failure && <InlineStatus tone="danger">{t(failure)}</InlineStatus>}
    {confirmation}
  </section>;
}
