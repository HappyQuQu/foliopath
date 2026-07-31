import { Translate } from "@phosphor-icons/react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { IconButton } from "../Button/IconButton";
import styles from "./LocaleToggle.module.css";

export function LocaleToggle() {
  const { locale, setLocale, t } = useLocale();
  const nextLocale = locale === "zh-CN" ? "en" : "zh-CN";
  const label =
    nextLocale === "en" ? t("locale.toEnglish") : t("locale.toChinese");

  return (
    <IconButton
      className={styles.button}
      label={label}
      onClick={() => setLocale(nextLocale)}
    >
      <Translate aria-hidden="true" size={20} />
    </IconButton>
  );
}
