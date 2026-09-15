import { Folder, Info, LockSimple, MagnifyingGlass, Users } from "@phosphor-icons/react";
import { Button, EmptyState, ErrorState, InlineStatus, LoadingState, Select, Switch } from "../../../components/ui";
import type { AIModel, FaceSettingsSnapshot, SemanticSettingsSnapshot } from "../../../lib/api/ai";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import styles from "./AISettingsOverview.module.css";
import { VideoSemanticProgress } from "./VideoSemanticProgress";

type SnapshotQuery<T> = { data: T | undefined; isPending: boolean; isError: boolean; refetch: () => unknown };

export function AISettingsOverview({ libraries, semantic, faces, selectedId, onSelect, onManage, models, activeModelId, activeFaceModelId, runtimeAvailable, pending, onSemantic, onFace, onVideoJob, onView, hasMore, loadingMore, onMore }: {
  libraries: LibrarySummary[];
  semantic: Array<SnapshotQuery<SemanticSettingsSnapshot>>;
  faces: Array<SnapshotQuery<FaceSettingsSnapshot>>;
  selectedId: string | null;
  onSelect: (id: string) => void;
  onManage: (id: string) => void;
  models: AIModel[];
  activeModelId: string | null;
  activeFaceModelId: string | null;
  runtimeAvailable: boolean | undefined;
  pending: boolean;
  onSemantic: (library: LibrarySummary, settings: SemanticSettingsSnapshot, enabled: boolean) => void;
  onFace: (library: LibrarySummary, settings: FaceSettingsSnapshot, enabled: boolean) => void;
  onVideoJob: (library: LibrarySummary, mode: "missing" | "all") => void;
  onView: (view: "models" | "libraries" | "faces" | "tasks") => void;
  hasMore: boolean;
  loadingMore: boolean;
  onMore: () => void;
}) {
  const { locale, t } = useLocale();
  const number = new Intl.NumberFormat(locale);
  const selected = Math.max(0, libraries.findIndex(item => item.id === selectedId));
  const library = libraries[selected];
  const semanticQuery = semantic[selected];
  const faceQuery = faces[selected];
  const model = (id: string | null) => models.find(item => item.id === id && item.active && item.state !== "unavailable");
  const semanticModel = model(activeModelId);
  const faceModel = model(activeFaceModelId);
  return <div className={styles.overview}>
    <section className={styles.section} aria-labelledby="ai-capabilities-heading">
      <div className={styles.sectionHeading}><div><h2 id="ai-capabilities-heading">{t("ai.settings.capabilities")}</h2><p>{t("ai.settings.scope")}</p></div>
        {library && <label className={styles.selection}>{t("ai.settings.currentLibrary")}<Select value={library.id} onChange={event => onSelect(event.currentTarget.value)} aria-label={t("ai.settings.currentLibrary")}>
          {libraries.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
        </Select></label>}
      </div>
      {!library ? <EmptyState title={t("ai.noLibraries")} description={t("ai.noLibrariesDescription")} /> : <>
        {library.status === "offline" && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
        <div className={styles.rows}>
          <div className={styles.capability}>
            <span className={styles.icon}><MagnifyingGlass size={28} aria-hidden="true" /></span>
            <div><h3>{t("ai.settings.semantic")}</h3><p>{t("ai.settings.semanticDescription")}</p></div>
            <div className={styles.model}><span>{t("ai.settings.currentModel")}</span><strong>{semanticModel ? `${t(`ai.purpose.${semanticModel.purpose}`)} · ${semanticModel.version}` : t("ai.settings.noActiveModel")}</strong><small>{t("ai.settings.serverReported")}</small></div>
            {semanticQuery?.data && !semanticQuery.isError && !semanticQuery.isPending && <label className={styles.toggle}><Switch aria-label={t("ai.toggleLibrary").replace("{name}", library.name)} checked={semanticQuery.data.enabled}
              disabled={pending || library.status === "offline" || semanticQuery.data.state === "clearing" || (!semanticQuery.data.enabled && (!semanticModel || (semanticModel.storageMode !== "service" && !runtimeAvailable)))}
              onChange={event => onSemantic(library, semanticQuery.data!, event.currentTarget.checked)} /><span>{t(semanticQuery.data.enabled ? "ai.enabled" : "ai.disabled")}</span></label>}
          </div>
          {semanticQuery?.isPending && <LoadingState label={t("ai.loadingStatus")} />}
          {(!semanticQuery || semanticQuery.isError) && <ErrorState message={t("ai.statusFailed")} onRetry={() => void semanticQuery?.refetch()} />}
          {semanticQuery?.data && !semanticQuery.isError && !semanticQuery.isPending && semanticQuery.data.state !== "disabled" && <p className={styles.state}>{t("ai.state")}: {t(`ai.state.${semanticQuery.data.state}`)}</p>}
          <div className={styles.capability}>
            <span className={styles.icon}><Users size={28} aria-hidden="true" /></span>
            <div><h3>{t("ai.settings.people")}</h3><p>{t("ai.settings.peopleDescription")}</p></div>
            <div className={styles.model}><span>{t("ai.settings.currentModel")}</span><strong>{faceModel ? `${t(`ai.purpose.${faceModel.purpose}`)} · ${faceModel.version}` : t("ai.settings.noActiveModel")}</strong><small>{t("ai.settings.serverReported")}</small></div>
            {faceQuery?.data && !faceQuery.isError && !faceQuery.isPending && <label className={styles.toggle}><Switch aria-label={t("ai.faceToggle").replace("{name}", library.name)} checked={faceQuery.data.enabled}
              disabled={pending || library.status === "offline" || faceQuery.data.state === "clearing" || (!faceQuery.data.enabled && !faceModel)}
              onChange={event => onFace(library, faceQuery.data!, event.currentTarget.checked)} /><span>{t(faceQuery.data.enabled ? "ai.enabled" : "ai.disabled")}</span></label>}
          </div>
          {faceQuery?.isPending && <LoadingState label={t("ai.loadingStatus")} />}
          {(!faceQuery || faceQuery.isError) && <ErrorState message={t("ai.statusFailed")} onRetry={() => void faceQuery?.refetch()} />}
          {faceQuery?.data && !faceQuery.isError && !faceQuery.isPending && faceQuery.data.state !== "disabled" && <p className={styles.state}>{t("ai.state")}: {t(`ai.state.${faceQuery.data.state}`)}</p>}
        </div>
      </>}
    </section>
    <section className={styles.section} aria-labelledby="ai-library-analysis-heading">
      <div className={styles.sectionHeading}><div><h2 id="ai-library-analysis-heading">{t("ai.settings.libraryAnalysis")}</h2><p>{t("ai.settings.analysisDescription")}</p></div></div>
      <div className={styles.rows}>{libraries.map((item, index) => {
        const query = semantic[index];
        const settings = query?.data;
        return <article className={styles.library} key={item.id}>
          <span className={styles.icon}><Folder size={26} aria-hidden="true" /></span>
          <div><h3>{item.name}</h3><p>{item.status === "offline" ? t("ai.libraryOffline") : settings && !query?.isError ? t(`ai.state.${settings.state}`) : t("ai.loadingStatus")}</p></div>
          <div className={styles.coverage}>
            {query?.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : !query || query.isError ? <ErrorState message={t("ai.statusFailed")} onRetry={() => void query?.refetch()} /> : settings?.coverageRecorded === false ? <InlineStatus>{t("ai.image.unrecorded")}</InlineStatus> : settings && <>
              <progress aria-label={`${item.name} ${t("ai.settings.libraryAnalysis")} ${t("ai.coverage")}`} max={Math.max(1, settings.coverage.eligible)} value={settings.coverage.completed} />
              <span>{number.format(settings.coverage.completed)} / {number.format(settings.coverage.eligible)} · {t("ai.failed")} {number.format(settings.coverage.failed)}</span>
            </>}
          </div>
          <Button size="small" variant="quiet" onClick={() => onManage(item.id)}>{t("ai.settings.manageAnalysis")}</Button>
        </article>;
      })}</div>
      {hasMore && <div className={styles.actions}><Button loading={loadingMore} onClick={onMore}>{t("ai.moreLibraries")}</Button><span>{t("ai.overview.partialLibraries")}</span></div>}
    </section>
    {library && <VideoSemanticProgress library={library} disabled={pending || !semanticModel || !semanticQuery?.data || semanticQuery.isError || semanticQuery.isPending || semanticQuery.data.state === "clearing"}
      onJob={(mode) => onVideoJob(library, mode)} />}
    <details className={styles.deployment}><summary>{t("ai.settings.deployment")}</summary>
      <InlineStatus>{t("ai.settings.migrationNotice")}</InlineStatus>
      <p>{t("ai.settings.legacyDescription")}</p><div className={styles.actions}><Button onClick={() => onView("models")}>{t("ai.settings.legacyMaintenance")}</Button><Button variant="quiet" onClick={() => onView("faces")}>{t("ai.settings.peopleDetails")}</Button></div>
    </details>
    <footer className={styles.footer}><span><LockSimple size={17} aria-hidden="true" />{t("ai.settings.readOnly")}</span><span><Info size={17} aria-hidden="true" />{t("ai.settings.realData")}</span></footer>
  </div>;
}
