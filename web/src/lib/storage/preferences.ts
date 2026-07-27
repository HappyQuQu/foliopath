const preferenceNamespace = "foliopath.preferences.v1";

export type ThemePreference = "system" | "light" | "dark";
export type LocalePreference = "en" | "zh-CN";

interface Preferences {
  locale?: LocalePreference;
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
