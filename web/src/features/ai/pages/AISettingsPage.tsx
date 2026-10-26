import {
  ArrowsClockwise,
  ImageSquare,
  ListChecks,
  Trash,
  Users,
} from "@phosphor-icons/react";
import { useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";

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
  Select,
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
import { AIServiceSettings, AISettingsOverview } from "./AISettingsOverview";
import { OCRSettings } from "./OCRSettings";
import { AutomaticTagSettings } from "./AutomaticTagSettings";
import { VideoSemanticProgress } from "./VideoSemanticProgress";
import { AIProcessingRow } from "./AIProcessingRow";
import { FaceAnalysisStatus, FaceFailureDetails, FaceGenerationHint } from "./FaceFailureDetails";

type AIView = "overview" | "tasks" | "service";
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
  const viewButtons = useRef<Partial<Record<AIView, HTMLButtonElement | null>>>({});
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
  const selectedId = searchParams.get("library");
  const selectedIndex = selectedId === null ? 0 : libraries.findIndex(library => library.id === selectedId);
  const selectedLibrary = libraries[selectedIndex];
  const selectedSemantic = semanticQueries[selectedIndex];
  const selectedFaces = faceQueries[selectedIndex];
  const serviceReady = (id: string | null | undefined) => !modelsQuery.isError && !modelsQuery.isPending &&
    (modelsQuery.data?.items.some(model => model.id === id && model.active && model.state === "available" && model.storageMode === "service") ?? false);
  const semanticReady = serviceReady(modelsQuery.data?.activeModelId);
  const faceReady = serviceReady(modelsQuery.data?.activeFaceModelId);

  function navigateView(item: AIView) {
    // Child actions disappear on navigation; keep keyboard focus on a stable tab.
    viewButtons.current[item]?.focus();
    setConfirmation(null);
    setSearchParams(current => {
      const next = new URLSearchParams(current);
      if (item === "overview") next.delete("view"); else next.set("view", item);
      if (selectedLibrary) next.set("library", selectedLibrary.id);
      return next;
    });
  }

  function selectLibrary(id: string) {
    setConfirmation(null);
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

  function rememberOperation(operation: Pick<AIModelOperation, "id">) {
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
      <div className={`${styles.main} ${view === "tasks" ? styles.taskPage : ""}`}>
        <header className={styles.heading}>
          <p>{t("ai.eyebrow")}</p><h1>{t("ai.title")}</h1><span>{t("ai.settings.workspaceDescription")}</span>
        </header>
        <div className={styles.tabs} role="group" aria-label={t("ai.settings.navigation")}>
          {(["overview", "tasks", "service"] as const).map(item => <Button key={item} ref={node => { viewButtons.current[item] = node; }} variant={view === item ? "secondary" : "quiet"} aria-pressed={view === item} onClick={() => navigateView(item)}>{t(`ai.workspace.${item}`)}</Button>)}
        </div>

        {view !== "service" && <div className={`${styles.libraryScope} ${view === "tasks" ? styles.taskScope : ""}`}>
          {view === "tasks" ? <h2>{t("ai.tasks.library")}</h2> : <div><h2>{t("ai.settings.libraryFeatures")}</h2><p>{t("ai.settings.libraryScope")}</p></div>}
          {libraries.length > 0 && <label><span>{t("ai.settings.currentLibrary")}</span><Select aria-label={t("ai.settings.currentLibrary")} value={selectedLibrary?.id ?? ""} onChange={event => selectLibrary(event.currentTarget.value)}>
            {!selectedLibrary && <option value="" disabled>{t("ai.settings.chooseLibrary")}</option>}
            {libraries.map(library => <option key={library.id} value={library.id}>{library.name}</option>)}
          </Select></label>}
          {view === "tasks" && <p>{t("ai.tasks.scope")}</p>}
          {librariesQuery.hasNextPage && <Button loading={librariesQuery.isFetchingNextPage} onClick={() => void librariesQuery.fetchNextPage()}>{t("ai.moreLibraries")}</Button>}
        </div>}
        {view !== "service" && librariesQuery.isPending && <LoadingState label={t("ai.loadingLibraries")} />}
        {view !== "service" && librariesQuery.isError && <ErrorState message={t("ai.loadFailed")} onRetry={() => void librariesQuery.refetch()} />}
        {view !== "service" && !librariesQuery.isPending && !librariesQuery.isError && !selectedLibrary && <EmptyState title={t(selectedId === null ? "ai.noLibraries" : "ai.settings.libraryNotFound")} description={t(selectedId === null ? "ai.noLibrariesDescription" : "ai.settings.chooseLibraryDescription")} />}
        {modelsQuery.isError && <ErrorState message={t("ai.overview.loadFailed")} onRetry={() => void modelsQuery.refetch()} />}
        {modelsQuery.isPending && <LoadingState label={t("ai.loadingStatus")} />}
        <div id="ai-settings-panel" className={styles.panel}>
          {view === "overview" && selectedLibrary && !librariesQuery.isError && !librariesQuery.isPending && <AISettingsOverview
            library={selectedLibrary} semantic={selectedSemantic} faces={selectedFaces}
            semanticReady={semanticReady} faceReady={faceReady}
            pending={updateSettings.isPending || updateFaces.isPending || requestJob.isPending || requestFaces.isPending}
            onSemantic={toggleSemantic} onFace={toggleFace} onTasks={() => navigateView("tasks")}
          ><AutomaticTagSettings key={selectedLibrary.id} libraryId={selectedLibrary.id} csrfToken={session.csrfToken} onOperation={rememberOperation} /><OCRSettings libraryId={selectedLibrary.id} csrfToken={session.csrfToken} offline={selectedLibrary.status!=="ready"} onSettings={()=>navigateView("overview")} onTasks={()=>navigateView("tasks")} /></AISettingsOverview>}

          {view === "service" && !modelsQuery.isPending && !modelsQuery.isError && <AIServiceSettings
            models={modelsQuery.data?.items ?? []} activeModelId={modelsQuery.data?.activeModelId ?? null} activeFaceModelId={modelsQuery.data?.activeFaceModelId ?? null}>
          </AIServiceSettings>}

          {view === "tasks" && <>
            {selectedLibrary && !librariesQuery.isError && !librariesQuery.isPending && <div className={styles.jobGrid} key={selectedLibrary.id}>
              <div className={styles.jobColumns} aria-hidden="true"><span>{t("ai.tasks.item")}</span><span>{t("ai.tasks.status")}</span><span>{t("ai.tasks.actions")}</span></div>
              <LibrariesPanel
                libraries={[selectedLibrary]} librariesPending={false} librariesError={false}
                queries={selectedSemantic ? [selectedSemantic] : []}
                actionPending={updateSettings.isPending || requestJob.isPending || clear.isPending || requestVideo.isPending}
                onClear={(library, settings) => setConfirmation({ kind: "clear", library, settings })}
                onMissing={library => void runAction(() => requestJob.mutateAsync({
                  csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: library.id, mode: "missing",
                }), "ai.jobStarted")}
                onRebuild={(library, settings) => setConfirmation({ kind: "rebuild", library, settings })}
                onRetry={() => void librariesQuery.refetch()} videoReady={semanticReady}
              />
              <FaceSettingsPanel libraries={[selectedLibrary]} queries={selectedFaces ? [selectedFaces] : []}
                pending={updateFaces.isPending || requestFaces.isPending || clearFaceDerived.isPending || clearFaceManual.isPending}
                serviceReady={faceReady}
                onJob={(library, mode) => runAction(() => requestFaces.mutateAsync({
                  csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: library.id, mode,
                }), "ai.jobStarted")}
                onDerived={(library, settings) => runAction(() => clearFaceDerived.mutateAsync({
                  csrfToken: session.csrfToken, etag: settings.etag, idempotencyKey: createRequestKey(), libraryId: library.id,
                }), "ai.clearStarted")}
                onManual={(library, settings, counts) => runAction(() => clearFaceManual.mutateAsync({
                  ...counts, csrfToken: session.csrfToken, etag: settings.etag, idempotencyKey: createRequestKey(), libraryId: library.id,
                }), "ai.clearStarted")}
              />
              <VideoSemanticProgress library={selectedLibrary}
                disabled={!semanticReady || !selectedSemantic?.data?.enabled || selectedSemantic.isError || selectedSemantic.isPending || selectedLibrary.status === "offline" || selectedSemantic.data.state === "clearing" || requestVideo.isPending || clear.isPending}
                onJob={mode => {
                  if (mode === "all") { setConfirmation({ kind: "videoRebuild", library: selectedLibrary }); return; }
                  void runAction(() => requestVideo.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: selectedLibrary.id, mode }), "ai.jobStarted");
                }}
              />
              <OCRSettings mode="tasks" libraryId={selectedLibrary.id} csrfToken={session.csrfToken} offline={selectedLibrary.status!=="ready"} onSettings={()=>navigateView("overview")} onTasks={()=>navigateView("tasks")} />
              <AutomaticTagSettings mode="tasks" onSettings={() => navigateView("overview")} libraryId={selectedLibrary.id} csrfToken={session.csrfToken} onOperation={rememberOperation} />
            </div>}
            <TasksPanel cancelPending={cancel.isPending} operationIds={trackedOperationIds} removableIds={operationIds}
              onBack={() => navigateView("overview")}
              onForget={id => setOperationIds(current => {
                const next = current.filter(value => value !== id); writeAIOperationIds(next); return next;
              })}
              queries={operationQueries}
              onCancel={operation => void cancel.mutateAsync({
                csrfToken: session.csrfToken, etag: operation.etag, operationId: operation.id,
              }).catch(error => toast.show({ message: actionErrorMessage(error, t), tone: "danger" }))}
            />
          </>}
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
      const blocked = actionPending || !videoReady || !settings?.enabled || settings?.state === "clearing" || library.status === "offline" || query?.isError || query?.isPending;
      return <AIProcessingRow key={library.id} title={t("ai.tasks.images")} description={t("ai.settings.imageJobDescription")} icon={<ImageSquare aria-hidden="true" size={26} />}
        action={<Button variant="primary" disabled={blocked} onClick={() => onMissing(library)}>{t("ai.fillMissing")}</Button>}
        maintenance={settings && !query?.isError && !query?.isPending ? <>
          <p>{t("ai.tasks.maintenanceHint")}</p>
          <div className={styles.actions}>
            <Button disabled={blocked} onClick={() => onRebuild(library, settings)}><ArrowsClockwise aria-hidden="true" size={17} />{t("ai.rebuildAll")}</Button>
            <Button disabled={actionPending || settings.state === "clearing"} onClick={() => onClear(library, settings)} variant="danger"><Trash aria-hidden="true" size={17} />{t("ai.clearData")}</Button>
          </div>
        </> : undefined}>
        {query?.isPending && <LoadingState density="compact" label={t("ai.tasks.loadingStatus")} />}
        {query?.isError && <ErrorState density="compact" message={t("ai.tasks.statusFailed")} onRetry={() => void query.refetch()} />}
        {settings && !query?.isError && !query?.isPending && <>
          {settings.coverageRecorded === false ? <InlineStatus appearance="plain">{t("ai.tasks.unknown")}</InlineStatus> : <>
          <InlineStatus appearance="plain" tone={settings.state === "degraded" ? "warning" : settings.state === "ready" ? "success" : "info"}>{t(`ai.state.${settings.state}`)}</InlineStatus>
          <p>{number.format(settings.coverage.completed)} / {number.format(settings.coverage.eligible)} · {t("ai.failed")} {number.format(settings.coverage.failed)}</p>
          <progress aria-label={t("ai.tasks.images")} max={Math.max(1, settings.coverage.eligible)} value={settings.coverage.completed} />
          <p>{formatMessage(t("ai.coverageDetail"), {
              completed: number.format(settings.coverage.completed),
              eligible: number.format(settings.coverage.eligible),
              stale: number.format(settings.coverage.stale),
            })}</p>
          </>}
          {settings.state === "degraded" && <InlineStatus tone="warning">{t("ai.degradedNotice")}</InlineStatus>}
          {library.status === "offline" && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
        </>}
      </AIProcessingRow>;
    })}
  </div>;
}

