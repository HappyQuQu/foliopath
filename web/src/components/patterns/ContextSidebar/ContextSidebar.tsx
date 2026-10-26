import { useId, type ReactNode } from "react";
import styles from "./ContextSidebar.module.css";

/** Layout only: feature owners supply URL-backed controls and bounded result lists. */
export function ContextSidebar({ children, description, density = "default", title }: {
  children: ReactNode; description?: string; density?: "default" | "compact"; title: string;
}) {
  const headingId = useId();
  return <section className={styles.panel} data-density={density} aria-labelledby={headingId}>
    <header><h2 id={headingId}>{title}</h2>{description && <p>{description}</p>}</header>
    {children}
  </section>;
}
