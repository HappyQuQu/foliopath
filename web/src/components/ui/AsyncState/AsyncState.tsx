import { ArrowClockwise, CircleNotch } from "@phosphor-icons/react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { Button } from "../Button/Button";
import styles from "./AsyncState.module.css";

export function LoadingState({ label }: { label?: string }) {
  const { t } = useLocale();

  return (
    <div className={styles.state} role="status">
      <CircleNotch aria-hidden="true" className={styles.spinner} size={24} />
      <span>{label ?? t("common.loading")}</span>
    </div>
  );
}

export function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  const { t } = useLocale();

  return (
    <div className={styles.state} role="alert">
      <strong>{t("common.loadFailed")}</strong>
      <span>{message}</span>
      {onRetry && (
        <Button onClick={onRetry} size="small">
          <ArrowClockwise aria-hidden="true" size={17} />
          {t("common.tryAgain")}
        </Button>
      )}
    </div>
  );
}
