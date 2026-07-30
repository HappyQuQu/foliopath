import type { InputHTMLAttributes } from "react";

import styles from "./Switch.module.css";

export type SwitchProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type">;

export function Switch({ className, ...props }: SwitchProps) {
  return (
    <input
      {...props}
      className={`${styles.switch} ${className ?? ""}`.trim()}
      role="switch"
      type="checkbox"
    />
  );
}
