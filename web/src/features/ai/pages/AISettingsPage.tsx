import {
  ArrowsClockwise,
  Play,
  Trash,
} from "@phosphor-icons/react";
import { useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import {
  Button,
  Dialog,
  DialogCloseButton,
  EmptyState,
  ErrorState,
  InlineStatus,
  Input,
  LoadingState,
  Switch,
  useToast,
} from "../../../components/ui";
import type {
  AIModelOperation,
  FaceSettingsSnapshot,
  SemanticSettingsSnapshot,
} from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { ApiError } from "../../../lib/api/errors";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import {
  readAIOperationIds,
  writeAIOperationIds,
} from "../../../lib/storage/preferences";
import { paths } from "../../../routes/paths";
import { useLibrariesQuery } from "../../libraries";
import {
  operationIsActive,
  useAIOperationQueries,
  useAIModelsQuery,
  useAIModelDownloadsQuery,
  useCancelAIOperationMutation,
  useClearDerivedFaceDataMutation,
  useClearManualFaceRelationshipsMutation,
  useClearSemanticDataMutation,
  useFaceSettingsQueries,
  useRequestSemanticJobMutation,
  useRequestVideoSemanticJobMutation,
  useRequestFaceJobMutation,
  useSemanticSettingsQueries,
  useUpdateSemanticSettingsMutation,
  useUpdateFaceSettingsMutation,
} from "../queries";
import styles from "./AISettingsPage.module.css";
import { AISettingsOverview } from "./AISettingsOverview";
import { VideoSemanticProgress } from "./VideoSemanticProgress";

type AIView = "overview" | "libraries" | "tasks" | "faces";
type Confirmation =
  | { kind: "videoRebuild"; library: LibrarySummary }
  | { kind: "rebuild"; library: LibrarySummary; settings: SemanticSettingsSnapshot }
  | { kind: "clear"; library: LibrarySummary; settings: SemanticSettingsSnapshot }
  | null;

export function AISettingsPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending?: boolean;
  onLogout?: () => Promise<void>;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const [searchParams, setSearchParams] = useSearchParams();
  const view = parseView(searchParams.get("view"));
  const librariesQuery = useLibrariesQuery();
  const libraries = useMemo(
    () => librariesQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [librariesQuery.data],
  );
  const semanticQueries = useSemanticSettingsQueries(libraries.map((library) => library.id));
  const faceQueries = useFaceSettingsQueries(libraries.map((library) => library.id));
  const modelsQuery = useAIModelsQuery();
  const downloadsQuery = useAIModelDownloadsQuery();
  const [operationIds, setOperationIds] = useState(readAIOperationIds);
  const trackedOperationIds = useMemo(() => [...new Set([
    ...(downloadsQuery.data?.items.flatMap((item) => item.operation ? [item.operation.id] : []) ?? []),
    ...operationIds,
  ])].slice(0, 66), [downloadsQuery.data, operationIds]);
  const operationQueries = useAIOperationQueries(trackedOperationIds);
  const [confirmation, setConfirmation] = useState<Confirmation>(null);
  const toast = useToast();
  const updateSettings = useUpdateSemanticSettingsMutation();
  const requestJob = useRequestSemanticJobMutation();
  const requestVideo = useRequestVideoSemanticJobMutation();
  const clear = useClearSemanticDataMutation();
  const cancel = useCancelAIOperationMutation();
  const updateFaces = useUpdateFaceSettingsMutation();
  const requestFaces = useRequestFaceJobMutation();
  const clearFaceDerived = useClearDerivedFaceDataMutation();
  const clearFaceManual = useClearManualFaceRelationshipsMutation();

  function navigateView(item: AIView) {
    setSearchParams(current => {
      const next = new URLSearchParams(current);
      if (item === "overview") next.delete("view"); else next.set("view", item);
      return next;
    });
  }

  function selectLibrary(id: string) {
    setSearchParams(current => { const next = new URLSearchParams(current); next.set("library", id); return next; });
  }

  async function toggleSemantic(library: LibrarySummary, settings: SemanticSettingsSnapshot, enabled: boolean) {
    try {
      await updateSettings.mutateAsync({ csrfToken: session.csrfToken, enabled, etag: settings.etag, libraryId: library.id });
      if (enabled) {
        const operation = await requestJob.mutateAsync({
          csrfToken: session.csrfToken,
          idempotencyKey: createRequestKey(),
          libraryId: library.id,
          mode: "missing",
        });
        rememberOperation(operation);
        toast.show({ message: t("ai.jobStarted"), tone: "success" });
        return;
      }
      toast.show({ message: t("ai.settingSaved"), tone: "success" });
    } catch (error) {
      toast.show({ message: actionErrorMessage(error, t), tone: "danger" });
    }
  }

  async function toggleFace(library: LibrarySummary, settings: FaceSettingsSnapshot, enabled: boolean) {
    try {
      await updateFaces.mutateAsync({ csrfToken: session.csrfToken, enabled, etag: settings.etag, libraryId: library.id });
      if (enabled) {
        const operation = await requestFaces.mutateAsync({
          csrfToken: session.csrfToken,
          idempotencyKey: createRequestKey(),
          libraryId: library.id,
          mode: "missing",
        });
        rememberOperation(operation);
        toast.show({ message: t("ai.jobStarted"), tone: "success" });
        return;
      }
      toast.show({ message: t("ai.settingSaved"), tone: "success" });
    } catch (error) {
      toast.show({ message: actionErrorMessage(error, t), tone: "danger" });
    }
  }

  function rememberOperation(operation: AIModelOperation) {
    setOperationIds((current) => {
      const next = [operation.id, ...current.filter((id) => id !== operation.id)].slice(0, 50);
      writeAIOperationIds(next);
      return next;
    });
  }

  async function runAction(action: () => Promise<AIModelOperation>, successKey: MessageKey) {
    try {
      const operation = await action();
      rememberOperation(operation);
      setConfirmation(null);
      toast.show({ message: t(successKey), tone: "success" });
    } catch (error) {
      toast.show({ message: actionErrorMessage(error, t), tone: "danger" });
    }
  }

  return (
    <ManagementShell
      active="ai"
      accountHref={paths.accountSettings}
      aiHref={paths.aiSettings}
      generalHref={paths.generalSettings}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      storageHref={paths.storageSettings}
    >
      <div className={styles.main}>
        <div className={styles.headerRow}><header className={styles.heading}>
          <p>{t("ai.eyebrow")}</p>
          <h1>{t("ai.title")}</h1>
          <span>{t("ai.description")}</span>
        </header><div className={styles.headerActions}>
          <Button variant="quiet" onClick={() => navigateView("tasks")}><ArrowsClockwise size={18} aria-hidden="true" />{t("ai.settings.progress")}</Button>
        </div></div>

        {view !== "overview" && <nav className={styles.actions} aria-label={t("ai.sections")}><Button onClick={() => navigateView("overview")}>{t("ai.settings.back")}</Button><h2>{t(`ai.tab.${view}`)}</h2></nav>}

        <div id="ai-settings-panel" className={styles.panel}>
        {view === "overview" && (librariesQuery.isError || modelsQuery.isError || semanticQueries.some((query) => query.isError)) &&
          <ErrorState message={t("ai.overview.loadFailed")} onRetry={() => { void librariesQuery.refetch(); void modelsQuery.refetch(); semanticQueries.forEach((query) => void query.refetch()); }} />}
        {view === "overview" && (librariesQuery.isPending || modelsQuery.isPending || semanticQueries.some((query) => query.isPending)) && <LoadingState label={t("ai.loadingStatus")} />}
        {view === "overview" && !librariesQuery.isPending && !librariesQuery.isError && !modelsQuery.isPending && !modelsQuery.isError && <AISettingsOverview
          libraries={libraries} semantic={semanticQueries} faces={faceQueries}
          models={modelsQuery.data?.items ?? []} activeModelId={modelsQuery.data?.activeModelId ?? null} activeFaceModelId={modelsQuery.data?.activeFaceModelId ?? null}
          selectedId={searchParams.get("library")} onSelect={selectLibrary}
          pending={requestVideo.isPending || updateSettings.isPending || updateFaces.isPending || requestJob.isPending || requestFaces.isPending || clear.isPending || clearFaceDerived.isPending || clearFaceManual.isPending}
          onSemantic={toggleSemantic} onFace={toggleFace}
          onView={navigateView}
        />}

        {view === "libraries" && (
          <LibrariesPanel
            libraries={searchParams.has("library") ? libraries.filter(item => item.id === searchParams.get("library")) : libraries}
            librariesPending={librariesQuery.isPending}
            librariesError={librariesQuery.isError}
            queries={searchParams.has("library") ? semanticQueries.filter((_, index) => libraries[index]?.id === searchParams.get("library")) : semanticQueries}
            actionPending={updateSettings.isPending || requestJob.isPending || clear.isPending}
            onClear={(library, settings) => setConfirmation({ kind: "clear", library, settings })}
            onMissing={(library) => void runAction(() => requestJob.mutateAsync({
              csrfToken: session.csrfToken,
              idempotencyKey: createRequestKey(),
              libraryId: library.id,
              mode: "missing",
            }), "ai.jobStarted")}
            onRebuild={(library, settings) => setConfirmation({ kind: "rebuild", library, settings })}
            onVideoJob={(library, mode) => {
              if (mode === "all") { setConfirmation({ kind: "videoRebuild", library }); return; }
              void runAction(() => requestVideo.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: library.id, mode }), "ai.jobStarted");
            }}
            onToggle={toggleSemantic}
            onRetry={() => void librariesQuery.refetch()}
            videoReady={modelsQuery.data?.items.some((item) => item.id === modelsQuery.data?.activeModelId && item.active && item.state !== "unavailable" && item.storageMode === "service") ?? false}
          />
        )}

        {view === "faces" && (
          <FaceSettingsPanel
            libraries={libraries}
            queries={faceQueries}
            pending={updateFaces.isPending || requestFaces.isPending || clearFaceDerived.isPending || clearFaceManual.isPending}
            onToggle={toggleFace}
            onJob={(library, mode) => void runAction(() => requestFaces.mutateAsync({
              csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: library.id, mode,
            }), "ai.jobStarted")}
            onDerived={(library, settings) => void runAction(() => clearFaceDerived.mutateAsync({
              csrfToken: session.csrfToken, etag: settings.etag, idempotencyKey: createRequestKey(), libraryId: library.id,
            }), "ai.clearStarted")}
            onManual={(library, settings, counts) => void runAction(() => clearFaceManual.mutateAsync({
              ...counts, csrfToken: session.csrfToken, etag: settings.etag,
              idempotencyKey: createRequestKey(), libraryId: library.id,
            }), "ai.clearStarted")}
          />
        )}

        {view === "tasks" && (
          <TasksPanel
            cancelPending={cancel.isPending}
            operationIds={trackedOperationIds}
            removableIds={operationIds}
            onForget={(id) => setOperationIds((current) => {
              const next = current.filter((value) => value !== id);
              writeAIOperationIds(next);
              return next;
            })}
            queries={operationQueries}
            onCancel={(operation) => void cancel.mutateAsync({
              csrfToken: session.csrfToken,
              etag: operation.etag,
              operationId: operation.id,
            }).catch((error) => toast.show({ message: actionErrorMessage(error, t), tone: "danger" }))}
          />
        )}
        {(view === "libraries" || view === "faces") && librariesQuery.hasNextPage && <Button loading={librariesQuery.isFetchingNextPage} onClick={() => void librariesQuery.fetchNextPage()}>{t("ai.moreLibraries")}</Button>}
        </div>
      </div>

      <ConfirmationDialog
        confirmation={confirmation}
        pending={requestJob.isPending || requestVideo.isPending || clear.isPending}
        onClose={() => setConfirmation(null)}
        onConfirm={() => {
          if (!confirmation) return;
          if (confirmation.kind === "videoRebuild") {
            void runAction(() => requestVideo.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: confirmation.library.id, mode: "all" }), "ai.jobStarted");
          } else if (confirmation.kind === "rebuild") {
            void runAction(() => requestJob.mutateAsync({
              csrfToken: session.csrfToken,
              idempotencyKey: createRequestKey(),
              libraryId: confirmation.library.id,
              mode: "all",
            }), "ai.jobStarted");
          } else {
            void runAction(() => clear.mutateAsync({
              csrfToken: session.csrfToken,
              etag: confirmation.settings.etag,
              idempotencyKey: createRequestKey(),
              libraryId: confirmation.library.id,
            }), "ai.clearStarted");
          }
        }}
      />
    </ManagementShell>
  );
}

