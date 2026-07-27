import { Info, WarningCircle, X } from "@phosphor-icons/react";
import type { ReactNode } from "react";

import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { IconButton } from "../Button/IconButton";
import styles from "./InlineStatus.module.css";

export type InlineStatusTone = "info" | "danger";

export interface InlineStatusProps {
  children: ReactNode;
  dismissLabel?: string;
  onDismiss?: () => void;
  tone?: InlineStatusTone;
}

export function InlineStatus({
  children,
  dismissLabel,
  onDismiss,
  tone = "info",
}: InlineStatusProps) {
  const { t } = useLocale();
  const Icon = tone === "danger" ? WarningCircle : Info;

  return (
    <div
      className={`${styles.status} ${styles[tone]}`}
      role={tone === "danger" ? "alert" : "status"}
    >
      <Icon aria-hidden="true" size={19} weight="fill" />
      <span>{children}</span>
      {onDismiss && (
        <IconButton
          className={styles.close}
          label={dismissLabel ?? t("common.closeNotice")}
          onClick={onDismiss}
        >
          <X aria-hidden="true" size={16} />
        </IconButton>
      )}
    </div>
  );
}
