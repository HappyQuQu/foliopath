import {
  Database,
  FolderOpen,
  GearSix,
  UserCircle,
} from "@phosphor-icons/react";
import type { ComponentType, ReactNode } from "react";
import { NavLink } from "react-router-dom";

import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { AppShell } from "../AppShell/AppShell";
import styles from "./ManagementShell.module.css";

export type ManagementSection = "general" | "libraries" | "storage" | "account";

const navigation: Array<{
  hrefKey: "accountHref" | "generalHref" | "librariesHref" | "storageHref";
  icon: ComponentType<{ "aria-hidden": "true"; size: number }>;
  key: MessageKey;
  section: ManagementSection;
}> = [
  {
    hrefKey: "generalHref",
    icon: GearSix,
    key: "management.general",
    section: "general",
  },
  {
    hrefKey: "librariesHref",
    icon: FolderOpen,
    key: "shell.libraries",
    section: "libraries",
  },
  {
    hrefKey: "storageHref",
    icon: Database,
    key: "settings.scanCache",
    section: "storage",
  },
  {
    hrefKey: "accountHref",
    icon: UserCircle,
    key: "account.account",
    section: "account",
  },
];

export function ManagementShell({
  active,
  accountHref,
  children,
  generalHref,
  homeHref,
  identity,
  librariesHref,
  logoutPending,
  onLogout,
  searchHref,
  storageHref,
}: {
  active: ManagementSection;
  accountHref: string;
  children: ReactNode;
  generalHref: string;
  homeHref: string;
  identity: string;
  librariesHref: string;
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  searchHref: string;
  storageHref: string;
}) {
  const { t } = useLocale();
  const hrefs = { accountHref, generalHref, librariesHref, storageHref };

  return (
    <AppShell
      active="settings"
      homeHref={homeHref}
      identity={identity}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={searchHref}
      settingsHref={generalHref}
      title={t("management.title")}
    >
      <div className={styles.layout}>
        <aside className={styles.sidebar}>
          <h2>{t("management.title")}</h2>
          <nav aria-label={t("management.navigation")}>
            {navigation.map(({ hrefKey, icon: Icon, key, section }) => (
              <NavLink
                aria-current={section === active ? "page" : false}
                className={section === active ? (styles.active ?? "") : ""}
                key={section}
                to={hrefs[hrefKey]}
              >
                <span className={styles.icon}>
                  <Icon aria-hidden="true" size={18} />
                </span>
                {t(key)}
              </NavLink>
            ))}
          </nav>
        </aside>
        <div className={styles.pane}>{children}</div>
      </div>
    </AppShell>
  );
}
