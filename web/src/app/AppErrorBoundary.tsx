import { Component, type ErrorInfo, type ReactNode } from "react";

import { Button } from "../components/ui/Button/Button";
import {
  translate,
  type Locale,
} from "../lib/i18n/LocaleProvider";
import styles from "./AppErrorBoundary.module.css";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

export class AppErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    if (import.meta.env.DEV) {
      console.error("Unhandled application error", error, info.componentStack);
    }
  }

  private reload = () => window.location.reload();

  render() {
    if (!this.state.hasError) return this.props.children;
    const locale: Locale = document.documentElement.lang === "en" ? "en" : "zh-CN";

    return (
      <main className={styles.page}>
        <section className={styles.card} role="alert">
          <p className={styles.eyebrow}>FolioPath</p>
          <h1>{translate(locale, "error.renderTitle")}</h1>
          <p>{translate(locale, "error.renderBody")}</p>
          <Button onClick={this.reload}>{translate(locale, "error.reload")}</Button>
        </section>
      </main>
    );
  }
}
