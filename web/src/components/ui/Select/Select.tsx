import { CaretDown } from "@phosphor-icons/react";
import { forwardRef, type SelectHTMLAttributes } from "react";

import styles from "./Select.module.css";

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  invalid?: boolean;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className, invalid = false, ...props },
  ref,
) {
  return (
    <span className={styles.root}>
      <select
        {...props}
        ref={ref}
        aria-invalid={invalid || undefined}
        className={[styles.select, className].filter(Boolean).join(" ")}
      />
      <CaretDown aria-hidden="true" className={styles.icon} size={16} weight="bold" />
    </span>
  );
});
