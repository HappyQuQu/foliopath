import { forwardRef, type InputHTMLAttributes } from "react";

import styles from "./Input.module.css";

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, invalid = false, ...props },
  ref,
) {
  return (
    <input
      {...props}
      ref={ref}
      className={[styles.input, className].filter(Boolean).join(" ")}
      aria-invalid={invalid || undefined}
    />
  );
});
