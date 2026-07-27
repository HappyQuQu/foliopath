import { SignOut } from "@phosphor-icons/react";
import { useNavigate } from "react-router-dom";

import { Button } from "../../../components/ui/Button/Button";
import { ThemeToggle } from "../../../components/ui/ThemeToggle/ThemeToggle";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { useToast } from "../../../components/ui/Toast/ToastProvider";
import { useLogoutMutation } from "../queries";
import styles from "./AccountPage.module.css";

export function AccountPage({ session }: { session: AuthenticatedSession }) {
  const navigate = useNavigate();
  const logoutMutation = useLogoutMutation();
  const toast = useToast();

  async function handleLogout() {
    try {
      await logoutMutation.mutateAsync(session.csrfToken);
      navigate("/login", { replace: true });
    } catch {
      toast.show({ message: "暂时无法退出，请稍后重试。", tone: "danger" });
    }
  }

  return (
    <div className={styles.page}>
      <a className={styles.skipLink} href="#main">
        跳到主要内容
      </a>
      <header className={styles.header}>
        <strong>FolioPath</strong>
        <div>
          <span>{session.administrator.displayName}</span>
          <ThemeToggle />
        </div>
      </header>
      <main className={styles.main} id="main" tabIndex={-1}>
        <header className={styles.heading}>
          <p>偏好</p>
          <h1>通用设置</h1>
          <span>管理当前管理员会话与界面外观。</span>
        </header>

        <section className={styles.panel} aria-labelledby="appearance-title">
          <h2 id="appearance-title">外观</h2>
          <div className={styles.row}>
            <div>
              <strong>主题</strong>
              <span>默认跟随系统，也可以从页面右上角切换。</span>
            </div>
            <ThemeToggle />
          </div>
        </section>

        <section className={styles.panel} aria-labelledby="account-title">
          <h2 id="account-title">账户</h2>
          <div className={styles.row}>
            <div>
              <strong>{session.administrator.displayName}</strong>
              <span>用户名：{session.administrator.username}</span>
            </div>
            <Button
              loading={logoutMutation.isPending}
              onClick={handleLogout}
              variant="secondary"
            >
              <SignOut aria-hidden="true" size={17} />
              退出登录
            </Button>
          </div>
        </section>
      </main>
    </div>
  );
}
