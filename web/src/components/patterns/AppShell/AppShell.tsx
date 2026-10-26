import { List, X } from "@phosphor-icons/react";
import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { readSidebarWidthPreference, writeSidebarWidthPreference } from "../../../lib/storage/preferences";
import { PanelResizer } from "../../ui/PanelResizer/PanelResizer";
import { IconButton } from "../../ui/Button/IconButton";
import { GlobalHeader, type GlobalSearchInputLabels } from "../GlobalHeader/GlobalHeader";
import styles from "./AppShell.module.css";

export type AppSection = "browse" | "libraries" | "search" | "settings";

export function AppShell({
  children,
  homeHref,
  identity,
  logoutPending = false,
  navigationKey,
  onLogout,
  peopleHref,
  searchHref,
  searchInput,
  settingsHref,
  sidebarContent,
  sidebarLabel,
  navigationLabel,
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
  /** Selected workspace destination; omit transient sidebar search/pagination state. */
  navigationKey?: string;
  onLogout?: (() => Promise<void>) | undefined;
  peopleHref?: string | undefined;
  searchHref: string;
  searchInput?: GlobalSearchInputLabels | undefined;
  settingsHref: string;
  showIdentityLabel?: boolean;
  sidebarContent?: ReactNode;
  sidebarLabel?: string | undefined;
  navigationLabel?: string | undefined;
  topbarContent?: ReactNode;
  title: string;
}) {
  const { t } = useLocale();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(readSidebarWidthPreference);
  const [viewportWidth, setViewportWidth] = useState(() => window.innerWidth);
  const minSidebarWidth = 192;
  const maxSidebarWidth = Math.max(minSidebarWidth, Math.min(480, Math.floor(viewportWidth * 0.45)));
  const effectiveSidebarWidth = Math.max(minSidebarWidth, Math.min(sidebarWidth, maxSidebarWidth));

  useEffect(() => {
    const updateWidth = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", updateWidth);
    return () => window.removeEventListener("resize", updateWidth);
  }, []);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const previousNavigationKey = useRef(navigationKey);

  useEffect(() => {
    if (previousNavigationKey.current === navigationKey) return;
    previousNavigationKey.current = navigationKey;
    if (navigationOpen) {
      setNavigationOpen(false);
      menuButtonRef.current?.focus();
    }
  }, [navigationKey, navigationOpen]);

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
    <div className={styles.shell} style={{ "--size-sidebar": `${effectiveSidebarWidth}px` } as CSSProperties}>
      <a className={styles.skipLink} href="#main">
        {t("common.skipToMain")}
      </a>
      <GlobalHeader
        homeHref={homeHref}
        identity={identity}
        logoutPending={logoutPending}
        onLogout={onLogout}
        peopleHref={peopleHref}
        searchHref={searchHref}
        searchInput={searchInput}
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
            aria-label={sidebarLabel ?? t("shell.directorySidebar")}
            className={`${styles.sidebar} ${navigationOpen ? styles.sidebarOpen : ""}`}
            id="primary-navigation"
          >
            <div className={styles.drawerHeader}>
              <strong>{sidebarLabel ?? title}</strong>
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
          <PanelResizer ariaLabel={t("shell.resizeSidebar")} className={styles.sidebarResizer}
            growDirection="right" min={minSidebarWidth} max={maxSidebarWidth} value={effectiveSidebarWidth}
            onChange={width => { setSidebarWidth(width); writeSidebarWidthPreference(width); }} />
        </>
      )}

      <div
        className={`${styles.workspace} ${
          sidebarContent ? "" : styles.workspaceWithoutSidebar
        }`}
      >
        <div className={styles.content}>
          {(topbarContent || sidebarContent) && (
            <div className={styles.contextBar}>
              {sidebarContent && (
                <IconButton
                  ref={menuButtonRef}
                  className={styles.menuButton}
                  label={navigationLabel ?? t("shell.openNavigation")}
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
