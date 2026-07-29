import {
  CheckCircle,
  CircleNotch,
  FolderOpen,
  Plus,
  WarningCircle,
} from "@phosphor-icons/react";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
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
import type {
  LibraryStatus,
  LibrarySummary,
} from "../../../lib/api/libraries";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { useSubmissionGuard } from "../../../lib/useSubmissionGuard";
import { paths } from "../../../routes/paths";
import { useRequestScanMutation } from "../scan-queries";
import {
  libraryKeys,
  useLibrariesQuery,
  useLibraryRemovalQuery,
  useRemoveLibraryMutation,
  useRenameLibraryMutation,
} from "../queries";
import styles from "./LibrariesPage.module.css";

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
    <AppShell
      active="libraries"
      browseHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      settingsHref={paths.generalSettings}
      title={t("libraries.title")}
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
          session={session}
        />
      )}
    </AppShell>
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
  session,
}: {
  fetchingMore: boolean;
  hasMore: boolean;
  libraries: LibrarySummary[];
  onLoadMore: () => void;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const navigate = useNavigate();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale),
    [locale],
  );

  return (
    <section className={styles.page} aria-labelledby="libraries-title">
      <header className={styles.heading}>
        <div>
          <p className={styles.eyebrow}>{t("libraries.manageEyebrow")}</p>
          <h1 id="libraries-title">{t("libraries.title")}</h1>
          <p>{t("libraries.description")}</p>
        </div>
        <Button onClick={() => navigate(paths.newLibrary)} variant="primary">
          <Plus aria-hidden="true" size={18} />
          {t("libraries.create")}
        </Button>
      </header>
      <div className={styles.list}>
        {libraries.map((library) => (
          <LibraryRow
            key={library.id}
            library={library}
            numberFormatter={numberFormatter}
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
    </section>
  );
}

function LibraryRow({
  library,
  numberFormatter,
  session,
}: {
  library: LibrarySummary;
  numberFormatter: Intl.NumberFormat;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();
  const renameMutation = useRenameLibraryMutation();
  const removeMutation = useRemoveLibraryMutation();
  const scanMutation = useRequestScanMutation();
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

  async function requestScan() {
    setActionError(undefined);
    await runSubmission(async () => {
      try {
        const scan = await scanMutation.mutateAsync({
          csrfToken: session.csrfToken,
          libraryId: library.id,
        });
        navigate(paths.libraryStatus(library.id), {
          state: { scanId: scan.id },
        });
      } catch (error) {
        toast.show({ message: actionMessage(error, t), tone: "danger" });
      }
    });
  }

  const removalActive =
    removeMutation.isPending ||
    removalQuery.data?.status === "queued" ||
    removalQuery.data?.status === "running";
  const removalFailed = removalQuery.data?.status === "failed";

  return (
    <>
      <article className={styles.library}>
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
            disabled={!library.latestScanId}
            onClick={() => navigate(paths.libraryStatus(library.id))}
            size="small"
            title={
              library.latestScanId
                ? undefined
                : t("libraries.noScanAvailable")
            }
          >
            {t("libraries.viewStatus")}
          </Button>
          <Button
            loading={scanMutation.isPending}
            onClick={() => void requestScan()}
            size="small"
          >
            {library.status === "offline"
              ? t("libraries.retryOffline")
              : t("libraries.rescan")}
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
    </>
  );
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
