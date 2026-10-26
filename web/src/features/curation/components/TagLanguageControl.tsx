import { useId } from "react";
import styles from "./TagLanguageControl.module.css";
import { Select } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { useTagDisplay } from "../../../lib/i18n/tagDisplay";

/** Same browser display preference in settings and the photo workspace. */
export function TagLanguageControl({ compact = false }: { compact?: boolean }) {
  const { t } = useLocale();
  const { language, setLanguage } = useTagDisplay();
  const id = useId();
  return <label className={compact ? styles.compact : undefined} htmlFor={id}>
    {t(compact ? "tagLanguage.shortLabel" : "tagLanguage.label")}
    <Select aria-label={t("tagLanguage.label")} id={id} value={language} onChange={event => {
      const value = event.currentTarget.value;
      if (value === "interface" || value === "zh-CN" || value === "en" || value === "bilingual") setLanguage(value);
    }}>
      <option value="interface">{t("tagLanguage.interface")}</option>
      <option value="zh-CN">中文</option>
      <option value="en">English</option>
      <option value="bilingual">{t("tagLanguage.bilingual")}</option>
    </Select>
  </label>;
}
