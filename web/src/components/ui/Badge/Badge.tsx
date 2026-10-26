import type { ReactNode } from "react";
import styles from "./Badge.module.css";

/** Static evidence/category label; never implies an action or live status. */
export function Badge({ children, tone = "neutral" }: { children: ReactNode; tone?: "neutral" | "accent" }) {
  return <span className={styles.badge} data-tone={tone}>{children}</span>;
}
