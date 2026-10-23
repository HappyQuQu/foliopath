import {
  ArrowClockwise,
  CircleNotch,
  CloudSlash,
  ImageSquare,
  type Icon,
} from "@phosphor-icons/react";
import type { ReactNode } from "react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { Button } from "../Button/Button";
import styles from "./AsyncState.module.css";

type Density = "default" | "compact";
export function LoadingState({ label, density = "default" }: { label?: string; density?: Density }) {
  const { t } = useLocale();

  return (
    <div className={`${styles.state} ${density === "compact" ? styles.compact : ""}`} role="status">
      <CircleNotch aria-hidden="true" className={styles.spinner} size={24} />
      <span>{label ?? t("common.loading")}</span>
    </div>
  );
}

export function ErrorState({
  message,
  onRetry,
  density = "default",
}: {
  message: string;
  onRetry?: () => void;
  density?: Density;
}) {
  const { t } = useLocale();

  return (
    <div className={`${styles.state} ${density === "compact" ? styles.compact : ""}`} role="alert">
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

export function EmptyState({
  action,
  description,
  icon: StateIcon = ImageSquare,
  title,
  density = "default",
}: {
  action?: ReactNode;
  description: string;
  icon?: Icon;
  title: string;
  density?: Density;
}) {
  return (
    <div className={`${styles.state} ${styles.empty} ${density === "compact" ? styles.compact : ""}`}>
      <StateIcon aria-hidden="true" className={styles.stateIcon} size={30} />
      <div className={styles.copy}>
        <strong>{title}</strong>
        <span>{description}</span>
      </div>
      {action}
    </div>
  );
}

export function OfflineState({
  action,
  description,
  title,
}: {
  action?: ReactNode;
  description: string;
  title: string;
}) {
  return (
    <div className={`${styles.state} ${styles.offline}`} role="status">
      <CloudSlash aria-hidden="true" className={styles.stateIcon} size={30} />
      <div className={styles.copy}>
        <strong>{title}</strong>
        <span>{description}</span>
      </div>
      {action}
    </div>
  );
}
