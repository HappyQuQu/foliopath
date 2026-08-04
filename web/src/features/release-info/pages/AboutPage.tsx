import { ArrowClockwise, CheckCircle, DownloadSimple } from "@phosphor-icons/react";
import { useMemo, useState } from "react";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import { Button, ErrorState, InlineStatus, LoadingState } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { getReleaseInformation } from "../../../lib/api/release-info";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import { ReleaseNotes } from "../components/ReleaseNotes";
import { useApplicationStatusQuery, useReleaseInformationQuery } from "../queries";
import styles from "./AboutPage.module.css";

export function AboutPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const status = useApplicationStatusQuery();
  const releases = useReleaseInformationQuery();
  const { refetch: refreshStatus } = status;
  const { refetch: refreshReleases } = releases;
  const [checking, setChecking] = useState(false);
  const date = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: "medium" }),
    [locale],
  );

  async function refresh() {
    setChecking(true);
    try {
      await getReleaseInformation(true);
      await refreshReleases({ cancelRefetch: true });
    } catch {
      // The existing query state and user-facing degraded message remain safe.
    } finally {
      setChecking(false);
    }
  }

  return (
    <ManagementShell
      active="about"
      accountHref={paths.accountSettings}
      aboutHref={paths.aboutSettings}
      generalHref={paths.generalSettings}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logsHref={paths.logsSettings}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      storageHref={paths.storageSettings}
    >
      <div className={styles.main}>
        <header className={styles.hero}>
          <p>{t("management.title")}</p>
          <h1>{t("about.title")}</h1>
          <span>{t("about.description")}</span>
        </header>

        {status.isPending && <LoadingState label={t("about.loading")} />}
        {status.isError && (
          <ErrorState message={t("about.loadFailed")} onRetry={() => void refreshStatus()} />
        )}
        {status.data && (
          <section className={styles.versionCard}>
            <div>
              <span>{t("about.installedVersion")}</span>
              <strong>{status.data.version}</strong>
              <small>API {status.data.apiVersion}</small>
            </div>
            <div className={styles.updateState}>
              {releases.data?.updateAvailable ? (
                <InlineStatus tone="warning">
                  <DownloadSimple aria-hidden="true" size={16} />
                  {t("about.updateAvailable").replace(
                    "{version}",
                    releases.data.latestVersion ?? "",
                  )}
                </InlineStatus>
              ) : (
                <InlineStatus tone="success">
                  <CheckCircle aria-hidden="true" size={16} />
                  {t("about.upToDate")}
                </InlineStatus>
              )}
              <Button loading={checking} onClick={() => void refresh()} variant="secondary">
                <ArrowClockwise aria-hidden="true" size={17} />
                {t("about.checkUpdates")}
              </Button>
            </div>
          </section>
        )}

        <section className={styles.section}>
          <h2>{t("about.releaseHistory")}</h2>
          {releases.isPending && <LoadingState label={t("about.checking")} />}
          {releases.isError && (
            <ErrorState message={t("about.checkFailed")} onRetry={() => void refreshReleases()} />
          )}
          {releases.data && releases.data.releases.length === 0 && (
            <p className={styles.muted}>{t("about.noPublishedReleases")}</p>
          )}
          <div className={styles.releaseList}>
            {releases.data?.releases.map((release) => (
              <article className={styles.release} key={release.version}>
                <div>
                  <strong>{release.name}</strong>
                  <span>{release.version} · {date.format(new Date(release.publishedAt))}</span>
                </div>
                {release.notes ? (
                  <div className={styles.releaseNotes}>
                    <ReleaseNotes notes={release.notes} version={release.version} />
                  </div>
                ) : (
                  release.summary && <p>{release.summary}</p>
                )}
              </article>
            ))}
          </div>
        </section>
      </div>
    </ManagementShell>
  );
}
