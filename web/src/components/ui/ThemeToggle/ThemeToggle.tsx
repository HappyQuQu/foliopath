import { Moon, Sun } from "@phosphor-icons/react";

import { useTheme } from "../../../lib/theme/ThemeProvider";
import { IconButton } from "../Button/IconButton";
import styles from "./ThemeToggle.module.css";

export function ThemeToggle() {
  const { resolvedTheme, setPreference } = useTheme();
  const light = resolvedTheme === "light";
  const label = light ? "切换到深色主题" : "切换到浅色主题";

  return (
    <IconButton
      className={styles.button}
      label={label}
      onClick={() => setPreference(light ? "dark" : "light")}
    >
      {light ? (
        <Moon aria-hidden="true" size={20} />
      ) : (
        <Sun aria-hidden="true" size={20} />
      )}
    </IconButton>
  );
}
