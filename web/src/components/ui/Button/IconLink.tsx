import type { ReactNode } from "react";
import { Link, type LinkProps } from "react-router-dom";

import styles from "./IconButton.module.css";

export interface IconLinkProps extends Omit<LinkProps, "children"> {
  children: ReactNode;
  current?: boolean;
  label: string;
}

export function IconLink({
  children,
  className,
  current = false,
  label,
  ...props
}: IconLinkProps) {
  return (
    <Link
      {...props}
      aria-current={current ? "page" : undefined}
      aria-label={label}
      className={[styles.button, className].filter(Boolean).join(" ")}
      title={label}
    >
      {children}
    </Link>
  );
}
