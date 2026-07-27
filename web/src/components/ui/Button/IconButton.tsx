import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";

import styles from "./IconButton.module.css";

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  children: ReactNode;
  pressed?: boolean;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function IconButton(
    { children, className, label, pressed, type = "button", ...props },
    ref,
  ) {
    return (
      <button
        {...props}
        ref={ref}
        className={[styles.button, className].filter(Boolean).join(" ")}
        type={type}
        aria-label={label}
        aria-pressed={pressed}
        title={label}
      >
        {children}
      </button>
    );
  },
);
