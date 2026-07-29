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
import { BrandMark } from "../../ui/BrandMark/BrandMark";
import { IconButton } from "../../ui/Button/IconButton";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import styles from "./AppShell.module.css";

export type AppSection = "browse" | "libraries" | "search" | "settings";

export function AppShell({
  active,
  browseHref,
  children,
  identity,
  librariesHref,
  searchHref,
  settingsHref,
  showIdentityLabel = true,
  sidebarContent,
  topbarContent,
  title,
}: {
  active: AppSection;
  browseHref?: string;
  children: ReactNode;
  identity: string;
  librariesHref: string;
  searchHref?: string;
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
          <div className={styles.brand}>
            <BrandMark size="small" />
            <strong>FolioPath</strong>
          </div>
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
        <nav
          aria-label={t("shell.primaryNavigation")}
          className={`${styles.navigation} ${sidebarContent ? styles.navigationBottom : ""}`}
        >
          {browseHref ? (
            <NavLink
              className={active === "browse" ? (styles.currentLink ?? "") : ""}
              onClick={() => setNavigationOpen(false)}
              to={browseHref}
            >
              <ImageSquare aria-hidden="true" size={20} />
              {t("shell.browse")}
            </NavLink>
          ) : (
            <span aria-disabled="true" className={styles.disabledLink}>
              <ImageSquare aria-hidden="true" size={20} />
              {t("shell.browse")}
            </span>
          )}
          {searchHref ? (
            <NavLink
              className={active === "search" ? (styles.currentLink ?? "") : ""}
              onClick={() => setNavigationOpen(false)}
              to={searchHref}
            >
              <MagnifyingGlass aria-hidden="true" size={20} />
              {t("shell.search")}
            </NavLink>
          ) : (
            <span aria-disabled="true" className={styles.disabledLink}>
              <MagnifyingGlass aria-hidden="true" size={20} />
              {t("shell.search")}
            </span>
          )}
          {!sidebarContent && (
            <NavLink
              className={active === "libraries" ? (styles.currentLink ?? "") : ""}
              onClick={() => setNavigationOpen(false)}
              to={librariesHref}
            >
              <FolderOpen aria-hidden="true" size={20} />
              {t("shell.libraries")}
            </NavLink>
          )}
          <NavLink
            className={active === "settings" ? (styles.currentLink ?? "") : ""}
            onClick={() => setNavigationOpen(false)}
            to={settingsHref}
          >
            <GearSix aria-hidden="true" size={20} />
            {t("shell.settings")}
          </NavLink>
        </nav>
        {!sidebarContent && <p className={styles.safety}>{t("common.readOnlyFooter")}</p>}
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
          {topbarContent ? (
            <div className={styles.topbarContent}>{topbarContent}</div>
          ) : (
            <strong>{title}</strong>
          )}
          <div className={styles.identity}>
            {showIdentityLabel && <span>{identity}</span>}
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