function LibrariesPanel({
  actionPending,
  libraries,
  librariesError,
  librariesPending,
  onClear,
  onMissing,
  onRebuild,
  onRetry,
  onToggle,
  onVideoJob,
  queries,
  videoReady,
}: {
  actionPending: boolean;
  libraries: LibrarySummary[];
  librariesError: boolean;
  librariesPending: boolean;
  onClear: (library: LibrarySummary, settings: SemanticSettingsSnapshot) => void;
  onMissing: (library: LibrarySummary) => void;
  onRebuild: (library: LibrarySummary, settings: SemanticSettingsSnapshot) => void;
  onRetry: () => void;
  onToggle: (library: LibrarySummary, settings: SemanticSettingsSnapshot, enabled: boolean) => void;
  onVideoJob: (library: LibrarySummary, mode: "missing" | "all") => void;
  queries: Array<{ data: SemanticSettingsSnapshot | undefined; isError: boolean; isPending: boolean; refetch: () => unknown }>;
  videoReady: boolean;
}) {
  const { locale, t } = useLocale();
  const number = useMemo(() => new Intl.NumberFormat(locale), [locale]);
  if (librariesPending) return <LoadingState label={t("ai.loadingLibraries")} />;
  if (librariesError) return <ErrorState message={t("ai.loadFailed")} onRetry={onRetry} />;
  if (libraries.length === 0) return <EmptyState title={t("ai.noLibraries")} description={t("ai.noLibrariesDescription")} />;

  return <div className={styles.cardList}>
    {libraries.map((library, index) => {
      const query = queries[index];
      const settings = query?.data;
      return <article className={styles.card} key={library.id}>
        <header className={styles.cardHeader}>
          <div>
            <h2>{library.name}</h2>
            <p>{library.status === "offline" ? t("ai.libraryOffline") : t("ai.semanticDescription")}</p>
          </div>
          {settings && <label className={styles.switchLabel}>
            <span>{settings.enabled ? t("ai.enabled") : t("ai.disabled")}</span>
            <Switch
              aria-label={formatMessage(t("ai.toggleLibrary"), { name: library.name })}
              checked={settings.enabled}
              disabled={actionPending || settings.state === "clearing" || library.status === "offline"}
              onChange={(event) => onToggle(library, settings, event.currentTarget.checked)}
            />
          </label>}
        </header>
        {query?.isPending && <LoadingState label={t("ai.loadingStatus")} />}
        {query?.isError && <ErrorState message={t("ai.statusFailed")} onRetry={() => void query.refetch()} />}
        {settings && <>
          {settings.coverageRecorded === false ? <InlineStatus>{t("ai.image.unrecorded")}</InlineStatus> : <>
          <div className={styles.metrics}>
            <Metric label={t("ai.state")} value={t(`ai.state.${settings.state}`)} />
            <Metric label={t("ai.coverage")} value={`${number.format(settings.coverage.completed)} / ${number.format(settings.coverage.eligible)}`} />
            <Metric label={t("ai.failed")} value={number.format(settings.coverage.failed)} />
          </div>
          <div className={styles.progress}>
            <div aria-hidden="true"><span style={{ width: `${coveragePercent(settings)}%` }} /></div>
            <span>{formatMessage(t("ai.coverageDetail"), {
              completed: number.format(settings.coverage.completed),
              eligible: number.format(settings.coverage.eligible),
              stale: number.format(settings.coverage.stale),
            })}</span>
          </div>
          </>}
          {settings.state === "degraded" && <InlineStatus tone="warning">{t("ai.degradedNotice")}</InlineStatus>}
          {library.status === "offline" && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
          <div className={styles.actions}>
            <Button disabled={actionPending || !settings.enabled || library.status === "offline"} onClick={() => onMissing(library)}>
              <Play aria-hidden="true" size={17} />{t("ai.fillMissing")}
            </Button>
          </div>
          <details className={styles.details}><summary>{t("ai.maintenance")}</summary>
            <p>{t("ai.maintenanceDescription")}</p>
            <div className={styles.actions}>
            <Button disabled={actionPending || !settings.enabled || library.status === "offline"} onClick={() => onRebuild(library, settings)}>
              <ArrowsClockwise aria-hidden="true" size={17} />{t("ai.rebuildAll")}
            </Button>
            <Button disabled={actionPending || settings.state === "clearing"} onClick={() => onClear(library, settings)} variant="danger">
              <Trash aria-hidden="true" size={17} />{t("ai.clearData")}
            </Button>
          </div>
          </details>
          <VideoSemanticProgress library={library} disabled={actionPending || !videoReady || !settings.enabled || library.status === "offline" || settings.state === "clearing"} onJob={(mode) => onVideoJob(library, mode)} />
        </>}
      </article>;
    })}
  </div>;
}

function FaceSettingsPanel({ libraries, onDerived, onJob, onManual, onToggle, pending, queries }: {
  libraries: LibrarySummary[];
  pending: boolean;
  queries: Array<{ data: FaceSettingsSnapshot | undefined; isError: boolean; isPending: boolean; refetch: () => unknown }>;
  onToggle: (library: LibrarySummary, settings: FaceSettingsSnapshot, enabled: boolean) => void;
  onJob: (library: LibrarySummary, mode: "missing" | "all") => void;
  onDerived: (library: LibrarySummary, settings: FaceSettingsSnapshot) => void;
  onManual: (library: LibrarySummary, settings: FaceSettingsSnapshot, counts: { assignmentCount: number; constraintCount: number; personCount: number }) => void;
}) {
  const { t } = useLocale();
  const [counts, setCounts] = useState({ assignmentCount: 0, constraintCount: 0, personCount: 0 });
  const [confirmation, setConfirmation] = useState<{ kind: "derived" | "manual"; library: LibrarySummary; settings: FaceSettingsSnapshot } | null>(null);
  return <div className={styles.cardList}>
    <InlineStatus>{t("ai.faceDescription")}</InlineStatus>
    <Link className={styles.textLink} to={paths.intelligence}>{t("ai.reviewWorkspace")}</Link>
    {libraries.map((library, index) => {
      const query = queries[index]; const settings = query?.data;
      return <article className={styles.card} key={library.id}>
        <header className={styles.cardHeader}><div><h2>{library.name}</h2><p>{t("ai.faceDescription")}</p></div>
          {settings && <Switch aria-label={formatMessage(t("ai.faceToggle"), { name: library.name })} checked={settings.enabled} disabled={pending || library.status === "offline"} onChange={(event) => onToggle(library, settings, event.currentTarget.checked)} />}
        </header>
        {query?.isPending && <LoadingState label={t("ai.loadingStatus")} />}
        {query?.isError && <ErrorState message={t("ai.statusFailed")} onRetry={() => void query.refetch()} />}
        {settings && <><div className={styles.metrics}>
          <Metric label={t("ai.state")} value={t(`ai.state.${settings.state}`)} />
          <Metric label={t("ai.coverage")} value={`${settings.coverage.completed} / ${settings.coverage.eligible}`} />
          <Metric label={t("ai.failed")} value={String(settings.coverage.failed)} />
          <Metric label={t("ai.generation")} value={settings.activeGenerationId ?? t("ai.none")} />
        </div><div className={styles.actions}>
          <Button disabled={pending || !settings.enabled} onClick={() => onJob(library, "missing")}>{t("ai.fillMissing")}</Button>
        </div><details className={styles.details}><summary>{t("ai.maintenance")}</summary><div className={styles.actions}>
          <Button disabled={pending || !settings.enabled} onClick={() => onJob(library, "all")}>{t("ai.rebuildAll")}</Button>
          <Button disabled={pending} variant="danger" onClick={() => setConfirmation({ kind: "derived", library, settings })}>{t("ai.clearFaceDerived")}</Button>
        </div><InlineStatus tone="warning">{t("ai.faceManualWarning")}</InlineStatus>
        <div className={styles.actions}>
          <Button disabled={pending} variant="danger" onClick={() => { setCounts({ assignmentCount: 0, constraintCount: 0, personCount: 0 }); setConfirmation({ kind: "manual", library, settings }); }}>{t("ai.clearFaceManual")}</Button>
        </div></details></>}
      </article>;
    })}
    <Dialog open={confirmation !== null} onOpenChange={(open) => { if (!open) setConfirmation(null); }}
      title={t(confirmation?.kind === "manual" ? "ai.clearFaceManual" : "ai.clearFaceDerived")}
      description={confirmation?.library.name ?? ""}
      actions={<><DialogCloseButton onClick={() => setConfirmation(null)} /><Button variant="danger" disabled={pending || Object.values(counts).some((value) => !Number.isSafeInteger(value) || value < 0)} onClick={() => {
        if (!confirmation) return;
        if (confirmation.kind === "manual") onManual(confirmation.library, confirmation.settings, counts);
        else onDerived(confirmation.library, confirmation.settings);
        setConfirmation(null);
      }}>{t(confirmation?.kind === "manual" ? "ai.clearFaceManual" : "ai.clearFaceDerived")}</Button></>}>
      <p>{t(confirmation?.kind === "manual" ? "ai.faceManualWarning" : "ai.faceDerivedWarning")}</p>
      {confirmation?.kind === "manual" && (["personCount", "assignmentCount", "constraintCount"] as const).map((key) => <label key={key}>{t(`ai.${key}`)}<Input min="0" step="1" type="number" value={counts[key]} onChange={(event) => setCounts((current) => ({ ...current, [key]: Number(event.target.value) }))} /></label>)}
    </Dialog>
  </div>;
}

function TasksPanel({ cancelPending, onCancel, onForget, operationIds, removableIds, queries }: {
  cancelPending: boolean;
  onCancel: (operation: AIModelOperation & { etag: string }) => void;
  operationIds: string[];
  removableIds: string[];
  onForget: (id: string) => void;
  queries: Array<{ data: (AIModelOperation & { etag: string }) | undefined; error?: unknown; isError: boolean; isPending: boolean; refetch: () => unknown }>;
}) {
  const { locale, t } = useLocale();
  if (operationIds.length === 0) return <EmptyState title={t("ai.noTasks")} description={t("ai.noTasksDescription")} />;
  return <div className={styles.cardList}>{operationIds.map((id, index) => {
    const query = queries[index];
    const operation = query?.data;
    if (query?.isPending) return <div className={styles.card} key={id}><LoadingState label={t("ai.loadingTask")} /></div>;
    if (query?.isError && query.error instanceof ApiError && query.error.status === 404 && query.error.code === "ai_operation_not_found") {
      return <div className={styles.card} key={id}>
        <EmptyState title={t("ai.taskNotFound")} description={t("ai.taskNotFoundDescription")} />
        {removableIds.includes(id) && <Button onClick={() => onForget(id)}>{t("ai.forgetTask")}</Button>}
      </div>;
    }
    if (query?.isError) return <div className={styles.card} key={id}><ErrorState message={t("ai.taskFailedToLoad")} onRetry={() => void query.refetch()} /></div>;
    if (!operation) return null;
    const percent = operation.totalItems && operation.totalItems > 0 ? Math.min(100, operation.completedItems / operation.totalItems * 100) : null;
    const amount = (value: number) => operation.kind === "model_install" ? formatBytes(value, new Intl.NumberFormat(locale, { maximumFractionDigits: 1 })) : new Intl.NumberFormat(locale).format(value);
    return <article className={styles.card} key={id}>
      <header className={styles.cardHeader}><div><h2>{t(`ai.operation.${operation.kind}`)}</h2><p>{new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(new Date(operation.updatedAt))}</p></div><InlineStatus tone={operationTone(operation)}>{t(`ai.operationState.${operation.state}`)}</InlineStatus></header>
      <div className={styles.metrics}>
        <Metric label={t("ai.phase")} value={t(`ai.phase.${operation.phase}`)} />
        <Metric label={t("ai.processed")} value={operation.totalItems === null ? amount(operation.completedItems) : `${amount(operation.completedItems)} / ${amount(operation.totalItems)}`} />
        <Metric label={t("ai.error")} value={operation.errorCode ? t(`ai.error.${operation.errorCode}`) : t("ai.none")} />
      </div>
      {percent !== null && <div className={styles.progress}><div aria-hidden="true"><span style={{ width: `${percent}%` }} /></div><span>{Math.round(percent)}%</span></div>}
      {operationIsActive(operation) && <div className={styles.actions}><Button disabled={cancelPending || operation.state === "cancelling"} onClick={() => onCancel(operation)}>{t("ai.cancelTask")}</Button></div>}
    </article>;
  })}</div>;
}

function ConfirmationDialog({ confirmation, onClose, onConfirm, pending }: { confirmation: Confirmation; onClose: () => void; onConfirm: () => void; pending: boolean }) {
  const { t } = useLocale();
  const kind = confirmation?.kind;
  return <Dialog
    actions={<><DialogCloseButton onClick={onClose} /><Button loading={pending} onClick={onConfirm} variant={kind === "clear" ? "danger" : "primary"}>{kind ? t(`ai.confirm.${kind}.action`) : t("common.cancel")}</Button></>}
    {...(kind ? { description: t(`ai.confirm.${kind}.description`) } : {})}
    onOpenChange={(open) => { if (!open) onClose(); }}
    open={confirmation !== null}
    title={kind ? t(`ai.confirm.${kind}.title`) : ""}
  ><p>{kind ? t(`ai.confirm.${kind}.safety`) : ""}</p></Dialog>;
}

function Metric({ label, value }: { label: string; value: string }) { return <div className={styles.metric}><span>{label}</span><strong>{value}</strong></div>; }
function coveragePercent(settings: SemanticSettingsSnapshot) { return settings.coverage.eligible === 0 ? 0 : Math.min(100, settings.coverage.completed / settings.coverage.eligible * 100); }
function parseView(value: string | null): AIView { return value === "libraries" || value === "tasks" || value === "faces" ? value : "overview"; }
function formatBytes(value: number, format: Intl.NumberFormat) { return value >= 1024 ** 3 ? `${format.format(value / 1024 ** 3)} GiB` : `${format.format(value / 1024 ** 2)} MiB`; }
function operationTone(operation: AIModelOperation): "info" | "success" | "warning" | "danger" { return operation.state === "succeeded" ? "success" : operation.state === "failed" ? "danger" : operation.state === "cancelled" ? "warning" : "info"; }
function formatMessage(message: string, values: Record<string, string>) { return Object.entries(values).reduce((result, [key, value]) => result.replaceAll(`{${key}}`, value), message); }
function actionErrorMessage(error: unknown, t: (key: MessageKey) => string) { if (error instanceof ApiError) { if (error.code === "model_download_active") return t("ai.download.active"); if (error.code === "model_unavailable") return t("ai.error.model_unavailable"); if (error.code === "insufficient_space") return t("ai.error.insufficient_space"); if (error.code === "precondition_failed") return t("ai.error.precondition_failed"); if (error.code === "model_source_unavailable") return t("ai.error.model_source_unavailable"); } return t("ai.actionFailed"); }
