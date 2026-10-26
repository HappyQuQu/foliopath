import { ArrowRight, Brain, MagnifyingGlass, Users } from "@phosphor-icons/react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { Button, ErrorState, InlineStatus, LoadingState, Switch } from "../../../components/ui";
import type { AIModel, FaceSettingsSnapshot, SemanticSettingsSnapshot } from "../../../lib/api/ai";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import styles from "./AISettingsOverview.module.css";
import { FaceAnalysisStatus, FaceGenerationHint } from "./FaceFailureDetails";

type SnapshotQuery<T> = { data: T | undefined; isPending: boolean; isError: boolean; refetch: () => unknown };

export function AISettingsOverview({ library, semantic, faces, semanticReady, faceReady, pending, onSemantic, onFace, onTasks, children }: {
  children?: ReactNode;
  library: LibrarySummary;
  semantic: SnapshotQuery<SemanticSettingsSnapshot> | undefined;
  faces: SnapshotQuery<FaceSettingsSnapshot> | undefined;
  semanticReady: boolean;
  faceReady: boolean;
  pending: boolean;
  onSemantic: (library: LibrarySummary, settings: SemanticSettingsSnapshot, enabled: boolean) => void;
  onFace: (library: LibrarySummary, settings: FaceSettingsSnapshot, enabled: boolean) => void;
  onTasks: () => void;
}) {
  const { locale, t } = useLocale();
  const number = new Intl.NumberFormat(locale);
  const semanticSettings = semantic?.data;
  const faceSettings = faces?.data;
  return <div className={styles.overview}>
    {library.status === "offline" && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
    <div className={styles.features}>
      <section className={styles.feature} aria-labelledby="ai-search-title">
        <header className={styles.featureHeader}>
          <MagnifyingGlass className={styles.featureIcon} size={24} aria-hidden="true" />
          <div><h2 id="ai-search-title">{t("ai.settings.semantic")}</h2><p>{t("ai.settings.semanticDescription")}</p></div>
          {semanticSettings && !semantic?.isError && !semantic?.isPending && <Switch aria-label={t("ai.toggleLibrary").replace("{name}", library.name)} checked={semanticSettings.enabled}
            disabled={pending || library.status === "offline" || semanticSettings.state === "clearing" || (!semanticSettings.enabled && !semanticReady)}
            onChange={event => onSemantic(library, semanticSettings, event.currentTarget.checked)} />}
        </header>
        {semantic?.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : !semantic || semantic.isError ? <ErrorState message={t("ai.statusFailed")} onRetry={() => void semantic?.refetch()} /> : semanticSettings && <>
          <CapabilityStatus state={semanticSettings.state} enabled={semanticSettings.enabled} />
          {semanticSettings.coverageRecorded === false ? <p className={styles.hint}>{t("ai.settings.progressUnknown")}</p> : <div className={styles.coverage}>
            <div><span>{t("ai.settings.indexProgress")}</span><strong>{number.format(semanticSettings.coverage.completed)} / {number.format(semanticSettings.coverage.eligible)}</strong></div>
            <progress aria-label={t("ai.settings.indexProgress")} max={Math.max(1, semanticSettings.coverage.eligible)} value={semanticSettings.coverage.completed} />
            <span>{number.format(semanticSettings.coverage.failed)} {t("ai.failed")} · {number.format(semanticSettings.coverage.stale)} {t("ai.video.stale")}</span>
          </div>}
          {!semanticReady && <InlineStatus tone="warning">{t("ai.settings.semanticUnavailable")}</InlineStatus>}
          <p className={styles.hint}>{t("ai.settings.enableSemanticDescription")}</p>
          <div className={styles.featureActions}><Link className={styles.textLink} to={paths.librarySearch(library.id) + "?mode=semantic"}>{t("ai.settings.openSearch")}<ArrowRight size={16} aria-hidden="true" /></Link><Button variant="quiet" onClick={onTasks}>{t("ai.settings.manageJobs")}</Button></div>
        </>}
      </section>
      <section className={styles.feature} aria-labelledby="ai-people-title">
        <header className={styles.featureHeader}>
          <Users className={styles.featureIcon} size={24} aria-hidden="true" />
          <div><h2 id="ai-people-title">{t("ai.settings.people")}</h2><p>{t("ai.settings.enablePeopleDescription")}</p></div>
          {faceSettings && !faces?.isError && !faces?.isPending && <Switch aria-label={t("ai.faceToggle").replace("{name}", library.name)} checked={faceSettings.enabled}
            disabled={pending || library.status === "offline" || faceSettings.state === "clearing" || (!faceSettings.enabled && !faceReady)}
            onChange={event => onFace(library, faceSettings, event.currentTarget.checked)} />}
        </header>
        {faces?.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : !faces || faces.isError ? <ErrorState message={t("ai.statusFailed")} onRetry={() => void faces?.refetch()} /> : faceSettings && <>
          <FaceAnalysisStatus settings={faceSettings} library={library} />
          <div className={styles.coverage}>
            <div><span>{t(faceSettings.generationCurrent === false ? "ai.faceDiagnostics.historicalProgress" : "ai.settings.peopleProgress")}</span><strong>{number.format(faceSettings.coverage.completed)} / {number.format(faceSettings.coverage.eligible)}</strong></div>
            <progress aria-label={t("ai.settings.peopleProgress")} max={Math.max(1, faceSettings.coverage.eligible)} value={faceSettings.coverage.completed} />
            <span>{number.format(faceSettings.coverage.failed)} {t("ai.failed")} · {number.format(faceSettings.coverage.stale)} {t("ai.video.stale")}</span>
          </div>
          <FaceGenerationHint settings={faceSettings} />
          <p className={styles.hint}>{t("ai.settings.peopleWorkflow")}</p>
          {!faceReady && <InlineStatus tone="warning">{t("ai.settings.peopleUnavailable")}</InlineStatus>}
          <div className={styles.featureActions}><Link className={styles.textLink} to={paths.intelligence + "?" + new URLSearchParams({ view: "clusters", libraryId: library.id })}>{t("ai.settings.openPeople")}<ArrowRight size={16} aria-hidden="true" /></Link><Button variant="quiet" onClick={onTasks}>{t("ai.settings.manageJobs")}</Button></div>
        </>}
      </section>
      {children}
    </div>
  </div>;
}

function CapabilityStatus({ enabled, state }: { enabled: boolean; state: SemanticSettingsSnapshot["state"] | FaceSettingsSnapshot["state"] }) {
  const { t } = useLocale();
  return <InlineStatus tone={!enabled ? "info" : state === "ready" ? "success" : state === "degraded" ? "warning" : "info"}>{t(!enabled ? "ai.disabled" : `ai.state.${state}`)}</InlineStatus>;
}

export function AIServiceSettings({ models, activeModelId, activeFaceModelId, children }: { models: AIModel[]; activeModelId: string | null; activeFaceModelId: string | null; children?: ReactNode }) {
  const { t } = useLocale();
  return <div className={styles.overview}>
    <section className={styles.feature} aria-labelledby="ai-global-title">
      <header className={styles.featureHeader}><Brain className={styles.featureIcon} size={24} aria-hidden="true" /><div><h2 id="ai-global-title">{t("ai.settings.globalService")}</h2><p>{t("ai.settings.globalServiceDescription")}</p></div><span className={styles.scopeLabel}>{t("ai.settings.deploymentManaged")}</span></header>
      <ModelRow label={t("ai.settings.semanticModel")} model={models.find(item => item.id === activeModelId && item.active)} />
      <ModelRow label={t("ai.settings.peopleModel")} model={models.find(item => item.id === activeFaceModelId && item.active)} />
    </section>
    {children}
  </div>;
}

function ModelRow({ label, model }: { label: string; model: AIModel | undefined }) {
  const { t } = useLocale();
  return <div className={styles.modelRow}><strong>{label}</strong>{!model && <span>{t("ai.settings.noActiveModel")}</span>}<InlineStatus tone={model?.state === "available" ? "success" : "warning"}>{t(model?.state === "available" ? "ai.state.ready" : "ai.settings.serviceUnavailable")}</InlineStatus></div>;
}
