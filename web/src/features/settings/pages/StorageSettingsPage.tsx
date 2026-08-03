import { Broom, FloppyDisk } from "@phosphor-icons/react";
import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import {
  Button,
  Dialog,
  DialogCloseButton,
  ErrorState,
  FormField,
  InlineStatus,
  LoadingState,
  Switch,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type { ResourceProfile } from "../../../lib/api/settings";
import { createRequestKey } from "../../../lib/requestKey";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import { useCacheSummaryQuery, useStartCacheCleanupMutation } from "../cache-queries";
import { useSettingsQuery, useUpdateSettingsMutation } from "../queries";
import styles from "./ManagementPage.module.css";

const bytesPerGiB = 1024 ** 3;

export function StorageSettingsPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const section = searchParams.get("section") === "cache" ? "cache" : "scan";
  const toast = useToast();
  const settingsQuery = useSettingsQuery();
  const cacheQuery = useCacheSummaryQuery();
  const { refetch: refreshSettings } = settingsQuery;
  const { refetch: refreshCache } = cacheQuery;
  const updateSettings = useUpdateSettingsMutation();
  const cleanup = useStartCacheCleanupMutation();
  const savePendingRef = useRef(false);
  const cleanupPendingRef = useRef(false);
  const [scheduleEnabled, setScheduleEnabled] = useState(false);
  const [interval, setInterval] = useState("24");
  const [quota, setQuota] = useState("10");
  const [resourceProfile, setResourceProfile] = useState<ResourceProfile>("balanced");
  const [confirmCleanup, setConfirmCleanup] = useState(false);

  useEffect(() => {
    if (!settingsQuery.data) return;
    setScheduleEnabled(settingsQuery.data.scheduledScanIntervalHours !== null);
    setInterval(String(settingsQuery.data.scheduledScanIntervalHours ?? 24));
    setQuota(String(Math.round(settingsQuery.data.thumbnailCacheQuotaBytes / bytesPerGiB)));
    setResourceProfile(settingsQuery.data.resourceProfile);
  }, [settingsQuery.data]);

  const numberFormat = useMemo(
    () => new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }),
    [locale],
  );
  const usagePercent = cacheQuery.data
    ? Math.min(100, (cacheQuery.data.usageBytes / cacheQuery.data.quotaBytes) * 100)
    : 0;

  function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!settingsQuery.data || savePendingRef.current) return;
    const parsedInterval = Number(interval);
    const parsedQuota = Number(quota);
    if (!Number.isInteger(parsedInterval) || parsedInterval < 1 || parsedInterval > 8760) return;
    if (!Number.isFinite(parsedQuota) || parsedQuota < 1 || parsedQuota > 1024) return;
    savePendingRef.current = true;
    void updateSettings
      .mutateAsync({
        csrfToken: session.csrfToken,
        etag: settingsQuery.data.etag,
        scheduledScanIntervalHours: scheduleEnabled ? parsedInterval : null,
        thumbnailCacheQuotaBytes: Math.round(parsedQuota * bytesPerGiB),
        resourceProfile,
      })
      .then(() =>
        toast.show({ message: t("settings.saved"), tone: "success" }),
      )
      .catch(() =>
        toast.show({ message: t("settings.saveFailed"), tone: "danger" }),
      )
      .finally(() => {
        savePendingRef.current = false;
      });
  }

  function runCleanup() {
    if (cleanupPendingRef.current) return;
    cleanupPendingRef.current = true;
    void cleanup
      .mutateAsync({
        csrfToken: session.csrfToken,
        idempotencyKey: createRequestKey(),
      })
      .then(() => {
        setConfirmCleanup(false);
        toast.show({ message: t("cache.cleanupStarted"), tone: "success" });
      })
      .catch(() =>
        toast.show({ message: t("settings.saveFailed"), tone: "danger" }),
      )
      .finally(() => {
        cleanupPendingRef.current = false;
      });
  }

  return (
    <ManagementShell
      active="general"
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
      <div className={styles.main}>
        <header className={styles.hero}>
          <p className={styles.eyebrow}>{t("management.title")}</p>
          <h1>{t("management.general")}</h1>
          <p>{t("configuration.description")}</p>
        </header>

        <div className={styles.tabs} role="tablist" aria-label={t("configuration.sections")}>
          <Button onClick={() => navigate(paths.generalSettings)} role="tab" variant="quiet">
            {t("configuration.generalTab")}
          </Button>
          <Button
            aria-selected={section === "scan"}
            onClick={() => navigate(`${paths.storageSettings}?section=scan`)}
            role="tab"
            variant={section === "scan" ? "secondary" : "quiet"}
          >
            {t("configuration.scanTab")}
          </Button>
          <Button
            aria-selected={section === "cache"}
            onClick={() => navigate(`${paths.storageSettings}?section=cache`)}
            role="tab"
            variant={section === "cache" ? "secondary" : "quiet"}
          >
            {t("configuration.cacheTab")}
          </Button>
        </div>

        {(settingsQuery.isPending || cacheQuery.isPending) && (
          <LoadingState label={t("settings.loading")} />
        )}
        {(settingsQuery.isError || cacheQuery.isError) && (
          <ErrorState
            message={t("cache.loadFailed")}
            onRetry={() => {
              void refreshSettings();
              void refreshCache();
            }}
          />
        )}
        {settingsQuery.data && cacheQuery.data && (
          <>
            {section === "cache" && <section className={styles.section}>
              <h2>{t("cache.overview")}</h2>
              <div className={styles.metricGrid}>
                <div className={styles.metric}>
                  <span>{t("cache.usage")}</span>
                  <strong>{formatBytes(cacheQuery.data.usageBytes, numberFormat)}</strong>
                </div>
                <div className={styles.metric}>
                  <span>{t("cache.quota")}</span>
                  <strong>{formatBytes(cacheQuery.data.quotaBytes, numberFormat)}</strong>
                </div>
                <div className={styles.metric}>
                  <span>{t("cache.available")}</span>
                  <strong>{formatBytes(cacheQuery.data.availableBytes, numberFormat)}</strong>
                </div>
              </div>
            </section>}

            <form onSubmit={saveSettings}>
              {section === "scan" && <section className={styles.section}>
                <h2>{t("cache.resourceProfile")}</h2>
                <div className={styles.card}>
                  <p className={styles.caption}>{t("cache.resourceProfileDescription")}</p>
                  <div className={styles.profileGrid}>
                    {(["eco", "balanced", "performance"] as const).map((profile) => (
                      <label className={styles.profileOption} key={profile}>
                        <input
                          checked={resourceProfile === profile}
                          name="resourceProfile"
                          onChange={() => setResourceProfile(profile)}
                          type="radio"
                          value={profile}
                        />
                        <span>
                          <strong>{t(`cache.resourceProfile.${profile}`)}</strong>
                          <span>{t(`cache.resourceProfile.${profile}Description`)}</span>
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              </section>}

              {section === "scan" && <section className={styles.section}>
                <h2>{t("cache.scanSchedule")}</h2>
                <div className={`${styles.card} ${styles.form}`}>
                  <label className={styles.row}>
                    <span>
                      <strong>{t("settings.scheduledScan")}</strong>
                      <span>{t("settings.scheduledScanDescription")}</span>
                    </span>
                    <Switch
                      checked={scheduleEnabled}
                      onChange={(event) => setScheduleEnabled(event.currentTarget.checked)}
                    />
                  </label>
                  <FormField
                    disabled={!scheduleEnabled}
                    label={t("settings.interval")}
                    max={8760}
                    min={1}
                    onChange={(event) => setInterval(event.currentTarget.value)}
                    type="number"
                    value={interval}
                  />
                </div>
                <div className={styles.sectionActions}>
                  <Button loading={updateSettings.isPending} type="submit" variant="primary">
                    <FloppyDisk aria-hidden="true" size={17} />
                    {t("settings.save")}
                  </Button>
                </div>
              </section>}

              {section === "cache" && <section className={styles.section}>
                <h2>{t("cache.thumbnailCache")}</h2>
                <div className={styles.card}>
                  <div className={styles.usageHeader}>
                    <div>
                      <strong>{t("cache.usage")}</strong>
                      <span>{t("settings.cacheSafety")}</span>
                    </div>
                    <strong>{numberFormat.format(usagePercent)}%</strong>
                  </div>
                  <div className={styles.track} aria-hidden="true">
                    <span style={{ width: `${usagePercent}%` }} />
                  </div>
                  {(cacheQuery.data.cleanup.status === "queued" ||
                    cacheQuery.data.cleanup.status === "running") && (
                    <div className={styles.status}>
                      <InlineStatus>{t("cache.cleanupRunning")}</InlineStatus>
                    </div>
                  )}
                  <div className={styles.row}>
                    <FormField
                      label={t("settings.cacheQuota")}
                      max={1024}
                      min={1}
                      onChange={(event) => setQuota(event.currentTarget.value)}
                      type="number"
                      value={quota}
                    />
                    <div>
                      <Button onClick={() => setConfirmCleanup(true)}>
                        <Broom aria-hidden="true" size={17} />
                        {t("cache.clear")}
                      </Button>
                      <Button loading={updateSettings.isPending} type="submit" variant="primary">
                        <FloppyDisk aria-hidden="true" size={17} />
                        {t("settings.save")}
                      </Button>
                    </div>
                  </div>
                </div>
              </section>}
            </form>
          </>
        )}
      </div>

      <Dialog
        actions={
          <>
            <DialogCloseButton onClick={() => setConfirmCleanup(false)} />
            <Button loading={cleanup.isPending} onClick={runCleanup} variant="danger">
              {t("cache.confirmClear")}
            </Button>
          </>
        }
        description={t("cache.clearDescription")}
        onOpenChange={setConfirmCleanup}
        open={confirmCleanup}
        title={t("cache.clearTitle")}
      >
        <p className={styles.caption}>{t("common.readOnlyFooter")}</p>
      </Dialog>
    </ManagementShell>
  );
}

function formatBytes(bytes: number, format: Intl.NumberFormat): string {
  if (bytes >= 1024 ** 3) return `${format.format(bytes / 1024 ** 3)} GiB`;
  if (bytes >= 1024 ** 2) return `${format.format(bytes / 1024 ** 2)} MiB`;
  return `${format.format(bytes / 1024)} KiB`;
}
