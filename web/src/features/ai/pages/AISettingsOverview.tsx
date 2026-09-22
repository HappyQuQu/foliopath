import {
  ArrowRight,
  Brain,
  CaretDown,
  CheckCircle,
  Database,
  LockSimple,
  MagnifyingGlass,
  Users,
  WarningCircle,
} from "@phosphor-icons/react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { Button, EmptyState, ErrorState, InlineStatus, LoadingState, Select, Switch } from "../../../components/ui";
import type { AIModel, FaceSettingsSnapshot, SemanticSettingsSnapshot } from "../../../lib/api/ai";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import styles from "./AISettingsOverview.module.css";

type SnapshotQuery<T> = { data: T | undefined; isPending: boolean; isError: boolean; refetch: () => unknown };

export function AISettingsOverview({ libraries, semantic, faces, selectedId, onSelect, models, activeModelId, activeFaceModelId, pending, onSemantic, onFace, onView }: {
  libraries: LibrarySummary[];
  semantic: Array<SnapshotQuery<SemanticSettingsSnapshot>>;
  faces: Array<SnapshotQuery<FaceSettingsSnapshot>>;
  selectedId: string | null;
  onSelect: (id: string) => void;
  models: AIModel[];
  activeModelId: string | null;
  activeFaceModelId: string | null;
  pending: boolean;
  onSemantic: (library: LibrarySummary, settings: SemanticSettingsSnapshot, enabled: boolean) => void;
  onFace: (library: LibrarySummary, settings: FaceSettingsSnapshot, enabled: boolean) => void;
  onView: (view: "libraries" | "faces" | "tasks") => void;
}) {
  const { locale, t } = useLocale();
  const number = new Intl.NumberFormat(locale);
  const selected = Math.max(0, libraries.findIndex(item => item.id === selectedId));
  const library = libraries[selected];
  const semanticQuery = semantic[selected];
  const faceQuery = faces[selected];
  const activeModel = (id: string | null) => models.find(item => item.id === id && item.active);
  const semanticModel = activeModel(activeModelId);
  const faceModel = activeModel(activeFaceModelId);
  const semanticReady = semanticModel?.storageMode === "service" && semanticModel.state === "available";
  const faceReady = faceModel?.storageMode === "service" && faceModel.state === "available";
  const serviceStatus = semanticReady && faceReady ? t("ai.settings.serviceReady") : semanticReady || faceReady ? t("ai.settings.servicePartial") : t("ai.settings.serviceUnavailable");

  return <div className={styles.overview}>
    <section className={styles.statusCard} aria-label={t("ai.settings.serviceStatus")}>
      <span className={semanticReady || faceReady ? styles.statusReady : styles.statusWarning}>
        {semanticReady || faceReady ? <CheckCircle size={22} weight="fill" aria-hidden="true" /> : <WarningCircle size={22} weight="fill" aria-hidden="true" />}
      </span>
      <div><strong>{serviceStatus}</strong><span>{t("ai.settings.serviceSummary")}</span></div>
      <span className={styles.deploymentBadge}>{t("ai.settings.composeBadge")}</span>
    </section>

    <section className={styles.scope} aria-labelledby="ai-library-settings-heading">
      <div><h2 id="ai-library-settings-heading">{t("ai.settings.libraryFeatures")}</h2><p>{t("ai.settings.libraryScope")}</p></div>
      {library && <label className={styles.selection}><span>{t("ai.settings.currentLibrary")}</span><Select value={library.id} onChange={event => onSelect(event.currentTarget.value)} aria-label={t("ai.settings.currentLibrary")}>
        {libraries.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
      </Select></label>}
    </section>

    {!library ? <EmptyState title={t("ai.noLibraries")} description={t("ai.noLibrariesDescription")} /> : <>
      {library.status === "offline" && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
      <div className={styles.sections} aria-label={t("ai.settings.libraryFeatures")}>
        <SettingsSection defaultOpen icon={<MagnifyingGlass size={22} aria-hidden="true" />} title={t("ai.settings.semantic")} description={t("ai.settings.semanticSectionDescription")} status={semanticQuery?.data ? t(semanticQuery.data.enabled ? "ai.enabled" : "ai.disabled") : t("ai.loadingStatus")}>
          {semanticQuery?.isPending && <LoadingState label={t("ai.loadingStatus")} />}
          {(!semanticQuery || semanticQuery.isError) && <ErrorState message={t("ai.statusFailed")} onRetry={() => void semanticQuery?.refetch()} />}
          {semanticQuery?.data && !semanticQuery.isError && !semanticQuery.isPending && <>
            <SettingRow title={t("ai.settings.enableSemantic")} description={t("ai.settings.enableSemanticDescription")}>
              <Switch aria-label={t("ai.toggleLibrary").replace("{name}", library.name)} checked={semanticQuery.data.enabled}
                disabled={pending || library.status === "offline" || semanticQuery.data.state === "clearing" || (!semanticQuery.data.enabled && !semanticReady)}
                onChange={event => onSemantic(library, semanticQuery.data!, event.currentTarget.checked)} />
            </SettingRow>
            <ModelRow label={t("ai.settings.semanticModel")} model={semanticModel} />
            <div className={styles.progressRow}><div><strong>{t("ai.settings.indexProgress")}</strong>
              {semanticQuery.data.coverageRecorded === false ? <span>{t("ai.settings.progressUnknown")}</span> : <span>{number.format(semanticQuery.data.coverage.completed)} / {number.format(semanticQuery.data.coverage.eligible)} {t("ai.settings.processed")} · {number.format(semanticQuery.data.coverage.failed)} {t("ai.failed")}</span>}
            </div>{semanticQuery.data.coverageRecorded !== false && <progress aria-label={t("ai.settings.indexProgress")} max={Math.max(1, semanticQuery.data.coverage.eligible)} value={semanticQuery.data.coverage.completed} />}</div>
            {!semanticReady && <InlineStatus tone="warning">{t("ai.settings.semanticUnavailable")}</InlineStatus>}
            <div className={styles.sectionActions}><Link className={styles.primaryLink} to={`${paths.search}?mode=semantic`}>{t("ai.settings.openSearch")}<ArrowRight size={15} aria-hidden="true" /></Link><Button variant="quiet" onClick={() => onView("libraries")}>{t("ai.settings.repairSemantic")}</Button></div>
          </>}
        </SettingsSection>

        <SettingsSection icon={<Users size={22} aria-hidden="true" />} title={t("ai.settings.people")} description={t("ai.settings.peopleSectionDescription")} status={faceQuery?.data ? t(faceQuery.data.enabled ? "ai.enabled" : "ai.disabled") : t("ai.loadingStatus")}>
          {faceQuery?.isPending && <LoadingState label={t("ai.loadingStatus")} />}
          {(!faceQuery || faceQuery.isError) && <ErrorState message={t("ai.statusFailed")} onRetry={() => void faceQuery?.refetch()} />}
          {faceQuery?.data && !faceQuery.isError && !faceQuery.isPending && <>
            <SettingRow title={t("ai.settings.enablePeople")} description={t("ai.settings.enablePeopleDescription")}>
              <Switch aria-label={t("ai.faceToggle").replace("{name}", library.name)} checked={faceQuery.data.enabled}
                disabled={pending || library.status === "offline" || faceQuery.data.state === "clearing" || (!faceQuery.data.enabled && !faceReady)}
                onChange={event => onFace(library, faceQuery.data!, event.currentTarget.checked)} />
            </SettingRow>
            <ModelRow label={t("ai.settings.peopleModel")} model={faceModel} />
            {!faceReady && <InlineStatus tone="warning">{t("ai.settings.peopleUnavailable")}</InlineStatus>}
            <p className={styles.note}>{t("ai.settings.qualityManaged")}</p>
            <div className={styles.sectionActions}><Link className={styles.primaryLink} to={`${paths.intelligence}?${new URLSearchParams({ view: "clusters", libraryId: library.id }).toString()}`}>{t("ai.settings.openPeople")}<ArrowRight size={15} aria-hidden="true" /></Link><Button variant="quiet" onClick={() => onView("faces")}>{t("ai.settings.repairPeople")}</Button></div>
          </>}
        </SettingsSection>

        <SettingsSection icon={<Database size={22} aria-hidden="true" />} title={t("ai.settings.tasksMaintenance")} description={t("ai.settings.tasksMaintenanceDescription")} status={t("ai.settings.perLibrary")}>
          <div className={styles.maintenanceGrid}>
            <div><strong>{t("ai.settings.imageMaintenance")}</strong><span>{t("ai.settings.imageMaintenanceDescription")}</span></div>
            <div><strong>{t("ai.settings.peopleDetails")}</strong><span>{t("ai.settings.peopleMaintenanceDescription")}</span></div>
          </div>
          <div className={styles.sectionActions}><Button onClick={() => onView("tasks")}>{t("ai.settings.progress")}</Button></div>
        </SettingsSection>
      </div>
    </>}

    <section className={styles.globalHeading} aria-labelledby="ai-global-settings-heading"><div><h2 id="ai-global-settings-heading">{t("ai.settings.globalService")}</h2><p>{t("ai.settings.globalServiceDescription")}</p></div><span>{t("ai.settings.deploymentManaged")}</span></section>
    <div className={styles.sections}>
      <SettingsSection icon={<Brain size={22} aria-hidden="true" />} title={t("ai.settings.serviceModels")} description={t("ai.settings.serviceModelsDescription")} status={serviceStatus}>
        <ModelRow label={t("ai.settings.semanticModel")} model={semanticModel} />
        <ModelRow label={t("ai.settings.peopleModel")} model={faceModel} />
        <p className={styles.note}>{t("ai.settings.composeDescription")}</p>
      </SettingsSection>
    </div>

    <footer className={styles.footer}><span><LockSimple size={17} aria-hidden="true" />{t("ai.settings.readOnly")}</span></footer>
  </div>;
}

function SettingsSection({ children, defaultOpen = false, description, icon, status, title }: { children: ReactNode; defaultOpen?: boolean; description: string; icon: ReactNode; status: string; title: string }) {
  return <details className={styles.settingsSection} open={defaultOpen || undefined}>
    <summary><span className={styles.sectionIcon}>{icon}</span><span className={styles.sectionTitle}><strong>{title}</strong><small>{description}</small></span><span className={styles.sectionStatus}>{status}</span><CaretDown className={styles.caret} size={18} aria-hidden="true" /></summary>
    <div className={styles.sectionBody}>{children}</div>
  </details>;
}

function SettingRow({ children, description, title }: { children: ReactNode; description: string; title: string }) {
  return <div className={styles.settingRow}><div><strong>{title}</strong><span>{description}</span></div>{children}</div>;
}

function ModelRow({ label, model }: { label: string; model: AIModel | undefined }) {
  const { locale, t } = useLocale();
  const size = model ? new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(model.packageSizeBytes / 1024 / 1024) : "";
  return <div className={styles.modelRow}><span>{label}</span>{model ? <div><strong>{t(`ai.purpose.${model.purpose}`)} · {model.version}</strong><small>{model.architecture} · {size} MiB · {model.licenseId}</small></div> : <strong>{t("ai.settings.noActiveModel")}</strong>}<small>{t("ai.settings.deploymentManaged")}</small></div>;
}
