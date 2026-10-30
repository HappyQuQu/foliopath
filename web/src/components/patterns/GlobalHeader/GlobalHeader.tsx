import {
  CaretDown,
  GearSix,
  MagnifyingGlass,
  SignOut,
  UsersThree,
} from "@phosphor-icons/react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { BrandMark } from "../../ui/BrandMark/BrandMark";
import { LocaleToggle } from "../../ui/LocaleToggle/LocaleToggle";
import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import { Menu } from "../../ui/Menu/Menu";
import { useToast } from "../../ui/Toast/ToastProvider";
import styles from "./GlobalHeader.module.css";
import { useHeaderAddon } from "./HeaderAddonContext";

export interface GlobalSearchInputLabels {
  label: string;
  placeholder: string;
}

export function GlobalHeader({
  homeHref,
  identity,
  logoutPending = false,
  onLogout,
  peopleHref,
  searchHref,
  searchInput,
  settingsHref,
}: {
  homeHref: string;
  identity: string;
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  peopleHref?: string | undefined;
  searchHref: string;
  searchInput?: GlobalSearchInputLabels | undefined;
  settingsHref: string;
}) {
  const { t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const toast = useToast();
  const headerAddon = useHeaderAddon();
  const [accountOpen, setAccountOpen] = useState(false);
  const [query, setQuery] = useState(
    () => new URLSearchParams(location.search).get("q") ?? "",
  );
  const triggerRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    setQuery(new URLSearchParams(location.search).get("q") ?? "");
  }, [location.search]);


  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = query.trim();
    if (!normalized) return;
    const target = new URL(searchHref, window.location.href);
    target.searchParams.set("q", normalized);
    navigate(`${target.pathname}${target.search}${target.hash}`);
  }

  return (
    <header className={styles.header} data-scroll-chrome>
      <Link aria-label="FolioPath" className={styles.brand} to={homeHref}>
        <BrandMark size="small" />
        <span className={styles.wordmark}>FolioPath</span>
      </Link>

      <form className={styles.search} onSubmit={submitSearch} role="search">
        <MagnifyingGlass aria-hidden="true" size={18} />
        <label className={styles.visuallyHidden} htmlFor="global-search">
          {searchInput?.label ?? t("search.inputLabel")}
        </label>
        <input
          autoComplete="off"
          id="global-search"
          onChange={(event) => setQuery(event.currentTarget.value)}
          placeholder={searchInput?.placeholder ?? t("search.globalPlaceholder")}
          type="search"
          value={query}
        />
        <button type="submit">{t("search.submit")}</button>
      </form>

      <div className={styles.headerActions}>
        {peopleHref && (
          <Link
            aria-current={location.pathname === "/intelligence" ? "page" : undefined}
            aria-label={t("shell.people")}
            className={styles.peopleLink}
            to={peopleHref}
          >
            <UsersThree aria-hidden="true" size={19} />
            <span>{t("shell.people")}</span>
          </Link>
        )}
        <LocaleToggle />
        <ThemeToggle />
        {headerAddon}
        <div className={styles.account}>
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
          {accountOpen && triggerRef.current && <Menu
            label={t("account.menu").replace("{name}", identity)}
            heading={<>{identity} · {t("account.administrator")}</>}
            anchor={triggerRef.current}
            align="end"
            returnFocusTo={triggerRef.current}
            onClose={() => setAccountOpen(false)}
            items={[
              { id: "settings", label: t("shell.settings"), href: settingsHref, icon: <GearSix aria-hidden="true" size={18} />, onSelect: () => navigate(settingsHref) },
              { id: "logout", label: t("account.logout"), icon: <SignOut aria-hidden="true" size={18} />, danger: true, disabled: !onLogout || logoutPending,
                onSelect: () => { if (onLogout) void onLogout().catch(() => toast.show({ message: t("account.logoutFailed"), tone: "danger" })); } },
            ]}
          />}
        </div>
      </div>
    </header>
  );
}
