import { useEffect, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import { Button, Switch } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  readMediaLayoutPreference,
  readMediaSortPreference,
  readPreviewAutoplayPreference,
  readPreviewPinnedPreference,
  writeMediaLayoutPreference,
  writeMediaSortPreference,
  writePreviewAutoplayPreference,
  writePreviewPinnedPreference,
  type MediaLayoutPreference,
  type MediaSortPreference,
} from "../../../lib/storage/preferences";
import { useTheme } from "../../../lib/theme/ThemeProvider";
import { paths } from "../../../routes/paths";
import styles from "./GeneralSettingsPage.module.css";

export function GeneralSettingsPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending: boolean;
  onLogout: () => Promise<void>;
  session: AuthenticatedSession;
}) {
  const { localePreference, setLocalePreference, t } = useLocale();
  const navigate = useNavigate();
  const { preference, setPreference } = useTheme();
  const [draftTheme, setDraftTheme] = useState(preference);
  const [draftLocale, setDraftLocale] = useState(localePreference);
  const [layout, setLayout] = useState<MediaLayoutPreference>(
    readMediaLayoutPreference,
  );
  const [mediaSort, setMediaSort] = useState<MediaSortPreference>(
    readMediaSortPreference,
  );
  const [previewPinned, setPreviewPinned] = useState(
    readPreviewPinnedPreference,
  );
  const [previewAutoplay, setPreviewAutoplay] = useState(
    readPreviewAutoplayPreference,
  );
  const [savedLayout, setSavedLayout] = useState(layout);
  const [savedMediaSort, setSavedMediaSort] = useState(mediaSort);
  const [savedPreviewPinned, setSavedPreviewPinned] = useState(previewPinned);
  const [savedPreviewAutoplay, setSavedPreviewAutoplay] = useState(
    previewAutoplay,
  );

  useEffect(() => {
    setDraftLocale(localePreference);
  }, [localePreference]);

  const dirty =
    draftTheme !== preference ||
    draftLocale !== localePreference ||
    layout !== savedLayout ||
    mediaSort !== savedMediaSort ||
    previewAutoplay !== savedPreviewAutoplay ||
    previewPinned !== savedPreviewPinned;

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPreference(draftTheme);
    setLocalePreference(draftLocale);
    writeMediaLayoutPreference(layout);
    writeMediaSortPreference(mediaSort);
    writePreviewAutoplayPreference(previewAutoplay);
    writePreviewPinnedPreference(previewPinned);
    setSavedLayout(layout);
    setSavedMediaSort(mediaSort);
    setSavedPreviewAutoplay(previewAutoplay);
    setSavedPreviewPinned(previewPinned);
  }

  function reset() {
    setDraftTheme(preference);
    setDraftLocale(localePreference);
    setLayout(savedLayout);
    setMediaSort(savedMediaSort);
    setPreviewAutoplay(savedPreviewAutoplay);
    setPreviewPinned(savedPreviewPinned);
  }

  return (
    <ManagementShell
      active="general"
      accountHref={paths.accountSettings}
      generalHref={paths.generalSettings}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      storageHref={paths.storageSettings}
    >
      <form className={styles.main} onSubmit={save}>
        <header className={styles.heading}>
          <p>{t("management.title")}</p>
          <h1>{t("management.general")}</h1>
          <span>{t("general.description")}</span>
        </header>

        <div className={styles.tabs} role="tablist" aria-label={t("configuration.sections")}>
          <Button aria-selected="true" role="tab" variant="secondary">
            {t("configuration.generalTab")}
          </Button>
          <Button onClick={() => navigate(`${paths.storageSettings}?section=scan`)} role="tab" variant="quiet">
            {t("configuration.scanTab")}
          </Button>
          <Button onClick={() => navigate(`${paths.storageSettings}?section=cache`)} role="tab" variant="quiet">
            {t("configuration.cacheTab")}
          </Button>
        </div>

        <section aria-labelledby="appearance-title">
          <h2 className={styles.sectionTitle} id="appearance-title">
            {t("account.appearanceLanguage")}
          </h2>
          <div className={styles.group}>
            <label className={styles.row}>
              <span>
                <strong>{t("account.theme")}</strong>
                <small>{t("general.themeDescription")}</small>
              </span>
              <select
                onChange={(event) =>
                  setDraftTheme(
                    event.currentTarget.value as "system" | "light" | "dark",
                  )
                }
                value={draftTheme}
              >
                <option value="system">{t("general.themeSystem")}</option>
                <option value="light">{t("general.themeLight")}</option>
                <option value="dark">{t("general.themeDark")}</option>
              </select>
            </label>
            <label className={styles.row}>
              <span>
                <strong>{t("account.language")}</strong>
                <small>{t("account.languageDescription")}</small>
              </span>
              <select
                onChange={(event) =>
                  setDraftLocale(
                    event.currentTarget.value === "browser"
                      ? "browser"
                      : event.currentTarget.value === "en"
                        ? "en"
                        : "zh-CN",
                  )
                }
                value={draftLocale}
              >
                <option value="browser">{t("general.languageBrowser")}</option>
                <option value="zh-CN">简体中文</option>
                <option value="en">English</option>
              </select>
            </label>
          </div>
        </section>

        <section aria-labelledby="browse-title">
          <h2 className={styles.sectionTitle} id="browse-title">
            {t("general.browsePreferences")}
          </h2>
          <div className={styles.group}>
            <label className={styles.row}>
              <span>
                <strong>{t("general.defaultLayout")}</strong>
                <small>{t("general.defaultLayoutDescription")}</small>
              </span>
              <select
                onChange={(event) => {
                  const next =
                    event.currentTarget.value === "masonry"
                      ? "masonry"
                      : "grid";
                  setLayout(next);
                }}
                value={layout}
              >
                <option value="grid">{t("browse.layoutGrid")}</option>
                <option value="masonry">{t("browse.layoutMasonry")}</option>
              </select>
            </label>
            <label className={styles.row}>
              <span>
                <strong>{t("general.defaultSort")}</strong>
                <small>{t("general.defaultSortDescription")}</small>
              </span>
              <select
                onChange={(event) =>
                  setMediaSort(
                    event.currentTarget.value as MediaSortPreference,
                  )
                }
                value={mediaSort}
              >
                <option value="contextual">
                  {t("general.defaultSortContextual")}
                </option>
                <option value="name:asc">{t("browse.sortNameAscending")}</option>
                <option value="name:desc">{t("browse.sortNameDescending")}</option>
                <option value="modifiedAt:desc">
                  {t("browse.sortModifiedDescending")}
                </option>
                <option value="modifiedAt:asc">
                  {t("browse.sortModifiedAscending")}
                </option>
                <option value="size:desc">{t("browse.sortSizeDescending")}</option>
                <option value="size:asc">{t("browse.sortSizeAscending")}</option>
              </select>
            </label>
            <label className={styles.row}>
              <span>
                <strong>{t("general.defaultPreview")}</strong>
                <small>{t("general.defaultPreviewDescription")}</small>
              </span>
              <Switch
                checked={previewPinned}
                onChange={(event) =>
                  setPreviewPinned(event.currentTarget.checked)
                }
              />
            </label>
            <label className={styles.row}>
              <span>
                <strong>{t("general.previewAutoplay")}</strong>
                <small>{t("general.previewAutoplayDescription")}</small>
              </span>
              <Switch
                checked={previewAutoplay}
                onChange={(event) =>
                  setPreviewAutoplay(event.currentTarget.checked)
                }
              />
            </label>
          </div>
        </section>
        <div className={styles.savebar}>
          <span>{dirty ? t("general.unsaved") : t("general.noChanges")}</span>
          <div>
            <Button disabled={!dirty} onClick={reset}>
              {t("general.restore")}
            </Button>
            <Button disabled={!dirty} type="submit" variant="primary">
              {t("general.save")}
            </Button>
          </div>
        </div>
      </form>
    </ManagementShell>
  );
}