function FaceSettingsPanel({ libraries, onDerived, onJob, onManual, pending, queries, serviceReady }: {
  libraries: LibrarySummary[];
  pending: boolean;
  queries: Array<{ data: FaceSettingsSnapshot | undefined; isError: boolean; isPending: boolean; refetch: () => unknown }>;
  serviceReady: boolean;
  onJob: (library: LibrarySummary, mode: "missing" | "all") => void;
  onDerived: (library: LibrarySummary, settings: FaceSettingsSnapshot) => void;
  onManual: (library: LibrarySummary, settings: FaceSettingsSnapshot, counts: { assignmentCount: number; constraintCount: number; personCount: number }) => void;
}) {
  const { t } = useLocale();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const [counts, setCounts] = useState({ assignmentCount: 0, constraintCount: 0, personCount: 0 });
  const [confirmation, setConfirmation] = useState<{ kind: "derived" | "manual" | "rebuild"; library: LibrarySummary; settings: FaceSettingsSnapshot } | null>(null);
  return <div className={styles.cardList}>
    {libraries.map((library, index) => {
      const query = queries[index]; const settings = query?.data;
      const jobActive = ["queued", "running", "cancelling"].includes(settings?.diagnostics?.latestTask?.state ?? "");
      const blocked = pending || jobActive || !serviceReady || !settings?.enabled || settings?.state === "clearing" || library.status !== "ready" || query?.isError || query?.isPending;
      const retryMissing = () => { headingRef.current?.focus(); onJob(library, "missing"); };
      return <AIProcessingRow headingRef={headingRef} key={library.id} title={t("ai.settings.people")} description={t("ai.settings.peopleJobDescription")} icon={<Users aria-hidden="true" size={26} />}
        action={<Button disabled={blocked} onClick={retryMissing}>{t("ai.fillMissing")}</Button>}
        details={settings && !query?.isError && !query?.isPending ? <FaceFailureDetails settings={settings} library={library} disabled={Boolean(blocked)} onRetry={retryMissing} /> : undefined}
        maintenance={settings && !query?.isError && !query?.isPending ? <>
          <p>{t("ai.tasks.maintenanceHint")}</p>
          <div className={styles.actions}>
            <Button disabled={blocked} onClick={() => setConfirmation({ kind: "rebuild", library, settings })}>{t("ai.rebuildAll")}</Button>
            <Button disabled={pending || settings.state === "clearing"} variant="danger" onClick={() => setConfirmation({ kind: "derived", library, settings })}>{t("ai.clearFaceDerived")}</Button>
            <Button disabled={pending || settings.state === "clearing"} variant="danger" onClick={() => { setCounts({ assignmentCount: 0, constraintCount: 0, personCount: 0 }); setConfirmation({ kind: "manual", library, settings }); }}>{t("ai.clearFaceManual")}</Button>
          </div>
          <p>{t("ai.faceManualWarning")}</p>
        </> : undefined}>
        {query?.isPending && <LoadingState density="compact" label={t("ai.tasks.loadingStatus")} />}
        {query?.isError && <ErrorState density="compact" message={t("ai.tasks.statusFailed")} onRetry={() => void query.refetch()} />}
        {settings && !query?.isError && !query?.isPending && <>
          <FaceAnalysisStatus settings={settings} library={library} plain />
          <p>{settings.generationCurrent === false && `${t("ai.faceDiagnostics.historicalProgress")} · `}{settings.coverage.completed} / {settings.coverage.eligible} · {t("ai.failed")} {settings.coverage.failed}</p>
          <progress aria-label={t("ai.settings.peopleProgress")} max={Math.max(1, settings.coverage.eligible)} value={settings.coverage.completed} />
          <FaceGenerationHint settings={settings} />
          {library.status === "offline" && <InlineStatus tone="warning">{t("ai.offlinePreserved")}</InlineStatus>}
          {!serviceReady && <p>{t("ai.settings.peopleUnavailable")}</p>}
        </>}
      </AIProcessingRow>;
    })}
    <Dialog fallbackFocusRef={headingRef} open={confirmation !== null} onOpenChange={(open) => { if (!open) setConfirmation(null); }}
      title={t(confirmation?.kind === "manual" ? "ai.clearFaceManual" : confirmation?.kind === "rebuild" ? "ai.confirm.faceRebuild.title" : "ai.clearFaceDerived")}
      description={confirmation?.library.name ?? ""}
      actions={<><DialogCloseButton onClick={() => setConfirmation(null)} /><Button variant="danger" disabled={pending || Object.values(counts).some((value) => !Number.isSafeInteger(value) || value < 0)} onClick={() => {
        if (!confirmation) return;
        if (confirmation.kind === "manual") onManual(confirmation.library, confirmation.settings, counts);
        else if (confirmation.kind === "rebuild") onJob(confirmation.library, "all");
        else onDerived(confirmation.library, confirmation.settings);
        setConfirmation(null);
      }}>{t(confirmation?.kind === "manual" ? "ai.clearFaceManual" : confirmation?.kind === "rebuild" ? "ai.confirm.faceRebuild.title" : "ai.clearFaceDerived")}</Button></>}>
      <p>{t(confirmation?.kind === "manual" ? "ai.faceManualWarning" : confirmation?.kind === "rebuild" ? "ai.confirm.faceRebuild.description" : "ai.faceDerivedWarning")}</p>
      {confirmation?.kind === "manual" && (["personCount", "assignmentCount", "constraintCount"] as const).map((key) => <label key={key}>{t(`ai.${key}`)}<Input min="0" step="1" type="number" value={counts[key]} onChange={(event) => setCounts((current) => ({ ...current, [key]: Number(event.target.value) }))} /></label>)}
    </Dialog>
  </div>;
}

