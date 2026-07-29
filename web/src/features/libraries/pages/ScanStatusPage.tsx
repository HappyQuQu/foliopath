import {
  ArrowLeft,
  CheckCircle,
  CircleNotch,
  WarningCircle,
} from "@phosphor-icons/react";
import { useMemo } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import {
  Button,
  ErrorState,
  InlineStatus,
  LoadingState,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type { ScanRun, ScanStatus } from "../../../lib/api/scans";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
import { useSubmissionGuard } from "../../../lib/useSubmissionGuard";
import { paths } from "../../../routes/paths";
import {
  useCancelScanMutation,
  useRequestScanMutation,
  useScanQuery,
} from "../scan-queries";
import { useLibraryQuery } from "../queries";
import styles from "./ScanStatusPage.module.css";

const scanStatusMessage: Record<ScanStatus, MessageKey> = {
  queued: "scan.statusQueued",
  running: "scan.statusRunning",
  succeeded: "scan.statusSucceeded",
  failed: "scan.statusFailed",
  cancelled: "scan.statusCancelled",
  offline: "scan.statusOffline",
  interrupted: "scan.statusInterrupted",
};

export function ScanStatusPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending?: boolean;
  onLogout?: () => Promise<void>;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const navigate = useNavigate();
  const location = useLocation();
  const { libraryId = "" } = useParams();
  const libraryQuery = useLibraryQuery(libraryId);
  const { refetch: refreshLibrary } = libraryQuery;
  const stateScanId = readScanId(location.state);
  const scanId = stateScanId ?? libraryQuery.data?.library.latestScanId ?? undefined;
  const scanQuery = useScanQuery(scanId);
  const { refetch: refreshScan } = scanQuery;

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
      title={t("scan.pageTitle")}
    >
      <section className={styles.page} aria-labelledby="scan-title">
        <header className={styles.heading}>
          <div>
            <p>{t("scan.eyebrow")}</p>
            <h1 id="scan-title">{t("scan.pageTitle")}</h1>
          </div>
          <Button onClick={() => navigate(paths.libraries)} variant="quiet">
            <ArrowLeft aria-hidden="true" size={18} />
            {t("scan.back")}
          </Button>
        </header>

        {libraryQuery.isPending && <LoadingState label={t("scan.loading")} />}
        {libraryQuery.isError && (
          <ErrorState
            message={t("scan.libraryFailed")}
            onRetry={() => void refreshLibrary()}
          />
        )}
        {libraryQuery.isSuccess && !scanId && (
          <NoScan
            libraryId={libraryId}
            libraryName={libraryQuery.data.library.name}
            session={session}
          />
        )}
        {scanId && scanQuery.isPending && (
          <LoadingState label={t("scan.loading")} />
        )}
        {scanId && scanQuery.isError && (
          <ErrorState
            message={t("scan.loadFailed")}
            onRetry={() => void refreshScan()}
          />
        )}
        {libraryQuery.isSuccess && scanQuery.isSuccess && (
          <ScanDetails
            libraryName={libraryQuery.data.library.name}
            scan={scanQuery.data}
            session={session}
          />
        )}
      </section>
    </AppShell>
  );
}

