import { FloppyDisk, Key, SignOut } from "@phosphor-icons/react";
import { useEffect, useState, type FormEvent } from "react";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import {
  Button,
  ErrorState,
  FormField,
  LoadingState,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { useSubmissionGuard } from "../../../lib/useSubmissionGuard";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import {
  useAccountQuery,
  useChangePasswordMutation,
  useUpdateAccountMutation,
} from "../account-queries";
import styles from "./ManagementPage.module.css";

export function AccountSettingsPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const toast = useToast();
  const accountQuery = useAccountQuery();
  const { refetch: refreshAccount } = accountQuery;
  const updateAccount = useUpdateAccountMutation();
  const changePassword = useChangePasswordMutation();
  const profileGuard = useSubmissionGuard();
  const passwordGuard = useSubmissionGuard();
  const [displayName, setDisplayName] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState<string>();

  useEffect(() => {
    if (accountQuery.data) setDisplayName(accountQuery.data.displayName);
  }, [accountQuery.data]);

  function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accountQuery.data || !displayName.trim()) return;
    void profileGuard(async () => {
      await updateAccount.mutateAsync({
        csrfToken: session.csrfToken,
        displayName: displayName.trim(),
        etag: accountQuery.data.etag,
      });
      toast.show({ message: t("account.profileSaved"), tone: "success" });
    }).catch(() =>
      toast.show({ message: t("settings.saveFailed"), tone: "danger" }),
    );
  }

  function savePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPasswordError(undefined);
    if (newPassword !== confirmPassword) {
      setPasswordError(t("account.passwordMismatch"));
      return;
    }
    if (newPassword.length < 8) {
      setPasswordError(t("account.passwordTooShort"));
      return;
    }
    if (!accountQuery.data) return;
    void passwordGuard(async () => {
      await changePassword.mutateAsync({
        csrfToken: session.csrfToken,
        currentPassword,
        etag: accountQuery.data.etag,
        newPassword,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.show({ message: t("account.passwordSaved"), tone: "success" });
    }).catch(() => setPasswordError(t("account.passwordFailed")));
  }

  return (
    <ManagementShell
      active="account"
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
          <h1>{t("account.account")}</h1>
          <p>{t("account.pageDescription")}</p>
        </header>

        {accountQuery.isPending && <LoadingState label={t("account.loading")} />}
        {accountQuery.isError && (
          <ErrorState
            message={t("account.loadFailed")}
            onRetry={() => void refreshAccount()}
          />
        )}
        {accountQuery.data && (
          <>
            <section className={styles.section}>
              <h2>{t("account.profile")}</h2>
              <form className={`${styles.card} ${styles.form}`} onSubmit={saveProfile}>
                <div className={styles.formGrid}>
                  <FormField
                    label={t("auth.displayName")}
                    maxLength={48}
                    onChange={(event) => setDisplayName(event.currentTarget.value)}
                    required
                    value={displayName}
                  />
                  <FormField
                    description={t("account.usernameImmutable")}
                    label={t("auth.username")}
                    readOnly
                    value={accountQuery.data.username}
                  />
                </div>
                <div className={styles.footer}>
                  <span className={styles.caption}>{t("account.profileCaption")}</span>
                  <Button
                    disabled={
                      !displayName.trim() ||
                      displayName.trim() === accountQuery.data.displayName
                    }
                    loading={updateAccount.isPending}
                    type="submit"
                    variant="primary"
                  >
                    <FloppyDisk aria-hidden="true" size={17} />
                    {t("account.saveProfile")}
                  </Button>
                </div>
              </form>
            </section>

            <section className={styles.section}>
              <h2>{t("account.changePassword")}</h2>
              <form className={`${styles.card} ${styles.form}`} onSubmit={savePassword}>
                <div className={`${styles.formGrid} ${styles.formGridOne}`}>
                  <FormField
                    autoComplete="current-password"
                    label={t("account.currentPassword")}
                    onChange={(event) => setCurrentPassword(event.currentTarget.value)}
                    required
                    type="password"
                    value={currentPassword}
                  />
                  <FormField
                    autoComplete="new-password"
                    description={t("account.passwordRule")}
                    label={t("account.newPassword")}
                    minLength={8}
                    onChange={(event) => setNewPassword(event.currentTarget.value)}
                    required
                    type="password"
                    value={newPassword}
                  />
                  <FormField
                    autoComplete="new-password"
                    error={passwordError}
                    label={t("account.confirmPassword")}
                    minLength={8}
                    onChange={(event) => setConfirmPassword(event.currentTarget.value)}
                    required
                    type="password"
                    value={confirmPassword}
                  />
                </div>
                <div className={styles.footer}>
                  <span className={styles.caption}>{t("account.passwordCaption")}</span>
                  <Button loading={changePassword.isPending} type="submit" variant="primary">
                    <Key aria-hidden="true" size={17} />
                    {t("account.updatePassword")}
                  </Button>
                </div>
              </form>
            </section>

            <section className={styles.section}>
              <h2>{t("account.currentSession")}</h2>
              <div className={`${styles.card} ${styles.row}`}>
                <div>
                  <strong>{t("account.thisBrowser")}</strong>
                  <span>{t("account.sessionActive")}</span>
                </div>
                <Button
                  loading={Boolean(logoutPending)}
                  onClick={() => void onLogout?.()}
                  variant="danger"
                >
                  <SignOut aria-hidden="true" size={17} />
                  {t("account.logout")}
                </Button>
              </div>
            </section>
          </>
        )}
      </div>
    </ManagementShell>
  );
}
