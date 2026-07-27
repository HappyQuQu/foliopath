import { useLocale } from "../../../lib/i18n/LocaleProvider";
import styles from "./LocaleSelect.module.css";

export function LocaleSelect() {
  const { locale, setLocale, t } = useLocale();

  return (
    <label className={styles.field}>
      <span>{t("account.language")}</span>
      <select
        className={styles.select}
        onChange={(event) => setLocale(event.currentTarget.value === "en" ? "en" : "zh-CN")}
        value={locale}
      >
        <option value="zh-CN">简体中文</option>
        <option value="en">English</option>
      </select>
    </label>
  );
}
