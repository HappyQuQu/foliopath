import { List, X } from "@phosphor-icons/react";
import { useEffect, useRef, useState, type ReactNode } from "react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { IconButton } from "../../ui/Button/IconButton";
import { GlobalHeader } from "../GlobalHeader/GlobalHeader";
import styles from "./AppShell.module.css";

export type AppSection = "browse" | "libraries" | "search" | "settings";

export function AppShell({
  children,
  homeHref,
  identity,
  logoutPending = false,
  onLogout,
  searchHref,
  settingsHref,
  sidebarContent,
  topbarContent,
  title,
}: {
  active: AppSection;
  browseHref?: string;
  children: ReactNode;
  homeHref: string;
  identity: string;
  librariesHref?: string;
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  searchHref: string;
  settingsHref: string;
  showIdentityLabel?: boolean;
  sidebarContent?: ReactNode;
  topbarContent?: ReactNode;
  title: string;
}) {
  const { t } = useLocale();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!navigationOpen) return;
    closeButtonRef.current?.focus();

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setNavigationOpen(false);
        menuButtonRef.current?.focus();
      }
    }

    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [navigationOpen]);

  function closeNavigation() {
    setNavigationOpen(false);
    menuButtonRef.current?.focus();
  }

  return (
    <div className={styles.shell}>
      <a className={styles.skipLink} href="#main">
        {t("common.skipToMain")}
      </a>
      <GlobalHeader
        homeHref={homeHref}
        identity={identity}
        logoutPending={logoutPending}
        onLogout={onLogout}
        searchHref={searchHref}
        settingsHref={settingsHref}
      />

      {sidebarContent && (
        <>
          <button
            aria-label={t("shell.closeNavigation")}
            className={`${styles.scrim} ${navigationOpen ? styles.scrimOpen : ""}`}
            onClick={closeNavigation}
            tabIndex={navigationOpen ? 0 : -1}
            type="button"
          />
          <aside
            aria-label={t("shell.directorySidebar")}
            className={`${styles.sidebar} ${navigationOpen ? styles.sidebarOpen : ""}`}
            id="primary-navigation"
          >
            <div className={styles.drawerHeader}>
              <strong>{title}</strong>
              <IconButton
                ref={closeButtonRef}
                className={styles.closeButton}
                label={t("shell.closeNavigation")}
                onClick={closeNavigation}
              >
                <X aria-hidden="true" size={20} />
              </IconButton>
            </div>
            {sidebarContent}
          </aside>
        </>
      )}

      <div
        className={`${styles.workspace} ${
          sidebarContent ? "" : styles.workspaceWithoutSidebar
        }`}
      >
        <div className={styles.content}>
          {topbarContent && (
            <div className={styles.contextBar}>
              {sidebarContent && (
                <IconButton
                  ref={menuButtonRef}
                  className={styles.menuButton}
                  label={t("shell.openNavigation")}
                  onClick={() => setNavigationOpen(true)}
                  aria-controls="primary-navigation"
                  aria-expanded={navigationOpen}
                >
                  <List aria-hidden="true" size={21} />
                </IconButton>
              )}
              <div className={styles.contextBarContent}>{topbarContent}</div>
            </div>
          )}
          <main id="main" tabIndex={-1}>
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
