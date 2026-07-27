import type { ReactNode } from "react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import styles from "./PublicLayout.module.css";

export function PublicLayout({ children }: { children: ReactNode }) {
  const { t } = useLocale();

  return (
    <div className={styles.page}>
      <a className={styles.skipLink} href="#main">
        {t("common.skipToMain")}
      </a>
      <header className={styles.brand}>
        <strong>FolioPath</strong>
        <ThemeToggle />
      </header>
      <main className={styles.main} id="main" tabIndex={-1}>
        {children}
      </main>
      <footer>{t("common.readOnlyFooter")}</footer>
    </div>
  );
}
