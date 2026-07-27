import {
  CheckCircle,
  CircleNotch,
  FolderOpen,
  Plus,
  WarningCircle,
} from "@phosphor-icons/react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import { Button, ErrorState, LoadingState } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type {
  LibraryStatus,
  LibrarySummary,
} from "../../../lib/api/libraries";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import { useLibrariesQuery } from "../queries";
import styles from "./LibrariesPage.module.css";

const statusMessage: Record<LibraryStatus, MessageKey> = {
  pending: "libraries.statusPending",
  scanning: "libraries.statusScanning",
  ready: "libraries.statusReady",
  offline: "libraries.statusOffline",
  error: "libraries.statusError",
};

export function LibrariesPage({ session }: { session: AuthenticatedSession }) {
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
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
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
}: {
  fetchingMore: boolean;
  hasMore: boolean;
  libraries: LibrarySummary[];
  onLoadMore: () => void;
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
          <article className={styles.library} key={library.id}>
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
            <Button
              disabled={!library.latestScanId}
              size="small"
              title={
                library.latestScanId
                  ? undefined
                  : t("libraries.noScanAvailable")
              }
            >
              {t("libraries.viewStatus")}
            </Button>
          </article>
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
        className={status === "scanning" ? styles.spinning : undefined}
        size={14}
        weight="bold"
      />
      {t(statusMessage[status])}
    </span>
  );
}
