import {
  CaretDown,
  GearSix,
  MagnifyingGlass,
  SignOut,
} from "@phosphor-icons/react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { BrandMark } from "../../ui/BrandMark/BrandMark";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import { useToast } from "../../ui/Toast/ToastProvider";
import styles from "./GlobalHeader.module.css";

export function GlobalHeader({
  homeHref,
  identity,
  logoutPending = false,
  onLogout,
  searchHref,
  settingsHref,
}: {
  homeHref: string;
  identity: string;
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  searchHref: string;
  settingsHref: string;
}) {
  const { t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const toast = useToast();
  const [accountOpen, setAccountOpen] = useState(false);
  const [query, setQuery] = useState(
    () => new URLSearchParams(location.search).get("q") ?? "",
  );
  const accountRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    setQuery(new URLSearchParams(location.search).get("q") ?? "");
  }, [location.search]);

  useEffect(() => {
    if (!accountOpen) return;

    function close(event: KeyboardEvent | PointerEvent) {
      if (event instanceof KeyboardEvent) {
        if (event.key !== "Escape") return;
        setAccountOpen(false);
        triggerRef.current?.focus();
        return;
      }
      if (
        event.target instanceof Node &&
        !accountRef.current?.contains(event.target)
      ) {
        setAccountOpen(false);
      }
    }

    document.addEventListener("keydown", close);
    document.addEventListener("pointerdown", close);
    return () => {
      document.removeEventListener("keydown", close);
      document.removeEventListener("pointerdown", close);
    };
  }, [accountOpen]);

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = query.trim();
    if (!normalized) return;
    navigate(`${searchHref}?${new URLSearchParams({ q: normalized })}`);
  }

  return (
    <header className={styles.header}>
      <Link aria-label="FolioPath" className={styles.brand} to={homeHref}>
        <BrandMark size="small" />
        <span className={styles.wordmark}>FolioPath</span>
      </Link>

      <form className={styles.search} onSubmit={submitSearch} role="search">
        <MagnifyingGlass aria-hidden="true" size={18} />
        <label className={styles.visuallyHidden} htmlFor="global-search">
          {t("search.inputLabel")}
        </label>
        <input
          autoComplete="off"
          id="global-search"
          onChange={(event) => setQuery(event.currentTarget.value)}
          placeholder={t("search.globalPlaceholder")}
          type="search"
          value={query}
        />
        <button type="submit">{t("search.submit")}</button>
      </form>

      <div className={styles.headerActions}>
        <ThemeToggle />
        <div className={styles.account} ref={accountRef}>
          <button
            aria-expanded={accountOpen}
            aria-haspopup="menu"
            aria-label={t("account.menu").replace("{name}", identity)}
            className={styles.trigger}
            onClick={() => setAccountOpen((open) => !open)}
            ref={triggerRef}
            type="button"
          >
            <span aria-hidden="true" className={styles.avatar}>
              {identity.trim().slice(0, 1) || "管"}
            </span>
            <span className={styles.identity}>{identity}</span>
            <CaretDown aria-hidden="true" size={14} />
          </button>
          {accountOpen && (
            <div className={styles.menu} role="menu">
              <div className={styles.menuHeader}>
                <strong>{identity}</strong>
                <span>{t("account.administrator")}</span>
              </div>
              <Link
                onClick={() => setAccountOpen(false)}
                role="menuitem"
                to={settingsHref}
              >
                <GearSix aria-hidden="true" size={18} />
                {t("shell.settings")}
              </Link>
              <button
                className={styles.logout}
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
  );
}
