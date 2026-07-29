import { Moon, Sun } from "@phosphor-icons/react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useTheme } from "../../../lib/theme/ThemeProvider";
import { IconButton } from "../Button/IconButton";
import styles from "./ThemeToggle.module.css";

export function ThemeToggle() {
  const { t } = useLocale();
  const { resolvedTheme, setPreference } = useTheme();
  const light = resolvedTheme === "light";
  const label = light ? t("theme.toDark") : t("theme.toLight");

  return (
    <IconButton
      className={styles.button}
      label={label}
      onClick={() => setPreference(light ? "dark" : "light")}
    >
      {light ? (
        <Sun aria-hidden="true" size={20} />
      ) : (
        <Moon aria-hidden="true" size={20} />
      )}
    </IconButton>
  );
}
