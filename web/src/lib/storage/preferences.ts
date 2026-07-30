const preferenceNamespace = "foliopath.preferences.v1";

export type ThemePreference = "system" | "light" | "dark";
export type LocalePreference = "en" | "zh-CN";
export type MediaLayoutPreference = "grid" | "masonry";

interface Preferences {
  locale?: LocalePreference;
  mediaLayout?: MediaLayoutPreference;
  previewPinned?: boolean;
  previewWidth?: number;
  sidebarWidth?: number;
  theme?: ThemePreference;
}

function readPreferences(): Preferences {
  try {
    const value = window.localStorage.getItem(preferenceNamespace);
    if (value === null) return {};
    const parsed: unknown = JSON.parse(value);
    if (typeof parsed !== "object" || parsed === null) return {};
    return parsed as Preferences;
  } catch {
    return {};
  }
}

function writePreferences(preferences: Preferences): void {
  try {
    window.localStorage.setItem(preferenceNamespace, JSON.stringify(preferences));
  } catch {
    // A blocked or full storage area must not prevent the application from running.
  }
}

export function readThemePreference(): ThemePreference {
  const theme = readPreferences().theme;
  return theme === "light" || theme === "dark" || theme === "system" ? theme : "system";
}

export function writeThemePreference(theme: ThemePreference): void {
  writePreferences({ ...readPreferences(), theme });
}

export function readLocalePreference(): LocalePreference | undefined {
  const locale = readPreferences().locale;
  return locale === "en" || locale === "zh-CN" ? locale : undefined;
}

export function writeLocalePreference(locale: LocalePreference): void {
  writePreferences({ ...readPreferences(), locale });
}

export function clearLocalePreference(): void {
  const { locale: _locale, ...preferences } = readPreferences();
  writePreferences(preferences);
}

export function readMediaLayoutPreference(): MediaLayoutPreference {
  const layout = readPreferences().mediaLayout;
  return layout === "masonry" ? "masonry" : "grid";
}

export function writeMediaLayoutPreference(
  mediaLayout: MediaLayoutPreference,
): void {
  writePreferences({ ...readPreferences(), mediaLayout });
}

export function readPreviewPinnedPreference(): boolean {
  return readPreferences().previewPinned === true;
}

export function writePreviewPinnedPreference(previewPinned: boolean): void {
  writePreferences({ ...readPreferences(), previewPinned });
}

function readWidthPreference(
  value: number | undefined,
  fallback: number,
): number {
  return typeof value === "number" &&
    Number.isFinite(value) &&
    value >= 160 &&
    value <= 800
    ? value
    : fallback;
}

export function readSidebarWidthPreference(): number {
  return readWidthPreference(readPreferences().sidebarWidth, 272);
}

export function writeSidebarWidthPreference(sidebarWidth: number): void {
  writePreferences({ ...readPreferences(), sidebarWidth });
}

export function readPreviewWidthPreference(): number {
  return readWidthPreference(readPreferences().previewWidth, 406);
}

export function writePreviewWidthPreference(previewWidth: number): void {
  writePreferences({ ...readPreferences(), previewWidth });
}
