import {
  CheckCircle,
  CircleNotch,
  FolderOpen,
  Plus,
  WarningCircle,
} from "@phosphor-icons/react";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import {
  Button,
  Dialog,
  ErrorState,
  FormField,
  InlineStatus,
  LoadingState,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { ApiError } from "../../../lib/api/errors";
import type { MediaJobProgress } from "../../../lib/api/media-processing";
import type {
  LibraryStatus,
  LibrarySummary,
} from "../../../lib/api/libraries";
import type { ScanStatus } from "../../../lib/api/scans";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { useSubmissionGuard } from "../../../lib/useSubmissionGuard";
import { paths } from "../../../routes/paths";
import { useRepairMediaProcessingMutation } from "../../diagnostics";
import {
  mediaProcessingIsActive,
  useMediaProcessingProgressQuery,
} from "../media-processing-queries";
import {
  isActiveScan,
  useCancelScanMutation,
  useRequestScanMutation,
  useScanQuery,
} from "../scan-queries";
import {
  libraryKeys,
  useLibrariesQuery,
  useLibraryRemovalQuery,
  useRemoveLibraryMutation,
  useRenameLibraryMutation,
} from "../queries";
import styles from "./LibrariesPage.module.css";
import {
  LibraryProcessingResults,
  LibraryScanRecords,
} from "./LibraryRecords";

type LibraryView = "libraries" | "scans" | "results";

const statusMessage: Record<LibraryStatus, MessageKey> = {
  pending: "libraries.statusPending",
  scanning: "libraries.statusScanning",
  ready: "libraries.statusReady",
  offline: "libraries.statusOffline",
  error: "libraries.statusError",
};

export function LibrariesPage({
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
  const view: LibraryView =
    searchParams.get("view") === "scans"
      ? "scans"
      : searchParams.get("view") === "results"
        ? "results"
        : "libraries";
  const query = useLibrariesQuery();
  const {
    fetchNextPage: loadNextPage,
    refetch: refreshLibraries,
  } = query;
  const libraries = useMemo(
    () => query.data?.pages.flatMap((page) => page.items) ?? [],
    [query.data],
  );
  return (
    <ManagementShell
      active="libraries"
      accountHref={paths.accountSettings}
      generalHref={paths.generalSettings}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      storageHref={paths.storageSettings}
    >
      {query.isPending && <LoadingLibraries />}
      {query.isError && (
        <LibrariesError onRetry={() => void refreshLibraries()} />
      )}
      {query.isSuccess && libraries.length === 0 && <LibraryWelcome />}
      {query.isSuccess && libraries.length > 0 && (
        <LibraryList
          fetchingMore={query.isFetchingNextPage}
          hasMore={query.hasNextPage}
          libraries={libraries}
          onLoadMore={() => void loadNextPage()}
          onViewChange={(nextView) => {
            const next = new URLSearchParams();
            if (nextView !== "libraries") next.set("view", nextView);
            setSearchParams(next, { replace: true });
          }}
          session={session}
          view={view}
        />
      )}
    </ManagementShell>
  );
}

function LoadingLibraries() {
  const { t } = useLocale();

  return (
    <div className={styles.statePage}>
      <LoadingState label={t("libraries.loading")} />
    </div>
  );
}

function LibrariesError({ onRetry }: { onRetry: () => void }) {
  const { t } = useLocale();

  return (
    <div className={styles.statePage}>
      <ErrorState message={t("libraries.loadFailed")} onRetry={onRetry} />
      <p className={styles.safetyNote}>{t("libraries.errorSafety")}</p>
    </div>
  );
}

function LibraryWelcome() {
  const { t } = useLocale();
  const navigate = useNavigate();
  const [helpOpen, setHelpOpen] = useState(false);

  return (
    <section className={styles.welcome} aria-labelledby="welcome-title">
      <div className={styles.welcomeIcon}>
        <FolderOpen aria-hidden="true" size={58} weight="duotone" />
      </div>
      <p className={styles.eyebrow}>{t("libraries.welcomeEyebrow")}</p>
      <h1 id="welcome-title">{t("libraries.emptyTitle")}</h1>
      <p>{t("libraries.emptyDescription")}</p>
      <div className={styles.welcomeActions}>
        <Button onClick={() => navigate(paths.newLibrary)} variant="primary">
          <Plus aria-hidden="true" size={18} />
          {t("libraries.create")}
        </Button>
        <Button
          aria-expanded={helpOpen}
          aria-controls="mount-help"
          onClick={() => setHelpOpen((open) => !open)}
        >
          {t("libraries.deploymentHelp")}
        </Button>
      </div>
      {helpOpen && (
        <div className={styles.help} id="mount-help">
          <strong>{t("libraries.mountTitle")}</strong>
          <code>/host/photos:/library:ro</code>
          <span>{t("libraries.mountDescription")}</span>
        </div>
      )}
    </section>
  );
}

function LibraryList({
  fetchingMore,
  hasMore,
  libraries,
  onLoadMore,
  onViewChange,
  session,
  view,
}: {
  fetchingMore: boolean;
  hasMore: boolean;
  libraries: LibrarySummary[];
  onLoadMore: () => void;
  onViewChange: (view: LibraryView) => void;
  session: AuthenticatedSession;
  view: LibraryView;
}) {
  const { locale, t } = useLocale();
  const navigate = useNavigate();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale),
    [locale],
  );
  const [searchParams] = useSearchParams();
  const [statusLibraryId, setStatusLibraryId] = useState<string | undefined>(
    searchParams.get("status") ?? undefined,
  );

  return (
    <section className={styles.page} aria-labelledby="libraries-title">
      <header className={styles.heading}>
        <div>
          <p className={styles.eyebrow}>{t("libraries.manageEyebrow")}</p>
          <h1 id="libraries-title">{t("libraries.title")}</h1>
          <p>{t("libraries.description")}</p>
        </div>
        {view === "libraries" && (
          <Button onClick={() => navigate(paths.newLibrary)} variant="primary">
            <Plus aria-hidden="true" size={18} />
            {t("libraries.create")}
          </Button>
        )}
      </header>
      <div className={styles.tabs} role="tablist" aria-label={t("libraryRecords.views")}>
        {(["libraries", "scans", "results"] as const).map((item) => (
          <Button
            aria-selected={view === item}
            key={item}
            onClick={() => onViewChange(item)}
            role="tab"
            variant={view === item ? "secondary" : "quiet"}
          >
            {t(`libraryRecords.${item}Tab` as MessageKey)}
          </Button>
        ))}
      </div>

      {view === "libraries" && (
        <>
          <div className={styles.list} role="tabpanel">
            {libraries.map((library) => (
              <LibraryRow
                expanded={statusLibraryId === library.id}
                key={library.id}
                library={library}
                numberFormatter={numberFormatter}
                onViewStatus={() =>
                  setStatusLibraryId((current) =>
                    current === library.id ? undefined : library.id,
                  )
                }
                session={session}
              />
            ))}
          </div>
          {hasMore && (
            <div className={styles.loadMore}>
              <Button loading={fetchingMore} onClick={onLoadMore}>
                {t("libraries.loadMore")}
              </Button>
            </div>
          )}
        </>
      )}
      {view === "scans" && <LibraryScanRecords libraries={libraries} />}
      {view === "results" && (
        <LibraryProcessingResults libraries={libraries} />
      )}
    </section>
  );
}

