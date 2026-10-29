import { ArrowClockwise, ArrowSquareOut, BookOpen, Brain, ChatCircleText, Code, FileText, FileZip, FolderOpen, Heart, Package, ShieldCheck } from "@phosphor-icons/react";
import { useMemo, useState } from "react";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import { BrandMark, Button, ErrorState, InlineStatus, LoadingState } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { getReleaseInformation } from "../../../lib/api/release-info";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import { ReleaseNotes } from "../components/ReleaseNotes";
import { useApplicationStatusQuery, useReleaseInformationQuery } from "../queries";
import styles from "./AboutPage.module.css";

const projectRepository = "https://github.com/HappyQuQu/foliopath";
const highlights = [
  { icon: FolderOpen, title: "about.foldersTitle", description: "about.foldersDescription" },
  { icon: FileZip, title: "about.archivesTitle", description: "about.archivesDescription" },
  { icon: Brain, title: "about.aiTitle", description: "about.aiDescription" },
  { icon: Heart, title: "about.curationTitle", description: "about.curationDescription" },
] as const;
const resources = [
  { icon: BookOpen, label: "about.docsLink", description: "about.docsDescription", href: `${projectRepository}/blob/main/README.md`, localized: true },
  { icon: Code, label: "about.sourceLink", description: "about.sourceDescription", href: projectRepository },
  { icon: ChatCircleText, label: "about.feedbackLink", description: "about.feedbackDescription", href: `${projectRepository}/issues` },
  { icon: Package, label: "about.imageLink", description: "about.imageDescription", href: "https://hub.docker.com/r/evanqu/foliopath" },
  { icon: FileText, label: "about.licenseLink", description: "about.licenseDescription", href: `${projectRepository}/blob/main/LICENSE` },
] as const;

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
  const [refreshFailed, setRefreshFailed] = useState(false);
  const date = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: "medium" }),
    [locale],
  );

  async function refresh() {
    setChecking(true);
    setRefreshFailed(false);
    try {
      await getReleaseInformation(true);
      await refreshReleases({ cancelRefetch: true });
    } catch {
      setRefreshFailed(true);
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

        <div className={styles.overview}>
          <section className={styles.projectCard} aria-labelledby="about-project-title">
            <div className={styles.projectHeading}>
              <BrandMark size="large" />
              <h2 id="about-project-title">{t("about.projectTitle")}</h2>
            </div>
            <p>{t("about.projectDescription")}</p>
            <div className={styles.privacy}>
              <ShieldCheck aria-hidden="true" size={21} />
              <span>{t("about.projectPrivacy")}</span>
            </div>
          </section>
          <section className={styles.versionCard} aria-label={t("about.installedVersion")}>
            {status.isPending && <LoadingState label={t("about.loading")} />}
            {status.isError && <ErrorState message={t("about.loadFailed")} onRetry={() => void refreshStatus()} />}
            {status.data && <>
              <div>
                <span>{t("about.installedVersion")}</span>
                <strong>{status.data.version}</strong>
              </div>
              <div className={styles.updateState}>
                {checking || releases.isPending ? (
                  <InlineStatus tone="info">{t("about.checking")}</InlineStatus>
                ) : refreshFailed || releases.isError ? (
                  <InlineStatus tone="warning">{t("about.checkFailed")}</InlineStatus>
                ) : releases.data?.updateAvailable ? (
                  <InlineStatus tone="warning">{t("about.updateAvailable").replace("{version}", releases.data.latestVersion ?? "")}</InlineStatus>
                ) : releases.data?.latestVersion ? (
                  <InlineStatus tone="success">{t("about.upToDate")}</InlineStatus>
                ) : (
                  <InlineStatus tone="info">{t("about.noPublishedReleases")}</InlineStatus>
                )}
                <Button loading={checking || releases.isFetching} onClick={() => void refresh()} variant="secondary">
                  <ArrowClockwise aria-hidden="true" size={17} />
                  {t("about.checkUpdates")}
                </Button>
              </div>
            </>}
          </section>
        </div>

        <section className={styles.section} aria-labelledby="about-features-title">
          <h2 id="about-features-title">{t("about.featuresTitle")}</h2>
          <div className={styles.highlights}>
            {highlights.map(({ icon: Icon, title, description }) => <article className={styles.highlight} key={title}>
              <Icon aria-hidden="true" size={25} />
              <h3>{t(title)}</h3>
              <p>{t(description)}</p>
            </article>)}
          </div>
        </section>

        <div className={styles.details}>
          <section className={styles.section} aria-labelledby="about-formats-title">
            <h2 id="about-formats-title">{t("about.formatsTitle")}</h2>
            <div className={styles.infoCard}>
              <dl className={styles.formats}>
                <div><dt>{t("about.imagesLabel")}</dt><dd>{t("about.imagesValue")}</dd></div>
                <div><dt>{t("about.videosLabel")}</dt><dd>{t("about.videosValue")}</dd></div>
                <div><dt>{t("about.archivesLabel")}</dt><dd>{t("about.archivesValue")}</dd></div>
                <div><dt>{t("about.usageLabel")}</dt><dd>{t("about.usageValue")}</dd></div>
              </dl>
              <p className={styles.muted}>{t("about.unsupported")}</p>
            </div>
          </section>
          <section className={styles.section} aria-labelledby="about-resources-title">
            <h2 id="about-resources-title">{t("about.resourcesTitle")}</h2>
            <ul className={styles.resourceList}>
              {resources.map(({ icon: Icon, label, description, href, ...resource }) => <li key={label}>
                <a href={"localized" in resource && resource.localized && locale === "zh-CN" ? `${projectRepository}/blob/main/README.zh-CN.md` : href} target="_blank" rel="noopener noreferrer">
                  <Icon aria-hidden="true" size={21} />
                  <span><strong>{t(label)}</strong><small>{t(description)}</small></span>
                  <ArrowSquareOut aria-hidden="true" size={17} />
                </a>
              </li>)}
            </ul>
          </section>
        </div>

        <section className={styles.section}>
          <h2>{t("about.releaseHistory")}</h2>
          {releases.isPending && <LoadingState label={t("about.checking")} />}
          {releases.isError && (
            <ErrorState message={t("about.checkFailed")} onRetry={() => void refresh()} />
          )}
          {releases.data && releases.data.releases.length === 0 && (
            <p className={styles.muted}>{t("about.noPublishedReleases")}</p>
          )}
          <div className={styles.releaseList}>
            {releases.data?.releases.map((release) => (
              <article className={styles.release} key={release.version}>
                <div className={styles.releaseHeader}>
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
