import { ArrowClockwise, Database } from "@phosphor-icons/react";

import { Button } from "../../components/ui/Button/Button";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import styles from "./SystemUnavailablePage.module.css";

export function SystemUnavailablePage({
  message,
  onRetry,
}: {
  message?: string;
  onRetry: () => void;
}) {
  const { t } = useLocale();

  return (
    <section className={styles.card} aria-labelledby="unavailable-title">
      <div className={styles.icon}>
        <Database aria-hidden="true" size={34} weight="duotone" />
      </div>
      <p className={styles.eyebrow}>{t("unavailable.eyebrow")}</p>
      <h1 id="unavailable-title">{t("unavailable.title")}</h1>
      <p className={styles.message}>{message ?? t("unavailable.default")}</p>
      <div className={styles.safety}>
        <strong>{t("unavailable.safety")}</strong>
        <span>{t("unavailable.readOnly")}</span>
        <span>{t("unavailable.noDiagnostics")}</span>
      </div>
      <Button className={styles.action} onClick={onRetry} variant="primary">
        <ArrowClockwise aria-hidden="true" size={18} />
        {t("common.retry")}
      </Button>
    </section>
  );
}
