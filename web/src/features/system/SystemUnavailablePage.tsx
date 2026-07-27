import { ArrowClockwise, Database } from "@phosphor-icons/react";

import { Button } from "../../components/ui/Button/Button";
import styles from "./SystemUnavailablePage.module.css";

export function SystemUnavailablePage({
  message = "FolioPath 暂时无法完成启动，系统已安全停止。媒体目录没有被修改。",
  onRetry,
}: {
  message?: string;
  onRetry: () => void;
}) {
  return (
    <section className={styles.card} aria-labelledby="unavailable-title">
      <div className={styles.icon}>
        <Database aria-hidden="true" size={34} weight="duotone" />
      </div>
      <p className={styles.eyebrow}>服务暂不可用</p>
      <h1 id="unavailable-title">FolioPath 无法完成启动</h1>
      <p className={styles.message}>{message}</p>
      <div className={styles.safety}>
        <strong>安全状态</strong>
        <span>原始媒体保持只读</span>
        <span>没有显示内部路径或诊断信息</span>
      </div>
      <Button onClick={onRetry} variant="primary">
        <ArrowClockwise aria-hidden="true" size={18} />
        重新检查
      </Button>
    </section>
  );
}