function LibraryRow({
  expanded,
  library,
  numberFormatter,
  onViewStatus,
  session,
}: {
  expanded: boolean;
  library: LibrarySummary;
  numberFormatter: Intl.NumberFormat;
  onViewStatus: () => void;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();
  const renameMutation = useRenameLibraryMutation();
  const removeMutation = useRemoveLibraryMutation();
  const [renameOpen, setRenameOpen] = useState(false);
  const [removeOpen, setRemoveOpen] = useState(false);
  const [nextName, setNextName] = useState(library.name);
  const [nameError, setNameError] = useState<string>();
  const [actionError, setActionError] = useState<string>();
  const [removalId, setRemovalId] = useState<string>();
  const removalQuery = useLibraryRemovalQuery(removalId);
  const removalKey = useRef(createRequestKey());
  const runSubmission = useSubmissionGuard();

  useEffect(() => {
    if (removalQuery.data?.status !== "succeeded") return;
    void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    toast.show({
      message: t("libraries.removeSucceeded").replace(
        "{name}",
        removalQuery.data.libraryName,
      ),
      tone: "success",
    });
    setRemovalId(undefined);
    setRemoveOpen(false);
  }, [queryClient, removalQuery.data, t, toast]);

  function openRename() {
    setNextName(library.name);
    setNameError(undefined);
    setActionError(undefined);
    setRenameOpen(true);
  }

  async function submitRename(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = nextName.trim();
    if (!normalized) {
      setNameError(t("libraries.nameRequired"));
      return;
    }
    if (normalized.length > 128) {
      setNameError(t("libraries.nameTooLong"));
      return;
    }

    await runSubmission(async () => {
      try {
        await renameMutation.mutateAsync({
          csrfToken: session.csrfToken,
          libraryId: library.id,
          name: normalized,
        });
        setRenameOpen(false);
        toast.show({ message: t("libraries.renameSucceeded"), tone: "success" });
      } catch (error) {
        setActionError(actionMessage(error, t));
      }
    });
  }

  async function submitRemoval() {
    setActionError(undefined);
    await runSubmission(async () => {
      try {
        const removal = await removeMutation.mutateAsync({
          csrfToken: session.csrfToken,
          idempotencyKey: removalKey.current,
          libraryId: library.id,
        });
        setRemovalId(removal.id);
      } catch (error) {
        setActionError(actionMessage(error, t));
      }
    });
  }

  const removalActive =
    removeMutation.isPending ||
    removalQuery.data?.status === "queued" ||
    removalQuery.data?.status === "running";
  const removalFailed = removalQuery.data?.status === "failed";

  const statusPanelId = `library-status-${library.id}`;

  return (
    <div className={styles.libraryGroup}>
      <article
        className={`${styles.library} ${expanded ? styles.libraryExpanded : ""}`}
      >
        <div className={styles.libraryIcon}>
          <FolderOpen aria-hidden="true" size={24} weight="duotone" />
        </div>
        <div className={styles.libraryDetails}>
          <h2>{library.name}</h2>
          <p>
            <span>{library.displayPath}</span>
            <span aria-hidden="true"> · </span>
            <span>
              {t("libraries.assetCount").replace(
                "{count}",
                numberFormatter.format(library.assetCount),
              )}
            </span>
          </p>
          <StatusPill status={library.status} />
        </div>
        <div className={styles.rowActions}>
          <Button
            onClick={() => navigate(paths.browse(library.id))}
            size="small"
            variant="primary"
          >
            {t("libraries.browse")}
          </Button>
          <Button
            aria-controls={statusPanelId}
            aria-expanded={expanded}
            disabled={!library.latestScanId}
            onClick={onViewStatus}
            size="small"
            title={
              library.latestScanId
                ? undefined
                : t("libraries.noScanAvailable")
            }
          >
            {expanded ? t("libraries.hideStatus") : t("libraries.viewStatus")}
          </Button>
          <Button onClick={openRename} size="small" variant="quiet">
            {t("libraries.rename")}
          </Button>
          <Button
            onClick={() => {
              removalKey.current = createRequestKey();
              setActionError(undefined);
              setRemoveOpen(true);
            }}
            size="small"
            variant="quiet"
          >
            {t("libraries.remove")}
          </Button>
        </div>
      </article>

      {expanded && (
        <LibraryInlineStatus
          id={statusPanelId}
          library={library}
          session={session}
        />
      )}

      <Dialog
        actions={
          <>
            <Button onClick={() => setRenameOpen(false)}>
              {t("newLibrary.cancel")}
            </Button>
            <Button
              form={`rename-${library.id}`}
              loading={renameMutation.isPending}
              type="submit"
              variant="primary"
            >
              {t("libraries.saveName")}
            </Button>
          </>
        }
        onOpenChange={setRenameOpen}
        open={renameOpen}
        title={t("libraries.renameTitle")}
      >
        <form id={`rename-${library.id}`} noValidate onSubmit={submitRename}>
          {actionError && <InlineStatus tone="danger">{actionError}</InlineStatus>}
          <FormField
            autoFocus
            error={nameError}
            label={t("libraries.newName")}
            maxLength={128}
            onChange={(event) => {
              setNextName(event.currentTarget.value);
              setNameError(undefined);
              setActionError(undefined);
            }}
            required
            value={nextName}
          />
        </form>
      </Dialog>

      <Dialog
        actions={
          <>
            <Button
              disabled={removalActive}
              onClick={() => setRemoveOpen(false)}
            >
              {t("newLibrary.cancel")}
            </Button>
            <Button
              loading={removalActive}
              onClick={() => void submitRemoval()}
              variant="danger"
            >
              {removalFailed
                ? t("libraries.retryRemoval")
                : t("libraries.confirmRemove")}
            </Button>
          </>
        }
        description={t("libraries.removeDescription").replace(
          "{name}",
          library.name,
        )}
        onOpenChange={(open) => {
          if (!removalActive) setRemoveOpen(open);
        }}
        open={removeOpen}
        title={t("libraries.removeTitle")}
      >
        {actionError && <InlineStatus tone="danger">{actionError}</InlineStatus>}
        {removalFailed && (
          <InlineStatus tone="danger">
            {t("libraries.removalFailed")}
          </InlineStatus>
        )}
        <ul className={styles.removalList}>
          <li>{t("libraries.removeConfiguration")}</li>
          <li>{t("libraries.removeIndex")}</li>
          <li>{t("libraries.removeJobs")}</li>
          <li>{t("libraries.removeCache")}</li>
        </ul>
        <InlineStatus tone="warning">
          {t("libraries.originalsSafe")}
        </InlineStatus>
      </Dialog>
    </div>
  );
}

function LibraryInlineStatus({
  id,
  library,
  session,
}: {
  id: string;
  library: LibrarySummary;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const toast = useToast();
  const scanQuery = useScanQuery(library.latestScanId ?? undefined);
  const scanActive = isActiveScan(scanQuery.data);
  const mediaQuery = useMediaProcessingProgressQuery(library.id, scanActive);
  const requestScanMutation = useRequestScanMutation();
  const cancelScanMutation = useCancelScanMutation();
  const repairMediaProcessingMutation = useRepairMediaProcessingMutation();
  const [rebuildOpen, setRebuildOpen] = useState(false);
  const runSubmission = useSubmissionGuard();
  const number = useMemo(() => new Intl.NumberFormat(locale), [locale]);
  const date = useMemo(
    () =>
      new Intl.DateTimeFormat(locale, {
        dateStyle: "medium",
        timeStyle: "short",
      }),
    [locale],
  );

  async function requestFullScan() {
    await runSubmission(async () => {
      try {
        await requestScanMutation.mutateAsync({
          csrfToken: session.csrfToken,
          libraryId: library.id,
        });
        toast.show({ message: t("scan.descriptionQueued"), tone: "success" });
      } catch {
        toast.show({ message: t("scan.actionFailed"), tone: "danger" });
      }
    });
  }

  async function cancelCurrentScan() {
    if (!scanQuery.data) return;
    await runSubmission(async () => {
      try {
        await cancelScanMutation.mutateAsync({
          csrfToken: session.csrfToken,
          scanId: scanQuery.data.id,
        });
      } catch {
        toast.show({ message: t("scan.cancelFailed"), tone: "danger" });
      }
    });
  }

  async function processMedia(mode: "missing" | "all") {
    await runSubmission(async () => {
      try {
        const summary = await repairMediaProcessingMutation.mutateAsync({
          csrfToken: session.csrfToken,
          libraryId: library.id,
          mode,
        });
        void mediaQuery.refetch();
        if (mode === "all") setRebuildOpen(false);
        toast.show({
          message: t("scan.mediaActionQueued").replace(
            "{count}",
            String(summary.requeued),
          ),
          tone: summary.requeued > 0 ? "success" : "neutral",
        });
      } catch {
        toast.show({ message: t("scan.mediaActionFailed"), tone: "danger" });
      }
    });
  }

  return (
    <section
      aria-label={t("libraries.statusPreviewTitle").replace("{name}", library.name)}
      className={styles.statusPreview}
      id={id}
    >
      <header className={styles.inlineStatusHeader}>
        <div>
          <h3>{t("libraries.statusPreviewTitle").replace("{name}", library.name)}</h3>
          <p>{t("libraries.statusPreviewDescription")}</p>
        </div>
      </header>
      <div className={styles.statusPreviewBody}>
        <div className={styles.previewOverview}>
          <StatusPill status={library.status} />
          <span>
            {t("libraries.assetCount").replace(
              "{count}",
              number.format(library.assetCount),
            )}
          </span>
        </div>

        {scanQuery.isPending && <LoadingState label={t("scan.loading")} />}
        {scanQuery.isError && (
          <ErrorState
            message={t("scan.loadFailed")}
            onRetry={() => void scanQuery.refetch()}
          />
        )}
        {scanQuery.data && (
          <section className={styles.previewSection} aria-labelledby="status-preview-scan">
            <div className={styles.previewHeading}>
              <h3 id="status-preview-scan">{t("libraries.latestScan")}</h3>
              <InlineStatus tone={scanTone(scanQuery.data.status)}>
                {t(scanStatusKey(scanQuery.data.status))}
              </InlineStatus>
            </div>
            <dl className={styles.previewCounters}>
              <div>
                <dt>{t("scan.assets")}</dt>
                <dd>{number.format(scanQuery.data.discoveredAssets)}</dd>
              </div>
              <div>
                <dt>{t("scan.directories")}</dt>
                <dd>{number.format(scanQuery.data.discoveredDirectories)}</dd>
              </div>
              <div>
                <dt>{t("scan.processed")}</dt>
                <dd>{number.format(scanQuery.data.processedAssets)}</dd>
              </div>
              <div>
                <dt>{t("scan.issues")}</dt>
                <dd>{number.format(scanQuery.data.errorCount)}</dd>
              </div>
            </dl>
            <p className={styles.previewTime}>
              {date.format(
                new Date(
                  scanQuery.data.finishedAt ??
                    scanQuery.data.startedAt ??
                    scanQuery.data.createdAt,
                ),
              )}
            </p>
          </section>
        )}

        <section className={styles.previewSection} aria-labelledby="status-preview-media">
          <div className={styles.previewHeading}>
            <h3 id="status-preview-media">{t("scan.mediaProcessing")}</h3>
            {mediaQuery.data && (
              <InlineStatus tone={mediaProcessingIsActive(mediaQuery.data) ? "info" : "success"}>
                {mediaProcessingIsActive(mediaQuery.data)
                  ? t("scan.mediaProcessingActive")
                  : t("scan.mediaProcessingComplete")}
              </InlineStatus>
            )}
          </div>
          {mediaQuery.isPending && (
            <LoadingState label={t("scan.mediaProcessingLoading")} />
          )}
          {mediaQuery.isError && (
            <ErrorState
              message={t("scan.mediaProcessingFailed")}
              onRetry={() => void mediaQuery.refetch()}
            />
          )}
          {mediaQuery.data && (
            <div className={styles.previewMediaList}>
              <StatusMediaRow
                label={t("scan.thumbnails")}
                number={number}
                progress={mediaQuery.data.thumbnails}
              />
              <StatusMediaRow
                label={t("scan.videoPreviews")}
                number={number}
                progress={mediaQuery.data.videoPreviews}
              />
            </div>
          )}
        </section>

        <footer className={styles.statusActions}>
          <p>{t("libraries.originalsSafe")}</p>
          <div>
            {scanActive ? (
              <Button
                disabled={
                  !scanQuery.data?.canCancel ||
                  Boolean(scanQuery.data.cancelRequestedAt)
                }
                loading={cancelScanMutation.isPending}
                onClick={() => void cancelCurrentScan()}
                variant="danger"
              >
                {scanQuery.data?.cancelRequestedAt
                  ? t("scan.cancelling")
                  : t("scan.cancel")}
              </Button>
            ) : (
              <>
                <Button
                  loading={requestScanMutation.isPending}
                  onClick={() => void requestFullScan()}
                  variant="quiet"
                >
                  {t("scan.rescanFiles")}
                </Button>
                <Button
                  disabled={!mediaQuery.data || repairMediaProcessingMutation.isPending}
                  loading={
                    repairMediaProcessingMutation.isPending &&
                    repairMediaProcessingMutation.variables?.mode === "missing"
                  }
                  onClick={() => void processMedia("missing")}
                  variant="secondary"
                >
                  {t("scan.fillMissing")}
                </Button>
                <Button
                  disabled={!mediaQuery.data || repairMediaProcessingMutation.isPending}
                  loading={
                    repairMediaProcessingMutation.isPending &&
                    repairMediaProcessingMutation.variables?.mode === "all"
                  }
                  onClick={() => setRebuildOpen(true)}
                  variant="primary"
                >
                  {t("scan.rebuildAll")}
                </Button>
              </>
            )}
          </div>
        </footer>
      </div>
      <Dialog
        actions={
          <>
            <Button onClick={() => setRebuildOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              loading={repairMediaProcessingMutation.isPending}
              onClick={() => void processMedia("all")}
              variant="primary"
            >
              {t("scan.rebuildAll")}
            </Button>
          </>
        }
        description={t("scan.rebuildAllDescription")}
        onOpenChange={(open) => {
          if (!repairMediaProcessingMutation.isPending) setRebuildOpen(open);
        }}
        open={rebuildOpen}
        title={t("scan.rebuildAllTitle")}
      >
        <InlineStatus tone="warning">
          {t("scan.rebuildAllWarning")}
        </InlineStatus>
      </Dialog>
    </section>
  );
}

function StatusMediaRow({
  label,
  number,
  progress,
}: {
  label: string;
  number: Intl.NumberFormat;
  progress: MediaJobProgress;
}) {
  const { t } = useLocale();
  const progressLabel = t("scan.mediaProcessingCount")
    .replace("{processed}", number.format(progress.processed))
    .replace("{total}", number.format(progress.total));
  return (
    <article className={styles.previewMediaRow}>
      <div>
        <strong>{label}</strong>
        <span>{progressLabel}</span>
      </div>
      <progress
        aria-label={`${label} · ${progressLabel}`}
        max={Math.max(1, progress.total)}
        value={progress.total === 0 ? 1 : progress.processed}
      />
      <p>
        {t("scan.mediaProcessingQueued").replace("{count}", number.format(progress.queued))}
        {" · "}
        {t("scan.mediaProcessingRunning").replace("{count}", number.format(progress.running))}
        {" · "}
        {t("scan.mediaProcessingFailures").replace("{count}", number.format(progress.failed))}
      </p>
    </article>
  );
}

function scanStatusKey(status: ScanStatus): MessageKey {
  return `scan.status${status.charAt(0).toUpperCase()}${status.slice(1)}` as MessageKey;
}

function scanTone(status: ScanStatus): "info" | "success" | "warning" {
  if (status === "succeeded") return "success";
  if (status === "queued" || status === "running") return "info";
  return "warning";
}

function StatusPill({ status }: { status: LibraryStatus }) {
  const { t } = useLocale();
  const Icon =
    status === "ready"
      ? CheckCircle
      : status === "offline" || status === "error"
        ? WarningCircle
        : CircleNotch;

  return (
    <span className={`${styles.status} ${styles[status]}`}>
      <Icon
        aria-hidden="true"
        className={status === "scanning" ? (styles.spinning ?? "") : ""}
        size={14}
        weight="bold"
      />
      {t(statusMessage[status])}
    </span>
  );
}

function actionMessage(
  error: unknown,
  t: (key: MessageKey) => string,
) {
  if (!(error instanceof ApiError)) return t("libraries.actionFailed");

  switch (error.code) {
    case "library_name_conflict":
      return t("newLibrary.nameConflict");
    case "precondition_failed":
      return t("libraries.changedElsewhere");
    case "scan_already_finished":
      return t("libraries.scanFinished");
    default:
      return t("libraries.actionFailed");
  }
}
