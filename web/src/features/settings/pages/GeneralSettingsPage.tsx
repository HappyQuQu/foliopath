import { FloppyDisk, SignOut } from "@phosphor-icons/react";
import {
  useEffect,
  useState,
  type FormEvent,
} from "react";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import {
  Button,
  ErrorState,
  FormField,
  InlineStatus,
  LoadingState,
  LocaleSelect,
  ThemeToggle,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { ApiError } from "../../../lib/api/errors";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
import { useSubmissionGuard } from "../../../lib/useSubmissionGuard";
import { paths } from "../../../routes/paths";
import {
  useSettingsQuery,
  useUpdateSettingsMutation,
} from "../queries";
import styles from "./GeneralSettingsPage.module.css";

const bytesPerGiB = 1024 ** 3;

export function GeneralSettingsPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending: boolean;
  onLogout: () => Promise<void>;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const toast = useToast();
  const settingsQuery = useSettingsQuery();
  const { refetch: refreshSettings } = settingsQuery;
  const updateMutation = useUpdateSettingsMutation();
  const [scheduleEnabled, setScheduleEnabled] = useState(true);
  const [interval, setInterval] = useState("24");
  const [cacheQuota, setCacheQuota] = useState("10");
  const [formError, setFormError] = useState<string>();
  const runSubmission = useSubmissionGuard();

  useEffect(() => {
    if (!settingsQuery.data) return;
    setScheduleEnabled(settingsQuery.data.scheduledScanIntervalHours !== null);
    setInterval(String(settingsQuery.data.scheduledScanIntervalHours ?? 24));
    setCacheQuota(
      formatQuota(settingsQuery.data.thumbnailCacheQuotaBytes / bytesPerGiB),
    );
  }, [settingsQuery.data]);

  async function submitSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!settingsQuery.data) return;

    const parsedInterval = Number(interval);
    const parsedQuota = Number(cacheQuota);
    if (
      scheduleEnabled &&
      (!Number.isInteger(parsedInterval) ||
        parsedInterval < 1 ||
        parsedInterval > 8760)
    ) {
      setFormError(t("settings.intervalInvalid"));
      return;
    }
    if (!Number.isFinite(parsedQuota) || parsedQuota <= 0) {
      setFormError(t("settings.cacheInvalid"));
      return;
    }

    setFormError(undefined);
    await runSubmission(async () => {
      try {
        await updateMutation.mutateAsync({
          csrfToken: session.csrfToken,
          etag: settingsQuery.data.etag,
          scheduledScanIntervalHours: scheduleEnabled ? parsedInterval : null,
          thumbnailCacheQuotaBytes: Math.round(parsedQuota * bytesPerGiB),
        });
        toast.show({ message: t("settings.saved"), tone: "success" });
      } catch (error) {
        setFormError(settingsError(error, t));
      }
    });
  }

  async function handleLogout() {
    try {
      await onLogout();
    } catch {
      toast.show({ message: t("account.logoutFailed"), tone: "danger" });
    }
  }

  return (
    <AppShell
      active="settings"
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      searchHref={paths.search}
      settingsHref={paths.generalSettings}
      title={t("account.title")}
    >
      <div className={styles.main}>
        <header className={styles.heading}>
          <p>{t("account.eyebrow")}</p>
          <h1>{t("account.title")}</h1>
          <span>{t("account.description")}</span>
        </header>

        <section className={styles.panel} aria-labelledby="appearance-title">
          <h2 id="appearance-title">{t("account.appearanceLanguage")}</h2>
          <div className={styles.row}>
            <div>
              <strong>{t("account.theme")}</strong>
              <span>{t("account.themeDescription")}</span>
            </div>
            <ThemeToggle />
          </div>
          <div className={styles.row}>
            <div>
              <strong>{t("account.language")}</strong>
              <span>{t("account.languageDescription")}</span>
            </div>
            <LocaleSelect />
          </div>
        </section>

        <section className={styles.panel} aria-labelledby="scan-cache-title">
          <h2 id="scan-cache-title">{t("settings.scanCache")}</h2>
          {settingsQuery.isPending && (
            <LoadingState label={t("settings.loading")} />
          )}
          {settingsQuery.isError && (
            <ErrorState
              message={t("settings.loadFailed")}
              onRetry={() => void refreshSettings()}
            />
          )}
          {settingsQuery.isSuccess && (
            <form className={styles.form} noValidate onSubmit={submitSettings}>
              {formError && <InlineStatus tone="danger">{formError}</InlineStatus>}
              <label className={styles.checkRow}>
                <input
                  checked={scheduleEnabled}
                  onChange={(event) => {
                    setScheduleEnabled(event.currentTarget.checked);
                    setFormError(undefined);
                  }}
                  type="checkbox"
                />
                <span>
                  <strong>{t("settings.scheduledScan")}</strong>
                  <small>{t("settings.scheduledScanDescription")}</small>
                </span>
              </label>
              <FormField
                description={t("settings.intervalDescription")}
                disabled={!scheduleEnabled}
                label={t("settings.interval")}
                max={8760}
                min={1}
                onChange={(event) => {
                  setInterval(event.currentTarget.value);
                  setFormError(undefined);
                }}
                required={scheduleEnabled}
                step={1}
                type="number"
                value={interval}
              />
              <FormField
                description={t("settings.cacheDescription")}
                label={t("settings.cacheQuota")}
                min={0.01}
                onChange={(event) => {
                  setCacheQuota(event.currentTarget.value);
                  setFormError(undefined);
                }}
                required
                step={0.1}
                type="number"
                value={cacheQuota}
              />
              <InlineStatus tone="info">
                {t("settings.cacheSafety")}
              </InlineStatus>
              <div className={styles.save}>
                <Button
                  loading={updateMutation.isPending}
                  type="submit"
                  variant="primary"
                >
                  <FloppyDisk aria-hidden="true" size={17} />
                  {t("settings.save")}
                </Button>
              </div>
            </form>
          )}
        </section>

        <section className={styles.panel} aria-labelledby="account-title">
          <h2 id="account-title">{t("account.account")}</h2>
          <div className={styles.row}>
            <div>
              <strong>{session.administrator.displayName}</strong>
              <span>
                {t("account.username")}
                {session.administrator.username}
              </span>
            </div>
            <Button
              loading={logoutPending}
              onClick={() => void handleLogout()}
              variant="secondary"
            >
              <SignOut aria-hidden="true" size={17} />
              {t("account.logout")}
            </Button>
          </div>
        </section>
      </div>
    </AppShell>
  );
}

function settingsError(
  error: unknown,
  t: (key: MessageKey) => string,
) {
  if (!(error instanceof ApiError)) return t("settings.saveFailed");
  if (error.code === "precondition_failed") return t("settings.changedElsewhere");
  if (error.code === "settings_invalid") return t("settings.invalid");
  return t("settings.saveFailed");
}

function formatQuota(value: number) {
  return Number.isInteger(value) ? String(value) : value.toFixed(2);
}