function TasksPanel({ cancelPending, onBack, onCancel, onForget, operationIds, removableIds, queries }: {
  cancelPending: boolean;
  onBack: () => void;
  onCancel: (operation: AIModelOperation & { etag: string }) => void;
  operationIds: string[];
  removableIds: string[];
  onForget: (id: string) => void;
  queries: Array<{ data: (AIModelOperation & { etag: string }) | undefined; error?: unknown; isError: boolean; isPending: boolean; refetch: () => unknown }>;
}) {
  const { locale, t } = useLocale();
  const recordsHeading = useRef<HTMLHeadingElement>(null);
  const isMissing = (index: number) => {
    const query = queries[index];
    return query?.isError && query.error instanceof ApiError && query.error.status === 404 && query.error.code === "ai_operation_not_found";
  };
  const missingIds = operationIds.filter((_, index) => isMissing(index));
  const visibleRecords = operationIds.map((id, index) => ({ id, index })).filter(({ index }) => !isMissing(index));
  const operations = queries.flatMap((query) => query.data && !query.isError && !query.isPending ? [query.data] : []);
  const summary = {
    active: operations.filter(operationIsActive).length,
    completed: operations.filter((operation) => operation.state === "succeeded").length,
    failed: operations.filter((operation) => operation.state === "failed").length,
  };
  const number = useMemo(() => new Intl.NumberFormat(locale), [locale]);
  const dateTime = useMemo(() => new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }), [locale]);

  return <section className={styles.taskWorkspace} aria-label={t("ai.tasks.records")}>
    <header className={styles.recordsHeading}><h2 ref={recordsHeading} tabIndex={-1}>{t("ai.tasks.records")}</h2><p>{t("ai.tasks.recordScope")}</p></header>
    {operations.length > 0 && <section className={styles.taskSummary} aria-label={t("ai.tasks.summary")}>
      <TaskSummary label={t("ai.tasks.running")} value={number.format(summary.active)} />
      <TaskSummary label={t("ai.tasks.completed")} value={number.format(summary.completed)} />
      <TaskSummary label={t("ai.tasks.failed")} value={number.format(summary.failed)} />
    </section>}

    {visibleRecords.length === 0 ? <EmptyState density="compact" icon={ListChecks} title={t("ai.tasks.noAvailable")} description={t("ai.tasks.noAvailableDescription")} action={missingIds.length === 0 ? <Button variant="quiet" onClick={onBack}>{t("ai.tasks.emptyAction")}</Button> : undefined} /> : <section className={styles.taskTable} aria-label={t("ai.tasks.recent")}>
      <header className={styles.taskTableHeader}>
        <h2>{t("ai.tasks.recent")}</h2>
        <span>{formatMessage(t("ai.tasks.count"), { count: number.format(visibleRecords.length) })}</span>
      </header>
      <div className={styles.taskRows}>{visibleRecords.map(({ id, index }) => {
    const query = queries[index];
    const operation = query?.data;
    if (query?.isPending) return <div className={styles.taskStateRow} key={id}><LoadingState density="compact" label={t("ai.loadingTask")} /></div>;
    if (query?.isError) return <div className={styles.taskStateRow} key={id}><ErrorState density="compact" message={t("ai.taskFailedToLoad")} onRetry={() => void query.refetch()} /></div>;
    if (!operation) return null;
    const percent = operation.totalItems && operation.totalItems > 0 ? Math.min(100, operation.completedItems / operation.totalItems * 100) : null;
    const amount = (value: number) => operation.kind === "model_install" ? formatBytes(value, new Intl.NumberFormat(locale, { maximumFractionDigits: 1 })) : number.format(value);
    const processed = operation.totalItems === null ? amount(operation.completedItems) : `${amount(operation.completedItems)} / ${amount(operation.totalItems)}`;
    const translatedError = operation.errorCode ? t(`ai.error.${operation.errorCode}`) : "";
    const errorLabel = operation.errorCode
      ? translatedError || t("ai.tasks.noFailureReason")
      : operation.state === "failed" ? t("ai.tasks.noFailureReason") : t("ai.none");
    return <article className={styles.taskRow} key={id}>
      <div className={styles.taskIdentity}>
        <h2>{t(`ai.operation.${operation.kind}`)}</h2>
        <time dateTime={operation.updatedAt}>{formatMessage(t("ai.tasks.updated"), { time: dateTime.format(new Date(operation.updatedAt)) })}</time>
      </div>
      <div className={styles.taskStatus}><InlineStatus tone={operationTone(operation)}>{t(`ai.operationState.${operation.state}`)}</InlineStatus><span>{t(`ai.phase.${operation.phase}`)}</span></div>
      <div className={styles.taskProgress}>
        <div className={styles.taskProgressLabel}><span>{t("ai.tasks.progress")}</span><strong>{percent === null ? t("ai.tasks.pendingTotal") : `${Math.round(percent)}%`}</strong></div>
        <div aria-label={t("ai.tasks.progress")} aria-valuemax={100} aria-valuemin={0} {...(percent === null ? {} : { "aria-valuenow": Math.round(percent) })} role="progressbar"><span style={{ width: `${percent ?? 0}%` }} /></div>
      </div>
      <div className={styles.taskProcessed}><span>{t("ai.processed")}</span><strong>{processed}</strong></div>
      <div className={styles.taskOutcome}><span>{t("ai.tasks.taskError")}</span><strong className={operation.state === "failed" ? styles.taskOutcomeError : undefined}>{errorLabel}</strong></div>
      {operationIsActive(operation) ? <Button disabled={cancelPending || operation.state === "cancelling"} onClick={() => onCancel(operation)}>{t("ai.cancelTask")}</Button> : <span className={styles.taskActionPlaceholder} aria-hidden="true" />}
    </article>;
  })}</div></section>}
    {missingIds.length > 0 && <details className={styles.missingRecords}>
      <summary>{formatMessage(t("ai.tasks.unavailable"), { count: number.format(missingIds.length) })}</summary>
      <p>{t("ai.taskNotFoundDescription")}</p>
      {missingIds.map((id, index) => <div className={styles.taskStateRow} key={id}>
        <span>{t("ai.taskNotFound")} · {number.format(index + 1)}</span>
        {removableIds.includes(id) && <Button variant="quiet" onClick={() => { recordsHeading.current?.focus(); onForget(id); }}>{t("ai.forgetTask")}</Button>}
      </div>)}
    </details>}
  </section>;
}

