import { Button, ErrorState, InlineStatus, LoadingState } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useVideoSemanticStatusQuery } from "../queries";
import styles from "./AISettingsOverview.module.css";

export function VideoSemanticProgress({ library, disabled, onJob }: { library: LibrarySummary; disabled: boolean; onJob: (mode: "missing" | "all") => void }) {
  const { locale, t } = useLocale();
  const query = useVideoSemanticStatusQuery(library.id);
  const status = query.data;
  const coverage = status?.coverage;
  const number = new Intl.NumberFormat(locale);
  const pending = coverage ? Math.max(0, coverage.eligible - coverage.completed - coverage.degraded - coverage.failed - coverage.stale) : 0;
  return <section className={styles.section} aria-labelledby="ai-video-progress-heading">
    <div className={styles.sectionHeading}><div>
      <h2 id="ai-video-progress-heading">{t("ai.video.progress")}</h2>
      <p>{library.name} · {t("ai.video.description")}</p>
    </div></div>
    {query.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : query.isError || !status ?
      <ErrorState message={t("ai.statusFailed")} onRetry={() => void query.refetch()} /> : <>
        {status.offline && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
        {!status.enabled && <InlineStatus>{t("ai.disabled")}</InlineStatus>}
        {!coverage ? <InlineStatus>{t("ai.video.unrecorded")}</InlineStatus> : <div className={styles.coverage}>
          <progress aria-label={`${library.name} ${t("ai.video.progress")}`} max={Math.max(1, coverage.eligible)} value={coverage.completed} />
          <span>{t("ai.video.completed")} {number.format(coverage.completed)} / {number.format(coverage.eligible)}</span>
          <span>{t("ai.video.pending")} {number.format(pending)} · {t("ai.video.degraded")} {number.format(coverage.degraded)} · {t("ai.failed")} {number.format(coverage.failed)} · {t("ai.video.stale")} {number.format(coverage.stale)}</span>
        </div>}
        <div className={styles.actions}>
          <Button disabled={disabled || !status.enabled || status.offline || library.status === "offline"} onClick={() => onJob("missing")}>{t("ai.video.fillMissing")}</Button>
          <Button variant="quiet" disabled={disabled || !status.enabled || status.offline || library.status === "offline"} onClick={() => onJob("all")}>{t("ai.video.rebuild")}</Button>
        </div>
      </>}
  </section>;
}
