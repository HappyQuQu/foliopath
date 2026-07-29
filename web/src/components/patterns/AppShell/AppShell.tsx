import {
  CaretDown,
  HouseLine,
  List,
  MagnifyingGlass,
  SignOut,
  SlidersHorizontal,
  UserCircle,
  X,
} from "@phosphor-icons/react";
import {
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  readSidebarWidthPreference,
  writeSidebarWidthPreference,
} from "../../../lib/storage/preferences";
import { BrandMark } from "../../ui/BrandMark/BrandMark";
import { IconButton } from "../../ui/Button/IconButton";
import { IconLink } from "../../ui/Button/IconLink";
import { PanelResizer } from "../../ui/PanelResizer/PanelResizer";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import { useToast } from "../../ui/Toast/ToastProvider";
import styles from "./AppShell.module.css";

export type AppSection = "browse" | "libraries" | "search" | "settings";

export function AppShell({
  active,
  browseHref,
  children,
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
  identity: string;
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  searchHref?: string;
  settingsHref: string;
  sidebarContent?: ReactNode;
  topbarContent?: ReactNode;
  title: string;
}) {
  const { t } = useLocale();
  const toast = useToast();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(readSidebarWidthPreference);
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window === "undefined" ? 1280 : window.innerWidth,
  );
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const accountRef = useRef<HTMLDivElement>(null);
  const accountTriggerRef = useRef<HTMLButtonElement>(null);

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

  useEffect(() => {
    if (!accountOpen) return;

    function closeAccountMenu(event: KeyboardEvent | PointerEvent) {
      if (event instanceof KeyboardEvent) {
        if (event.key !== "Escape") return;
        setAccountOpen(false);
        accountTriggerRef.current?.focus();
        return;
      }
      if (
        event.target instanceof Node &&
        !accountRef.current?.contains(event.target)
      ) {
        setAccountOpen(false);
      }
    }

    document.addEventListener("keydown", closeAccountMenu);
    document.addEventListener("pointerdown", closeAccountMenu);
    return () => {
      document.removeEventListener("keydown", closeAccountMenu);
      document.removeEventListener("pointerdown", closeAccountMenu);
    };
  }, [accountOpen]);

  const sidebarMinWidth = 224;
  const sidebarMaxWidth = Math.max(
    272,
    Math.min(420, Math.floor(viewportWidth * 0.36)),
  );

  useEffect(() => {
    const updateViewportWidth = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", updateViewportWidth);
    return () => window.removeEventListener("resize", updateViewportWidth);
  }, []);

  useEffect(() => {
    setSidebarWidth((currentWidth) =>
      Math.min(sidebarMaxWidth, Math.max(sidebarMinWidth, currentWidth)),
    );
  }, [sidebarMaxWidth]);

  function updateSidebarWidth(nextWidth: number) {
    setSidebarWidth(nextWidth);
    writeSidebarWidthPreference(nextWidth);
  }

  function closeNavigation() {
    setNavigationOpen(false);
    menuButtonRef.current?.focus();
  }

  return (
    <div
      className={`${styles.shell} ${sidebarContent ? "" : styles.shellWithoutSidebar}`}
      style={
        {
          "--app-sidebar-width": `${sidebarWidth}px`,
        } as CSSProperties
      }
    >
      <a className={styles.skipLink} href="#main">
        {t("common.skipToMain")}
      </a>
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
            id="directory-sidebar"
          >
            <PanelResizer
              ariaLabel={t("shell.resizeSidebar")}
              className={styles.sidebarResizer}
              growDirection="right"
              max={sidebarMaxWidth}
              min={sidebarMinWidth}
              onChange={updateSidebarWidth}
              value={sidebarWidth}
            />
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
          </aside>
        </>
      )}
      <div className={styles.content}>
        <header className={styles.topbar}>
          {sidebarContent && (
            <IconButton
              ref={menuButtonRef}
              className={styles.menuButton}
              label={t("shell.openNavigation")}
              onClick={() => setNavigationOpen(true)}
              aria-controls="directory-sidebar"
              aria-expanded={navigationOpen}
            >
              <List aria-hidden="true" size={21} />
            </IconButton>
          )}
          {topbarContent ? (
            <div className={styles.topbarContent}>{topbarContent}</div>
          ) : (
            <strong>{title}</strong>
          )}
          <div className={styles.identity}>
            {active !== "browse" && browseHref && (
              <IconLink label={t("shell.browse")} to={browseHref}>
                <HouseLine aria-hidden="true" size={20} />
              </IconLink>
            )}
            {active !== "search" && !topbarContent && searchHref && (
              <IconLink
                label={t("shell.search")}
                to={searchHref}
              >
                <MagnifyingGlass aria-hidden="true" size={20} />
              </IconLink>
            )}
            {active !== "settings" && (
              <IconLink label={t("shell.settings")} to={settingsHref}>
                <SlidersHorizontal aria-hidden="true" size={20} />
              </IconLink>
            )}
            <ThemeToggle />
            <div className={styles.account} ref={accountRef}>
              <button
                aria-expanded={accountOpen}
                aria-haspopup="menu"
                aria-label={t("account.menu").replace("{name}", identity)}
                className={styles.accountTrigger}
                onClick={() => setAccountOpen((open) => !open)}
                ref={accountTriggerRef}
                type="button"
              >
                <UserCircle aria-hidden="true" size={21} />
                <span className={styles.accountName}>{identity}</span>
                <CaretDown aria-hidden="true" size={14} />
              </button>
              {accountOpen && (
                <div
                  aria-label={t("account.menu").replace("{name}", identity)}
                  className={styles.accountMenu}
                  role="menu"
                >
                  <button
                    className={styles.logoutButton}
                    disabled={!onLogout || logoutPending}
                    onClick={() => {
                      if (!onLogout) return;
                      void onLogout()
                        .then(() => setAccountOpen(false))
                        .catch(() =>
                          toast.show({
                            message: t("account.logoutFailed"),
                            tone: "danger",
                          }),
                        );
                    }}
                    role="menuitem"
                    type="button"
                  >
                    <SignOut aria-hidden="true" size={18} />
                    {t("account.logout")}
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>
        <main id="main" tabIndex={-1}>
          {children}
        </main>
      </div>
    </div>
  );
}
