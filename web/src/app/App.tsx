import styles from "./App.module.css";

export function App() {
  return (
    <div className={styles.shell}>
      <a className={styles.skipLink} href="#main">
        跳到主要内容
      </a>
      <header className={styles.header}>
        <strong>FolioPath</strong>
        <span>前端基础层</span>
      </header>
      <main className={styles.main} id="main" tabIndex={-1}>
        <section className={styles.status} aria-labelledby="foundation-title">
          <p>Stage 1</p>
          <h1 id="foundation-title">安全的产品界面正在准备中</h1>
          <span>
            应用启动、主题和共享组件基础已启用。认证产品路由将在 Backend Ready 后接入。
          </span>
        </section>
      </main>
    </div>
  );
}
