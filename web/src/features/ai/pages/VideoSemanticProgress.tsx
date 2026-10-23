import { Button, ErrorState, InlineStatus, LoadingState } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useVideoSemanticStatusQuery } from "../queries";
import styles from "./AISettingsOverview.module.css";
import { VideoCamera } from "@phosphor-icons/react";
import { AIProcessingRow } from "./AIProcessingRow";

export function VideoSemanticProgress({ library, disabled, onJob }: { library: LibrarySummary; disabled: boolean; onJob: (mode: "missing" | "all") => void }) {
  const { locale, t } = useLocale();
  const query = useVideoSemanticStatusQuery(library.id);
  const status = query.data;
  const coverage = status?.coverage;
  const number = new Intl.NumberFormat(locale);
  const pending = coverage ? Math.max(0, coverage.eligible - coverage.completed - coverage.degraded - coverage.failed - coverage.stale) : 0;
  const blocked = disabled || query.isPending || query.isError || !status || !status.enabled || status.offline || library.status === "offline";
  return <AIProcessingRow title={t("ai.tasks.video")} description={t("ai.tasks.videoDescription")} icon={<VideoCamera aria-hidden="true" size={26} />}
    action={<Button disabled={blocked} onClick={() => onJob("missing")}>{t("ai.fillMissing")}</Button>}
    maintenance={status && !query.isPending && !query.isError ? <><p>{t("ai.tasks.maintenanceHint")}</p><Button disabled={blocked} onClick={() => onJob("all")}>{t("ai.video.rebuild")}</Button></> : undefined}>
    {query.isPending ? <LoadingState density="compact" label={t("ai.tasks.loadingStatus")} /> : query.isError || !status ?
      <ErrorState density="compact" message={t("ai.tasks.statusFailed")} onRetry={() => void query.refetch()} /> : <>
        {status.offline && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
        {!status.enabled && <InlineStatus>{t("ai.disabled")}</InlineStatus>}
        {!coverage ? <InlineStatus appearance="plain">{t("ai.tasks.unknown")}</InlineStatus> : <div className={styles.coverage}>
          <progress aria-label={`${library.name} ${t("ai.video.progress")}`} max={Math.max(1, coverage.eligible)} value={coverage.completed} />
          <span>{t("ai.video.completed")} {number.format(coverage.completed)} / {number.format(coverage.eligible)}</span>
          <span>{t("ai.video.pending")} {number.format(pending)} · {t("ai.video.degraded")} {number.format(coverage.degraded)} · {t("ai.failed")} {number.format(coverage.failed)} · {t("ai.video.stale")} {number.format(coverage.stale)}</span>
        </div>}
      </>}
  </AIProcessingRow>;
}
