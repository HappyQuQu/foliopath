import {
  CaretDown,
  FolderOpen,
  GearSix,
  ImageSquare,
  List,
  MagnifyingGlass,
  SignOut,
  UserCircle,
  X,
} from "@phosphor-icons/react";
import {
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { NavLink } from "react-router-dom";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { IconButton } from "../../ui/Button/IconButton";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import { useToast } from "../../ui/Toast/ToastProvider";
import styles from "./AppShell.module.css";

export type AppSection = "browse" | "libraries" | "search" | "settings";

export function AppShell({
  active,
  browseHref,
  children,
  identity,
  librariesHref,
  logoutPending = false,
  onLogout,
  searchHref,
  settingsHref,
  showIdentityLabel = false,
  sidebarContent,
  topbarContent,
  title,
}: {
  active: AppSection;
  browseHref?: string;
  children: ReactNode;
  identity: string;
  librariesHref?: string;
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  searchHref?: string;
  settingsHref: string;
  showIdentityLabel?: boolean;
  sidebarContent?: ReactNode;
  topbarContent?: ReactNode;
  title: string;
}) {
  const { t } = useLocale();
  const toast = useToast();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false);
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
        {sidebarContent}
        <nav
          aria-label={t("shell.primaryNavigation")}
          className={`${styles.navigation} ${sidebarContent ? styles.navigationBottom : ""}`}
        >
          {active !== "browse" && browseHref ? (
            <NavLink
              onClick={() => setNavigationOpen(false)}
              to={browseHref}
            >
              <ImageSquare aria-hidden="true" size={20} />
              {t("shell.browse")}
            </NavLink>
          ) : active !== "browse" ? (
            <span aria-disabled="true" className={styles.disabledLink}>
              <ImageSquare aria-hidden="true" size={20} />
              {t("shell.browse")}
            </span>
          ) : null}
          {active !== "search" && searchHref ? (
            <NavLink
              onClick={() => setNavigationOpen(false)}
              to={searchHref}
            >
              <MagnifyingGlass aria-hidden="true" size={20} />
              {t("shell.search")}
            </NavLink>
          ) : active !== "search" ? (
            <span aria-disabled="true" className={styles.disabledLink}>
              <MagnifyingGlass aria-hidden="true" size={20} />
              {t("shell.search")}
            </span>
          ) : null}
          {active !== "libraries" && !sidebarContent && librariesHref && (
            <NavLink
              onClick={() => setNavigationOpen(false)}
              to={librariesHref}
            >
              <FolderOpen aria-hidden="true" size={20} />
              {t("shell.libraries")}
            </NavLink>
          )}
          {active !== "settings" && (
            <NavLink
              onClick={() => setNavigationOpen(false)}
              to={settingsHref}
            >
              <GearSix aria-hidden="true" size={20} />
              {t("shell.settings")}
            </NavLink>
          )}
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
                {showIdentityLabel && (
                  <span className={styles.accountName}>{identity}</span>
                )}
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