function TaskSummary({ label, value }: { label: string; value: string }) {
  return <div className={styles.taskSummaryItem}><strong>{value}</strong><span>{label}</span></div>;
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
  ><p>{confirmation?.library.name}</p><p>{kind ? t(`ai.confirm.${kind}.safety`) : ""}</p></Dialog>;
}

function parseView(value: string | null): AIView { return value === "libraries" || value === "tasks" || value === "faces" ? "tasks" : value === "service" || value === "models" ? "service" : "overview"; }
function formatBytes(value: number, format: Intl.NumberFormat) { return value >= 1024 ** 3 ? `${format.format(value / 1024 ** 3)} GiB` : `${format.format(value / 1024 ** 2)} MiB`; }
function operationTone(operation: AIModelOperation): "info" | "success" | "warning" | "danger" { return operation.state === "succeeded" ? "success" : operation.state === "failed" ? "danger" : operation.state === "cancelled" ? "warning" : "info"; }
function formatMessage(message: string, values: Record<string, string>) { return Object.entries(values).reduce((result, [key, value]) => result.replaceAll(`{${key}}`, value), message); }
function actionErrorMessage(error: unknown, t: (key: MessageKey) => string) { if (error instanceof ApiError) { if (error.code === "model_download_active") return t("ai.download.active"); if (error.code === "model_unavailable") return t("ai.error.model_unavailable"); if (error.code === "insufficient_space") return t("ai.error.insufficient_space"); if (error.code === "precondition_failed") return t("ai.error.precondition_failed"); if (error.code === "model_source_unavailable") return t("ai.error.model_source_unavailable"); } return t("ai.actionFailed"); }
