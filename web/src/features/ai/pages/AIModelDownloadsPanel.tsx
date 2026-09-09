import { CheckCircle, DownloadSimple, ArrowClockwise, Cpu, CloudArrowDown } from "@phosphor-icons/react";
import type { AIModelDownloadPage, AIOperationSnapshot } from "../../../lib/api/ai";
import { Button, ErrorState, InlineStatus, LoadingState } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { operationIsActive } from "../queries";
import styles from "./AISettingsPage.module.css";

export function AIModelDownloadsPanel({ query, pending, onDownload, onCancel }: {
  query: { data: AIModelDownloadPage | undefined; isPending: boolean; isError: boolean; refetch: () => unknown };
  pending: boolean;
  onDownload: (id: string) => void;
  onCancel: (operation: AIOperationSnapshot) => void;
}) {
  const { locale, t } = useLocale();
  const bytes = (value: number) => new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(value / 1024 ** 2) + " MiB";
  return <section className={styles.card} aria-labelledby="model-download-title">
    <header className={styles.cardHeader}>
      <div><h2 id="model-download-title">{t("ai.download.title")}</h2><p>{t("ai.download.description")}</p></div>
      <CloudArrowDown aria-hidden="true" size={26} />
    </header>
    {query.isPending ? <LoadingState label={t("ai.loadingModels")} /> : query.isError ?
      <ErrorState message={t("ai.download.loadFailed")} onRetry={() => void query.refetch()} /> : <>
      {!query.data?.runtimeAvailable && <InlineStatus tone="warning"><Cpu aria-hidden="true" size={18} /> {t("ai.download.runtimeMissing")}</InlineStatus>}
      {query.data?.items.length === 0 && <div className={styles.setupEmpty}>
        <h3>{t("ai.download.noSource")}</h3><p>{t("ai.download.noSourceDescription")}</p>
      </div>}
      <div className={styles.rows}>{query.data?.items.map((item) => {
        const operation = item.operation;
        const active = operation ? operationIsActive(operation) : false;
        const failed = operation?.state === "failed" || operation?.state === "cancelled";
        const total = operation?.totalItems ?? item.packageSizeBytes;
        const completed = operation?.completedItems ?? 0;
        const installed = Boolean(item.installedModelId);
        return <article key={item.id} className={styles.downloadRow}>
          <div className={styles.modelRow}>
            <div className={styles.modelIcon}>{installed ? <CheckCircle aria-hidden="true" size={23} /> : <DownloadSimple aria-hidden="true" size={23} />}</div>
            <div className={styles.modelInfo}><strong>{item.name}</strong><span>{t(`ai.purpose.${item.purpose}`)}</span><span>{item.sourceLabel} · {item.version} · {bytes(item.packageSizeBytes)} · {item.licenseId}</span></div>
            {active && operation ? <Button disabled={pending || operation.state === "cancelling"} size="small" onClick={() => onCancel(operation)}>{t(operation.state === "cancelling" ? "ai.operationState.cancelling" : "ai.cancelTask")}</Button> : installed ?
              <InlineStatus tone="success">{t("ai.download.installed")}</InlineStatus> :
              <Button disabled={pending || item.compatibility !== "compatible"} variant="primary" onClick={() => onDownload(item.id)}>
                {failed ? <ArrowClockwise aria-hidden="true" size={17} /> : <DownloadSimple aria-hidden="true" size={17} />}{t(failed ? "ai.download.retry" : "ai.download.action")}
              </Button>}
          </div>
          {item.compatibility !== "compatible" && <InlineStatus tone="warning">{t("ai.compatibility.wrong_architecture")}</InlineStatus>}
          {operation && active && <div className={styles.downloadProgress}>
            <div className={styles.progressLabels}><span>{operation.state === "queued" ? t("ai.operationState.queued") : completed >= total ? t("ai.phase.verifying") : t("ai.download.receiving")}</span><span>{bytes(completed)} / {bytes(total)}</span></div>
            <progress aria-label={`${item.name} ${t("ai.download.progress")}`} max={total} value={completed} />
            <p>{t("ai.download.background")}</p>
          </div>}
          {failed && !installed && <InlineStatus tone="danger">{operation?.errorCode ? t(`ai.error.${operation.errorCode}`) : t("ai.download.failed")}</InlineStatus>}
        </article>;
      })}</div>
      {Boolean(query.data?.presets?.length) && <section aria-labelledby="model-presets-title">
        <header className={styles.cardHeader}><div><h3 id="model-presets-title">{t("ai.download.presetsTitle")}</h3><p>{t("ai.download.presetsDescription")}</p></div></header>
        <div className={styles.rows}>{query.data?.presets?.map((preset) => <article className={styles.downloadRow} key={preset.id}>
          <div className={styles.modelRow}>
            <div className={styles.modelIcon}><Cpu aria-hidden="true" size={23} /></div>
            <div className={styles.modelInfo}><strong>{preset.name}</strong><span>{t(`ai.purpose.${preset.purpose}`)}</span><span>{preset.sourceLabel}</span></div>
            <InlineStatus tone="warning">{t(`ai.download.${preset.readiness}`)}</InlineStatus>
          </div>
          <p>{t(`ai.download.${preset.readiness}Description`)}</p>
        </article>)}</div>
      </section>}
      <details className={styles.details}><summary>{t("ai.download.setupGuide")}</summary>
        <p>{t("ai.download.setupInstructions")}</p>
      </details>
    </>}
  </section>;
}
