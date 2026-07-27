import {
  FolderOpen,
  GearSix,
  ImageSquare,
  List,
  MagnifyingGlass,
  X,
} from "@phosphor-icons/react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { NavLink } from "react-router-dom";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { IconButton } from "../../ui/Button/IconButton";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import styles from "./AppShell.module.css";

export type AppSection = "libraries" | "settings";

export function AppShell({
  active,
  children,
  identity,
  librariesHref,
  settingsHref,
  title,
}: {
  active: AppSection;
  children: ReactNode;
  identity: string;
  librariesHref: string;
  settingsHref: string;
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
      <button
        aria-label={t("shell.closeNavigation")}
        className={`${styles.scrim} ${navigationOpen ? styles.scrimOpen : ""}`}
        onClick={closeNavigation}
        tabIndex={navigationOpen ? 0 : -1}
        type="button"
      />
      <aside
        aria-label={t("shell.primaryNavigation")}
        className={`${styles.sidebar} ${navigationOpen ? styles.sidebarOpen : ""}`}
        id="primary-navigation"
      >
        <div className={styles.brandRow}>
          <strong className={styles.brand}>FolioPath</strong>
          <IconButton
            ref={closeButtonRef}
            className={styles.closeButton}
            label={t("shell.closeNavigation")}
            onClick={closeNavigation}
          >
            <X aria-hidden="true" size={20} />
          </IconButton>
        </div>
        <nav className={styles.navigation}>
          <span aria-disabled="true" className={styles.disabledLink}>
            <ImageSquare aria-hidden="true" size={20} />
            {t("shell.browse")}
          </span>
          <span aria-disabled="true" className={styles.disabledLink}>
            <MagnifyingGlass aria-hidden="true" size={20} />
            {t("shell.search")}
          </span>
          <NavLink
            className={active === "libraries" ? (styles.currentLink ?? "") : ""}
            onClick={() => setNavigationOpen(false)}
            to={librariesHref}
          >
            <FolderOpen aria-hidden="true" size={20} />
            {t("shell.libraries")}
          </NavLink>
          <NavLink
            className={active === "settings" ? (styles.currentLink ?? "") : ""}
            onClick={() => setNavigationOpen(false)}
            to={settingsHref}
          >
            <GearSix aria-hidden="true" size={20} />
            {t("shell.settings")}
          </NavLink>
        </nav>
        <p className={styles.safety}>{t("common.readOnlyFooter")}</p>
      </aside>
      <div className={styles.content}>
        <header className={styles.topbar}>
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
          <strong>{title}</strong>
          <div className={styles.identity}>
            <span>{identity}</span>
            <ThemeToggle />
          </div>
        </header>
        <main id="main" tabIndex={-1}>
          {children}
        </main>
      </div>
    </div>
  );
}
