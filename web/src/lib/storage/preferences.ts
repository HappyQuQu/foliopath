const preferenceNamespace = "foliopath.preferences.v1";

export type ThemePreference = "system" | "light" | "dark";
export type LocalePreference = "en" | "zh-CN";
export type MediaLayoutPreference = "grid" | "masonry";
export type MediaSortPreference =
  | "contextual"
  | "name:asc"
  | "name:desc"
  | "modifiedAt:asc"
  | "modifiedAt:desc"
  | "size:asc"
  | "size:desc";

interface Preferences {
  aiOperationIds?: string[];
  locale?: LocalePreference;
  mediaLayout?: MediaLayoutPreference;
  mediaSort?: MediaSortPreference;
  dismissedCompletedNotifications?: string[];
  acknowledgedMediaFailureRevision?: string;
  clearedMediaFailureRevision?: string;
  previewAutoplay?: boolean;
  previewMuted?: boolean;
  previewPinned?: boolean;
  previewWidth?: number;
  sidebarWidth?: number;
  theme?: ThemePreference;
}

export function readAIOperationIds(): string[] {
  const values = readPreferences().aiOperationIds;
  if (!Array.isArray(values)) return [];
  return values
    .filter((value): value is string => typeof value === "string" && value.length > 0)
    .slice(0, 50);
}

export function writeAIOperationIds(values: string[]): void {
  writePreferences({
    ...readPreferences(),
    aiOperationIds: values
      .filter((value) => typeof value === "string" && value.length > 0)
      .slice(0, 50),
  });
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

export function readMediaSortPreference(): MediaSortPreference {
  const value = readPreferences().mediaSort;
  return value === "name:asc" ||
    value === "name:desc" ||
    value === "modifiedAt:asc" ||
    value === "modifiedAt:desc" ||
    value === "size:asc" ||
    value === "size:desc"
    ? value
    : "contextual";
}

export function writeMediaSortPreference(mediaSort: MediaSortPreference): void {
  writePreferences({ ...readPreferences(), mediaSort });
}

export function readPreviewPinnedPreference(): boolean {
  return readPreferences().previewPinned === true;
}

export function writePreviewPinnedPreference(previewPinned: boolean): void {
  writePreferences({ ...readPreferences(), previewPinned });
}

export function readPreviewAutoplayPreference(): boolean {
  return readPreferences().previewAutoplay !== false;
}

export function writePreviewAutoplayPreference(previewAutoplay: boolean): void {
  writePreferences({ ...readPreferences(), previewAutoplay });
}

export function readPreviewMutedPreference(): boolean {
  return readPreferences().previewMuted !== false;
}

export function writePreviewMutedPreference(previewMuted: boolean): void {
  writePreferences({ ...readPreferences(), previewMuted });
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

export function readDismissedCompletedNotifications(): string[] {
  const values = readPreferences().dismissedCompletedNotifications;
  if (!Array.isArray(values)) return [];
  return values.filter(
    (value): value is string =>
      typeof value === "string" && /^scan_[1-9][0-9]*$/.test(value),
  ).slice(-200);
}

export function writeDismissedCompletedNotifications(values: string[]): void {
  writePreferences({
    ...readPreferences(),
    dismissedCompletedNotifications: values
      .filter((value) => /^scan_[1-9][0-9]*$/.test(value))
      .slice(-200),
  });
}

export function readAcknowledgedMediaFailureRevision(): string | undefined {
  const value = readPreferences().acknowledgedMediaFailureRevision;
  return typeof value === "string" && /^mfailrev_[1-9][0-9]*_[1-9][0-9]*$/.test(value)
    ? value
    : undefined;
}

export function writeAcknowledgedMediaFailureRevision(value: string): void {
  if (!/^mfailrev_[1-9][0-9]*_[1-9][0-9]*$/.test(value)) return;
  writePreferences({ ...readPreferences(), acknowledgedMediaFailureRevision: value });
}

export function readClearedMediaFailureRevision(): string | undefined {
  const value = readPreferences().clearedMediaFailureRevision;
  return typeof value === "string" && /^mfailrev_[1-9][0-9]*_[1-9][0-9]*$/.test(value)
    ? value
    : undefined;
}

export function writeClearedMediaFailureRevision(value: string): void {
  if (!/^mfailrev_[1-9][0-9]*_[1-9][0-9]*$/.test(value)) return;
  writePreferences({ ...readPreferences(), clearedMediaFailureRevision: value });
}

export function clearClearedMediaFailureRevision(): void {
  const { clearedMediaFailureRevision: _revision, ...preferences } = readPreferences();
  writePreferences(preferences);
}
