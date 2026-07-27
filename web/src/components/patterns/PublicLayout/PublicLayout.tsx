import type { ReactNode } from "react";

import { ThemeToggle } from "../../ui/ThemeToggle/ThemeToggle";
import styles from "./PublicLayout.module.css";

export function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <div className={styles.page}>
      <a className={styles.skipLink} href="#main">
        跳到主要内容
      </a>
      <header className={styles.brand}>
        <strong>FolioPath</strong>
        <ThemeToggle />
      </header>
      <main className={styles.main} id="main" tabIndex={-1}>
        {children}
      </main>
      <footer>您的原始媒体始终保持只读</footer>
    </div>
  );
}
