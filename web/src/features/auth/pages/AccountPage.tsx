import { SignOut } from "@phosphor-icons/react";
import { useNavigate } from "react-router-dom";

import { Button } from "../../../components/ui/Button/Button";
import { LocaleSelect } from "../../../components/ui/LocaleSelect/LocaleSelect";
import { ThemeToggle } from "../../../components/ui/ThemeToggle/ThemeToggle";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useToast } from "../../../components/ui/Toast/ToastProvider";
import { useLogoutMutation } from "../queries";
import styles from "./AccountPage.module.css";

export function AccountPage({ session }: { session: AuthenticatedSession }) {
  const { t } = useLocale();
  const navigate = useNavigate();
  const logoutMutation = useLogoutMutation();
  const toast = useToast();

  async function handleLogout() {
    try {
      await logoutMutation.mutateAsync(session.csrfToken);
      navigate("/login", { replace: true });
    } catch {
      toast.show({ message: t("account.logoutFailed"), tone: "danger" });
    }
  }

  return (
    <div className={styles.page}>
      <a className={styles.skipLink} href="#main">
        {t("common.skipToMain")}
      </a>
      <header className={styles.header}>
        <strong>FolioPath</strong>
        <div>
          <span className={styles.identity}>{session.administrator.displayName}</span>
          <ThemeToggle />
        </div>
      </header>
      <main className={styles.main} id="main" tabIndex={-1}>
        <header className={styles.heading}>
          <p>{t("account.eyebrow")}</p>
          <h1>{t("account.title")}</h1>
          <span className={styles.description}>{t("account.description")}</span>
        </header>

        <section className={styles.panel} aria-labelledby="appearance-title">
          <h2 id="appearance-title">{t("account.appearance")}</h2>
          <div className={styles.row}>
            <div>
              <strong>{t("account.theme")}</strong>
              <span className={styles.description}>
                {t("account.themeDescription")}
              </span>
            </div>
            <ThemeToggle />
          </div>
        </section>

        <section className={styles.panel} aria-labelledby="language-title">
          <h2 id="language-title">{t("account.language")}</h2>
          <div className={styles.row}>
            <span className={styles.description}>{t("account.languageDescription")}</span>
            <LocaleSelect />
          </div>
        </section>

        <section className={styles.panel} aria-labelledby="account-title">
          <h2 id="account-title">{t("account.account")}</h2>
          <div className={styles.row}>
            <div>
              <strong>{session.administrator.displayName}</strong>
              <span className={styles.description}>
                {t("account.username")}
                {session.administrator.username}
              </span>
            </div>
            <Button
              loading={logoutMutation.isPending}
              onClick={handleLogout}
              variant="secondary"
            >
              <SignOut aria-hidden="true" size={17} />
              {t("account.logout")}
            </Button>
          </div>
        </section>
      </main>
    </div>
  );
}
