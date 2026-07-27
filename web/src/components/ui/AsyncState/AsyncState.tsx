import { ArrowClockwise, CircleNotch } from "@phosphor-icons/react";

import { Button } from "../Button/Button";
import styles from "./AsyncState.module.css";

export function LoadingState({ label = "正在载入…" }: { label?: string }) {
  return (
    <div className={styles.state} role="status">
      <CircleNotch aria-hidden="true" className={styles.spinner} size={24} />
      <span>{label}</span>
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
  return (
    <div className={styles.state} role="alert">
      <strong>暂时无法载入</strong>
      <span>{message}</span>
      {onRetry && (
        <Button onClick={onRetry} size="small">
          <ArrowClockwise aria-hidden="true" size={17} />
          重新尝试
        </Button>
      )}
    </div>
  );
}
