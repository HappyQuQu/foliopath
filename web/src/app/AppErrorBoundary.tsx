import { Component, type ErrorInfo, type ReactNode } from "react";

import { Button } from "../components/ui/Button/Button";
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

    return (
      <main className={styles.page}>
        <section className={styles.card} role="alert">
          <p className={styles.eyebrow}>FolioPath</p>
          <h1>界面暂时无法显示</h1>
          <p>您的媒体没有被修改。请重新载入界面；如果问题持续，请检查服务状态。</p>
          <Button onClick={this.reload}>重新载入</Button>
        </section>
      </main>
    );
  }
}