function NoScan({
  libraryId,
  libraryName,
  session,
}: {
  libraryId: string;
  libraryName: string;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const toast = useToast();
  const requestMutation = useRequestScanMutation();
  const runSubmission = useSubmissionGuard();

  async function startScan() {
    await runSubmission(async () => {
      try {
        await requestMutation.mutateAsync({
          csrfToken: session.csrfToken,
          libraryId,
        });
      } catch {
        toast.show({ message: t("scan.actionFailed"), tone: "danger" });
      }
    });
  }

  return (
    <div className={styles.noScan}>
      <h2>{libraryName}</h2>
      <p>{t("scan.noHistory")}</p>
      <Button
        loading={requestMutation.isPending}
        onClick={() => void startScan()}
        variant="primary"
      >
        {t("scan.start")}
      </Button>
    </div>
  );
}

function ScanDetails({
  libraryName,
  scan,
  session,
}: {
  libraryName: string;
  scan: ScanRun;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const toast = useToast();
  const requestMutation = useRequestScanMutation();
  const cancelMutation = useCancelScanMutation();
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
  const active = scan.status === "queued" || scan.status === "running";
  const cancelling = Boolean(scan.cancelRequestedAt);
  const Icon =
    scan.status === "succeeded"
      ? CheckCircle
      : scan.status === "failed" ||
          scan.status === "offline" ||
          scan.status === "interrupted"
        ? WarningCircle
        : CircleNotch;

  async function requestScan() {
    await runSubmission(async () => {
      try {
        await requestMutation.mutateAsync({
          csrfToken: session.csrfToken,
          libraryId: scan.libraryId,
        });
      } catch {
        toast.show({ message: t("scan.actionFailed"), tone: "danger" });
      }
    });
  }

  async function cancelCurrentScan() {
    await runSubmission(async () => {
      try {
        await cancelMutation.mutateAsync({
          csrfToken: session.csrfToken,
          scanId: scan.id,
        });
      } catch {
        toast.show({ message: t("scan.cancelFailed"), tone: "danger" });
      }
    });
  }

  return (
    <>
      <article className={styles.summary}>
        <div className={`${styles.statusIcon} ${styles[scan.status]}`}>
          <Icon
            aria-hidden="true"
            className={active ? (styles.spinning ?? "") : ""}
            size={38}
            weight="duotone"
          />
        </div>
        <div>
          <p className={styles.statusLabel}>{t(scanStatusMessage[scan.status])}</p>
          <h2>
            {active
              ? t("scan.activeTitle").replace("{name}", libraryName)
              : t("scan.finishedTitle").replace("{name}", libraryName)}
          </h2>
          <p>{statusDescription(scan, t)}</p>
        </div>
      </article>

      {(scan.status === "offline" ||
        scan.status === "failed" ||
        scan.status === "interrupted" ||
        scan.status === "cancelled") && (
        <InlineStatus tone={scan.status === "cancelled" ? "info" : "warning"}>
          {t("scan.indexPreserved")}
        </InlineStatus>
      )}

      <section className={styles.progress} aria-labelledby="progress-title">
        <div className={styles.progressHeading}>
          <h2 id="progress-title">{t("scan.progress")}</h2>
          <span>{t(`scan.phase.${scan.phase}` as MessageKey)}</span>
        </div>
        <progress
          aria-label={t("scan.progress")}
          max={1}
          value={scan.status === "succeeded" ? 1 : scan.progressRatio ?? undefined}
        />
        {scan.progressRatio === null && active && (
          <p className={styles.progressNote}>{t("scan.progressUnknown")}</p>
        )}
        <dl className={styles.counters}>
          <div>
            <dt>{t("scan.directories")}</dt>
            <dd>{number.format(scan.discoveredDirectories)}</dd>
          </div>
          <div>
            <dt>{t("scan.assets")}</dt>
            <dd>{number.format(scan.discoveredAssets)}</dd>
          </div>
          <div>
            <dt>{t("scan.processed")}</dt>
            <dd>{number.format(scan.processedAssets)}</dd>
          </div>
          <div>
            <dt>{t("scan.issues")}</dt>
            <dd>{number.format(scan.errorCount)}</dd>
          </div>
        </dl>
        <dl className={styles.metadata}>
          <div>
            <dt>{t("scan.started")}</dt>
            <dd>{scan.startedAt ? date.format(new Date(scan.startedAt)) : t("scan.notStarted")}</dd>
          </div>
          <div>
            <dt>{t("scan.finished")}</dt>
            <dd>{scan.finishedAt ? date.format(new Date(scan.finishedAt)) : t("scan.notFinished")}</dd>
          </div>
        </dl>
      </section>

      {scan.issues.length > 0 && (
        <section className={styles.issues} aria-labelledby="issues-title">
          <h2 id="issues-title">{t("scan.issueSummary")}</h2>
          <ul>
            {scan.issues.map((issue) => (
              <li key={`${issue.code}-${issue.sampleRelativePath ?? ""}`}>
                <strong>{t(`scan.issue.${issue.code}` as MessageKey)}</strong>
                <span>
                  {t("scan.issueCount").replace(
                    "{count}",
                    number.format(issue.count),
                  )}
                  {issue.sampleRelativePath
                    ? ` · ${issue.sampleRelativePath}`
                    : ""}
                </span>
              </li>
            ))}
          </ul>
          {scan.issuesTruncated && <p>{t("scan.issuesTruncated")}</p>}
        </section>
      )}

      <div className={styles.actions}>
        {active ? (
          <Button
            disabled={!scan.canCancel || cancelling}
            loading={cancelMutation.isPending}
            onClick={() => void cancelCurrentScan()}
            variant="danger"
          >
            {cancelling ? t("scan.cancelling") : t("scan.cancel")}
          </Button>
        ) : (
          <Button
            loading={requestMutation.isPending}
            onClick={() => void requestScan()}
            variant="primary"
          >
            {t("libraries.rescan")}
          </Button>
        )}
      </div>
    </>
  );
}

function statusDescription(
  scan: ScanRun,
  t: (key: MessageKey) => string,
) {
  if (scan.cancelRequestedAt) return t("scan.descriptionCancelling");
  switch (scan.status) {
    case "queued":
      return t("scan.descriptionQueued");
    case "running":
      return t("scan.descriptionRunning");
    case "succeeded":
      return t("scan.descriptionSucceeded");
    case "cancelled":
      return t("scan.descriptionCancelled");
    case "offline":
      return t("scan.descriptionOffline");
    case "interrupted":
      return t("scan.descriptionInterrupted");
    default:
      return t("scan.descriptionFailed");
  }
}

function readScanId(state: unknown) {
  if (
    typeof state === "object" &&
    state !== null &&
    "scanId" in state &&
    typeof state.scanId === "string"
  ) {
    return state.scanId;
  }
  return undefined;
}
