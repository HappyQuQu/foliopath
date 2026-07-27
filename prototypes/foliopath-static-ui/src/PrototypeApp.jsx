import { useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  ArrowsOut,
  CaretDown,
  Check,
  CircleNotch,
  Database,
  FolderOpen,
  GearSix,
  ImageSquare,
  Info,
  ListBullets,
  MagnifyingGlass,
  Moon,
  Play,
  Plus,
  PushPin,
  SignOut,
  Sun,
  Warning,
  X,
} from "@phosphor-icons/react";
import { App } from "./App.jsx";

const themeKey = "foliopath-prototype-theme";

const screens = [
  { path: "/setup/admin", group: "启动与认证", label: "首次管理员设置" },
  { path: "/login", group: "启动与认证", label: "登录与会话失效" },
  { path: "/welcome", group: "媒体库", label: "无媒体库欢迎页" },
  { path: "/settings/libraries/new", group: "媒体库", label: "新建媒体库向导" },
  { path: "/libraries/family/browse/kyoto", group: "浏览与查看", label: "主浏览界面" },
  { path: "/libraries/family/browse/kyoto?recursive=1", group: "浏览与查看", label: "递归浏览状态" },
  { path: "/libraries/family/search?q=京都", group: "浏览与查看", label: "搜索结果" },
  { path: "/libraries/family/search/empty", group: "浏览与查看", label: "搜索空态与故障" },
  { path: "/libraries/family/media/pagoda", group: "浏览与查看", label: "全屏查看器" },
  { path: "/libraries/family/media/offline", group: "浏览与查看", label: "查看器媒体离线" },
  { path: "/status/scanning", group: "扫描状态", label: "扫描进行中" },
  { path: "/status/offline", group: "扫描状态", label: "媒体库离线" },
  { path: "/status/error", group: "扫描状态", label: "扫描失败与部分不可读" },
  { path: "/settings/libraries", group: "设置", label: "媒体库管理" },
  { path: "/settings/general", group: "设置", label: "通用设置" },
  { path: "/system/unavailable", group: "系统", label: "服务不可用" },
];

const reviewMedia = [
  { name: "2026-07-21 19-12-43.jpg", place: "京都 / 八坂塔", src: "/media/kyoto-pagoda-clean.jpg", kind: "图片" },
  { name: "2026-07-21 16-45-10.mp4", place: "京都 / 伏见稻荷大社", src: "/media/torii-tunnel.jpg", kind: "视频" },
  { name: "2026-07-20 10-22-31.jpg", place: "京都 / 金阁寺", src: "/media/golden-pavilion.jpg", kind: "图片" },
  { name: "2026-07-19 09-11-05.mp4", place: "京都 / 岚山", src: "/media/bamboo-grove.jpg", kind: "视频" },
  { name: "2026-07-18 18-33-27.jpg", place: "京都 / 祇园", src: "/media/gion-kimono.jpg", kind: "图片" },
  { name: "2026-07-17 19-47-56.jpg", place: "京都 / 祇园", src: "/media/gion-evening.jpg", kind: "图片" },
];

function currentLocation() {
  const location = `${window.location.pathname}${window.location.search}`;
  return location === "/" ? "/libraries/family/browse/kyoto" : location;
}

function usePrototypeRoute() {
  const [route, setRoute] = useState(currentLocation);
  useEffect(() => {
    const update = () => setRoute(currentLocation());
    window.addEventListener("popstate", update);
    return () => window.removeEventListener("popstate", update);
  }, []);
  const navigate = (path) => {
    window.history.pushState({}, "", path);
    setRoute(path);
    window.scrollTo({ top: 0, behavior: "instant" });
  };
  return [route, navigate];
}

function usePrototypeTheme() {
  const [theme, setTheme] = useState(() => {
    const stored = localStorage.getItem(themeKey);
    if (stored === "light" || stored === "dark") return stored;
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  });
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    localStorage.setItem(themeKey, theme);
  }, [theme]);
  return [theme, setTheme];
}

function PrototypeNavigator({ route, navigate, theme, setTheme }) {
  const [open, setOpen] = useState(false);
  const routePath = route.split("?")[0];
  const exactIndex = screens.findIndex((screen) => screen.path === route);
  const pathIndex = screens.findIndex((screen) => screen.path.split("?")[0] === routePath);
  const currentIndex = exactIndex >= 0 ? exactIndex : pathIndex >= 0 ? pathIndex : 4;
  const groups = useMemo(
    () =>
      screens.reduce((result, screen) => {
        (result[screen.group] ??= []).push(screen);
        return result;
      }, {}),
    [],
  );

  return (
    <div className={`prototype-nav ${open ? "is-open" : ""}`}>
      <button className="prototype-nav-trigger" type="button" onClick={() => setOpen(!open)}>
        <ListBullets size={18} />
        原型目录
        <span>{currentIndex + 1}/{screens.length}</span>
      </button>
      {open && (
        <aside className="prototype-nav-panel" aria-label="原型验收目录">
          <header>
            <div>
              <strong>完整界面验收</strong>
              <small>仅原型可见，不属于产品导航</small>
            </div>
            <button type="button" aria-label="关闭原型目录" onClick={() => setOpen(false)}>
              <X size={18} />
            </button>
          </header>
          <div className="prototype-theme">
            <span>外观</span>
            <button
              type="button"
              onClick={() => setTheme(theme === "light" ? "dark" : "light")}
            >
              {theme === "light" ? <Moon size={17} /> : <Sun size={17} />}
              {theme === "light" ? "切换深色" : "切换浅色"}
            </button>
          </div>
          <div className="prototype-screen-list">
            {Object.entries(groups).map(([group, items]) => (
              <section key={group}>
                <h2>{group}</h2>
                {items.map((screen) => (
                  <button
                    key={screen.path}
                    type="button"
                    className={screen.path === route ? "is-current" : ""}
                    onClick={() => {
                      navigate(screen.path);
                      setOpen(false);
                    }}
                  >
                    <span>{screens.indexOf(screen) + 1}</span>
                    {screen.label}
                  </button>
                ))}
              </section>
            ))}
          </div>
          <footer>
            <button
              type="button"
              disabled={currentIndex === 0}
              onClick={() => navigate(screens[currentIndex - 1].path)}
            >
              <ArrowLeft size={16} /> 上一页
            </button>
            <button
              type="button"
              disabled={currentIndex === screens.length - 1}
              onClick={() => navigate(screens[currentIndex + 1].path)}
            >
              下一页 <ArrowRight size={16} />
            </button>
          </footer>
        </aside>
      )}
    </div>
  );
}

function ThemeButton({ theme, setTheme }) {
  return (
    <button
      className="quiet-icon"
      type="button"
      aria-label={theme === "light" ? "切换到深色主题" : "切换到浅色主题"}
      onClick={() => setTheme(theme === "light" ? "dark" : "light")}
    >
      {theme === "light" ? <Moon size={20} /> : <Sun size={20} />}
    </button>
  );
}

function PublicFrame({ children, theme, setTheme }) {
  return (
    <main className="public-page">
      <header className="public-brand">
        <strong>FolioPath</strong>
        <ThemeButton theme={theme} setTheme={setTheme} />
      </header>
      {children}
      <footer>您的原始媒体始终保持只读</footer>
    </main>
  );
}

function AuthCard({ mode, navigate }) {
  const setup = mode === "setup";
  const [showNotice, setShowNotice] = useState(!setup);
  const [submitted, setSubmitted] = useState(false);
  return (
    <section className="auth-card">
      <div className="auth-mark"><ImageSquare size={30} weight="duotone" /></div>
      <p className="eyebrow">{setup ? "首次使用" : "欢迎回来"}</p>
      <h1>{setup ? "创建管理员账户" : "登录 FolioPath"}</h1>
      <p>{setup ? "此账户用于管理媒体库、扫描任务与系统设置。" : "使用管理员账户继续访问您的媒体库。"}</p>
      {showNotice && (
        <div className="inline-notice">
          <Info size={18} />
          <span>为了保护您的媒体库，会话已过期。请重新登录。</span>
          <button type="button" aria-label="关闭提示" onClick={() => setShowNotice(false)}><X size={16} /></button>
        </div>
      )}
      <form onSubmit={(event) => { event.preventDefault(); setSubmitted(true); }}>
        <label>用户名<input defaultValue={setup ? "" : "admin"} autoComplete="username" /></label>
        <label>密码<input type="password" defaultValue={setup ? "" : "foliopath-demo"} autoComplete={setup ? "new-password" : "current-password"} /></label>
        {setup && <label>确认密码<input type="password" autoComplete="new-password" /></label>}
        <button className="primary-button" type="submit">{setup ? "创建账户" : "登录"}</button>
      </form>
      {submitted && (
        <div className="success-line" role="status">
          <Check size={17} weight="bold" />
          {setup ? "账户信息有效，可以进入下一步。" : "登录状态已验证。"}
        </div>
      )}
      <button className="text-button" type="button" onClick={() => navigate(setup ? "/welcome" : "/libraries/family/browse/kyoto")}>
        继续查看后续原型
      </button>
    </section>
  );
}

function WelcomePage({ navigate }) {
  const [help, setHelp] = useState(false);
  return (
    <ProductShell title="媒体库" navigate={navigate} active="libraries">
      <section className="welcome-panel">
        <div className="welcome-art"><FolderOpen size={58} weight="duotone" /></div>
        <p className="eyebrow">开始整理</p>
        <h1>还没有媒体库</h1>
        <p>选择服务器已允许的目录后，FolioPath 会建立只读索引与缩略图，不会移动或修改原文件。</p>
        <div className="welcome-actions">
          <button className="primary-button" type="button" onClick={() => navigate("/settings/libraries/new")}><Plus size={18} /> 新建媒体库</button>
          <button className="secondary-button" type="button" onClick={() => setHelp(!help)}>查看部署帮助</button>
        </div>
        {help && (
          <div className="help-box">
            <strong>在容器中挂载媒体目录</strong>
            <code>/host/photos:/library:ro</code>
            <span>完成挂载后返回这里，从 /library 下选择一个目录。</span>
          </div>
        )}
      </section>
    </ProductShell>
  );
}

function LibraryWizard({ navigate }) {
  const [step, setStep] = useState(1);
  const [name, setName] = useState("家庭影像");
  const [path, setPath] = useState("家庭影像");
  const choices = [
    { name: "家庭影像", meta: "可用 · 842 GB 可读", status: "ok" },
    { name: "摄影归档", meta: "可用 · 只读", status: "ok" },
    { name: "家庭影像/京都", meta: "与已选根目录重叠", status: "blocked" },
    { name: "离线磁盘", meta: "当前不可读", status: "blocked" },
  ];
  return (
    <ProductShell title="新建媒体库" navigate={navigate} active="libraries">
      <section className="wizard-page">
        <header className="page-heading">
          <div><p className="eyebrow">媒体库设置</p><h1>新建媒体库</h1><p>媒体库根目录创建后不可修改，但可以随时重命名或移除配置。</p></div>
          <button className="text-button" type="button" onClick={() => navigate("/settings/libraries")}>取消</button>
        </header>
        <ol className="stepper">
          {["命名", "选择目录", "确认"].map((label, index) => (
            <li key={label} className={step >= index + 1 ? "is-active" : ""}>
              <span>{step > index + 1 ? <Check size={15} /> : index + 1}</span>{label}
            </li>
          ))}
        </ol>
        <div className="wizard-card">
          {step === 1 && (
            <>
              <h2>为媒体库命名</h2><p>名称需要在此 FolioPath 实例中保持唯一。</p>
              <label className="field">媒体库名称<input value={name} onChange={(event) => setName(event.target.value)} /></label>
              <div className="field-hint"><Check size={16} /> 名称可用</div>
            </>
          )}
          {step === 2 && (
            <>
              <h2>选择允许的目录</h2><p>这里只显示服务器批准的 /library 子目录，不会暴露宿主机路径。</p>
              <div className="path-picker">
                <div className="path-crumb"><Database size={18} /><span>/library</span></div>
                {choices.map((choice) => (
                  <button
                    type="button"
                    key={choice.name}
                    disabled={choice.status === "blocked"}
                    className={path === choice.name ? "is-selected" : ""}
                    onClick={() => setPath(choice.name)}
                  >
                    <FolderOpen size={22} /><span><strong>{choice.name}</strong><small>{choice.meta}</small></span>
                    {path === choice.name && choice.status === "ok" && <Check size={18} weight="bold" />}
                  </button>
                ))}
              </div>
            </>
          )}
          {step === 3 && (
            <>
              <h2>确认并开始扫描</h2><p>创建后会立即启动首次完整扫描。浏览可以在扫描期间进行。</p>
              <dl className="review-list"><div><dt>名称</dt><dd>{name}</dd></div><div><dt>目录</dt><dd>/library/{path}</dd></div><div><dt>原始媒体</dt><dd>只读，不移动、不删除</dd></div><div><dt>首次扫描</dt><dd>创建后立即开始</dd></div></dl>
              <div className="inline-notice"><Info size={18} /><span>此路径在 API 中仅保存为 /library 的相对路径。</span></div>
            </>
          )}
          <footer>
            <button className="secondary-button" type="button" disabled={step === 1} onClick={() => setStep(step - 1)}>上一步</button>
            <button className="primary-button" type="button" onClick={() => step < 3 ? setStep(step + 1) : navigate("/status/scanning")}>{step < 3 ? "继续" : "创建并扫描"}</button>
          </footer>
        </div>
      </section>
    </ProductShell>
  );
}

function ProductShell({ title, active, navigate, children }) {
  const [drawer, setDrawer] = useState(false);
  return (
    <div className="product-shell">
      <button className={`product-scrim ${drawer ? "is-open" : ""}`} type="button" aria-label="关闭导航" onClick={() => setDrawer(false)} />
      <aside className={`product-sidebar ${drawer ? "is-open" : ""}`}>
        <strong className="product-brand">FolioPath</strong>
        <nav>
          <button type="button" className={active === "browse" ? "is-current" : ""} onClick={() => navigate("/libraries/family/browse/kyoto")}><ImageSquare size={20} />浏览</button>
          <button type="button" className={active === "search" ? "is-current" : ""} onClick={() => navigate("/libraries/family/search?q=京都")}><MagnifyingGlass size={20} />搜索</button>
          <button type="button" className={active === "libraries" ? "is-current" : ""} onClick={() => navigate("/settings/libraries")}><FolderOpen size={20} />媒体库</button>
          <button type="button" className={active === "settings" ? "is-current" : ""} onClick={() => navigate("/settings/general")}><GearSix size={20} />设置</button>
        </nav>
        <div className="product-sidebar-status"><span className="status-dot" />家庭影像在线<small>上次扫描：今天 18:42</small></div>
      </aside>
      <main className="product-main">
        <header className="product-topbar">
          <button className="mobile-menu" type="button" onClick={() => setDrawer(true)}><ListBullets size={21} /></button>
          <strong>{title}</strong><span>管理员</span>
        </header>
        {children}
      </main>
    </div>
  );
}

function SearchPage({ empty = false, navigate }) {
  const [scope, setScope] = useState("library");
  const [state, setState] = useState(empty ? "empty" : "results");
  const [preview, setPreview] = useState(null);
  return (
    <ProductShell title="搜索" active="search" navigate={navigate}>
      <section className="search-page">
        <header className="page-heading"><div><p className="eyebrow">家庭影像</p><h1>搜索媒体</h1></div></header>
        <div className="search-command">
          <MagnifyingGlass size={21} />
          <input defaultValue={empty ? "不存在的旅行" : "京都"} aria-label="搜索媒体" />
          <button className="primary-button" type="button" onClick={() => setState(empty ? "empty" : "results")}>搜索</button>
        </div>
        <div className="search-filters">
          <label>范围<select value={scope} onChange={(event) => setScope(event.target.value)}><option value="library">当前媒体库</option><option value="directory">当前目录</option><option value="all">全部媒体库</option></select></label>
          <label>类型<select><option>全部媒体</option><option>图片</option><option>视频</option></select></label>
          <label>日期<select><option>不限日期</option><option>最近 30 天</option><option>今年</option></select></label>
          {empty && <div className="state-tabs">{["empty", "offline", "error"].map((item) => <button type="button" key={item} className={state === item ? "is-active" : ""} onClick={() => setState(item)}>{item === "empty" ? "无结果" : item === "offline" ? "离线" : "失败"}</button>)}</div>}
        </div>
        {state === "results" ? (
          <>
            <div className="result-summary"><strong>找到 126 项</strong><span>按修改时间从新到旧</span></div>
            <div className={`search-results ${preview ? "has-preview" : ""}`}>
              <div className="result-grid">
                {reviewMedia.map((item) => <button type="button" key={item.name} onClick={() => setPreview(item)}><img src={item.src} alt="" /><span><strong>{item.name}</strong><small>{item.place}</small></span>{item.kind === "视频" && <i><Play size={12} weight="fill" />00:18</i>}</button>)}
              </div>
              {preview && <DockedPreview item={preview} onClose={() => setPreview(null)} navigate={navigate} />}
            </div>
          </>
        ) : (
          <SearchState state={state} />
        )}
      </section>
    </ProductShell>
  );
}

function SearchState({ state }) {
  const content = {
    empty: ["没有找到匹配的媒体", "试试缩短关键词、扩大搜索范围或清除筛选条件。"],
    offline: ["“摄影归档”当前离线", "仍可查看上次可靠扫描的索引；原始文件与新搜索结果暂不可用。"],
    error: ["搜索暂时不可用", "服务没有返回完整结果。您的索引未被删除，请稍后重试。"],
  }[state];
  return <div className={`large-state is-${state}`}>{state === "empty" ? <MagnifyingGlass size={38} /> : <Warning size={38} />}<h2>{content[0]}</h2><p>{content[1]}</p><button className="secondary-button" type="button">{state === "empty" ? "清除筛选" : "重试"}</button></div>;
}

function DockedPreview({ item, onClose, navigate }) {
  return (
    <aside className="mini-preview">
      <header><strong>预览</strong><div><button type="button"><PushPin size={16} />固定</button><button type="button" aria-label="关闭预览" onClick={onClose}><X size={18} /></button></div></header>
      <img src={item.src} alt={`${item.place}预览`} />
      <h2>{item.name}</h2><p>{item.place}</p>
      <dl><div><dt>类型</dt><dd>{item.kind}</dd></div><div><dt>大小</dt><dd>20.3 MB</dd></div></dl>
      <button className="secondary-button" type="button" onClick={() => navigate("/libraries/family/media/pagoda")}><ArrowsOut size={17} />进入完整查看器</button>
    </aside>
  );
}

function ViewerPage({ navigate, unavailable = false }) {
  const [fit, setFit] = useState(true);
  const [info, setInfo] = useState(true);
  return (
    <main className="viewer-page">
      <header><button type="button" onClick={() => navigate("/libraries/family/browse/kyoto")}><X size={20} />关闭</button><strong>2026-07-21 19-12-43.jpg</strong><div>{!unavailable && <button type="button" className={fit ? "is-active" : ""} onClick={() => setFit(!fit)}>{fit ? "适应" : "1:1"}</button>}<button type="button" onClick={() => setInfo(!info)}><Info size={19} />信息</button><button type="button"><ArrowsOut size={19} />全屏</button></div></header>
      <button className="viewer-arrow is-left" type="button" aria-label="上一项"><ArrowLeft size={24} /></button>
      <div className={`viewer-stage ${fit ? "is-fit" : "is-original"}`}>{unavailable ? <section className="viewer-unavailable"><span><Warning size={38} /></span><div><h2>媒体库当前离线</h2><p>媒体库挂载当前不可用。FolioPath 保留了上次可靠索引，不会把离线误判为空目录。</p></div><button className="secondary-button" type="button">重新检查</button></section> : <img src="/media/kyoto-pagoda-clean.jpg" alt="京都八坂塔" />}</div>
      <button className="viewer-arrow is-right" type="button" aria-label="下一项"><ArrowRight size={24} /></button>
      {info && <aside className="viewer-info"><h2>基本信息</h2><dl><div><dt>位置</dt><dd>旅行 / 日本 / 京都</dd></div><div><dt>拍摄日期</dt><dd>2026-07-21 19:12</dd></div><div><dt>分辨率</dt><dd>6000 × 3376</dd></div><div><dt>大小</dt><dd>20.3 MB</dd></div></dl></aside>}
      <footer><span>1 / 312</span><span>滚轮缩放 · 拖动平移 · Esc 退出</span></footer>
    </main>
  );
}

function StatusPage({ type, navigate }) {
  const [cancelled, setCancelled] = useState(false);
  const data = {
    scanning: { icon: <CircleNotch className="spin" size={38} />, eyebrow: "扫描进行中", title: cancelled ? "正在安全停止扫描" : "正在扫描“家庭影像”", body: cancelled ? "已提交取消请求；此前可靠的索引会被保留。" : "您可以继续浏览。扫描任务会在后台按有界队列进行。", tone: "info" },
    offline: { icon: <Warning size={38} />, eyebrow: "媒体库离线", title: "无法读取“摄影归档”", body: "FolioPath 保留了上次可靠扫描的 42,608 项索引，不会将离线误判为空目录。", tone: "warning" },
    error: { icon: <Warning size={38} />, eyebrow: "扫描未完成", title: "有 18 个目录无法读取", body: "本次扫描没有清理旧索引；安全提交的新项目仍然保留。修复权限后请重新完整扫描。", tone: "danger" },
  }[type];
  return (
    <ProductShell title="扫描状态" active="libraries" navigate={navigate}>
      <section className="status-page">
        <header className="page-heading"><div><p className="eyebrow">任务详情</p><h1>媒体库状态</h1></div></header>
        <div className={`status-hero is-${data.tone}`}><span>{data.icon}</span><div><p className="eyebrow">{data.eyebrow}</p><h2>{data.title}</h2><p>{data.body}</p></div></div>
        {type === "scanning" && <div className="progress-card"><div><strong>{cancelled ? "正在停止…" : "63%"}</strong><span>{cancelled ? "等待当前安全批次完成" : "已检查 18,842 / 29,760 项"}</span></div><progress max="100" value={cancelled ? 68 : 63} /><dl><div><dt>新发现</dt><dd>1,248</dd></div><div><dt>已更新</dt><dd>326</dd></div><div><dt>已跳过</dt><dd>38</dd></div><div><dt>耗时</dt><dd>12:48</dd></div></dl></div>}
        {type !== "scanning" && <div className="issue-list"><div><strong>{type === "offline" ? "最近可靠扫描" : "受影响的目录"}</strong><span>{type === "offline" ? "2026-07-23 18:42 · 42,608 项" : "旅行/日本/京都/归档 等 18 个"}</span></div><div><strong>建议操作</strong><span>{type === "offline" ? "检查只读挂载与磁盘连接" : "检查目录读取权限，然后启动完整扫描"}</span></div></div>}
        <div className="page-actions"><button className="secondary-button" type="button" onClick={() => navigate("/settings/libraries")}>返回媒体库</button>{type === "scanning" ? <button className="danger-button" type="button" disabled={cancelled} onClick={() => setCancelled(true)}>{cancelled ? "取消请求已提交" : "取消扫描"}</button> : <button className="primary-button" type="button">重新扫描</button>}</div>
      </section>
    </ProductShell>
  );
}

function LibrariesSettings({ navigate }) {
  const [dialog, setDialog] = useState(null);
  return (
    <ProductShell title="媒体库" active="libraries" navigate={navigate}>
      <section className="settings-page">
        <header className="page-heading"><div><p className="eyebrow">管理</p><h1>媒体库</h1><p>添加、重命名、扫描或移除媒体库配置。原始文件不会被修改。</p></div><button className="primary-button" type="button" onClick={() => navigate("/settings/libraries/new")}><Plus size={18} />新建媒体库</button></header>
        <div className="library-table">
          <article><div className="library-avatar"><ImageSquare size={24} /></div><div><h2>家庭影像</h2><p>/library/家庭影像 · 29,760 项</p><span className="status-pill is-scanning"><CircleNotch className="spin" size={14} />扫描中 · 63%</span></div><div className="row-actions"><button type="button" onClick={() => navigate("/status/scanning")}>查看状态</button><button type="button" onClick={() => setDialog("rename")}>重命名</button><button type="button" onClick={() => setDialog("remove")}>移除</button></div></article>
          <article><div className="library-avatar"><FolderOpen size={24} /></div><div><h2>摄影归档</h2><p>/library/摄影归档 · 上次可靠索引 42,608 项</p><span className="status-pill is-offline"><Warning size={14} />离线</span></div><div className="row-actions"><button type="button" onClick={() => navigate("/status/offline")}>诊断</button><button type="button">重新扫描</button><button type="button" onClick={() => setDialog("remove")}>移除</button></div></article>
        </div>
      </section>
      {dialog && <ConfirmDialog type={dialog} onClose={() => setDialog(null)} />}
    </ProductShell>
  );
}

function ConfirmDialog({ type, onClose }) {
  return <div className="dialog-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}><section className="dialog" role="dialog" aria-modal="true" aria-labelledby="dialog-title"><header><h2 id="dialog-title">{type === "rename" ? "重命名媒体库" : "移除媒体库？"}</h2><button type="button" onClick={onClose}><X size={19} /></button></header>{type === "rename" ? <label className="field">新名称<input defaultValue="家庭影像" autoFocus /></label> : <><p>只会移除配置、索引、任务与缓存。原始媒体文件不会被删除。</p><div className="inline-notice"><Warning size={18} /><span>如需再次使用，需要重新添加并完整扫描。</span></div></>}<footer><button className="secondary-button" type="button" onClick={onClose}>取消</button><button className={type === "rename" ? "primary-button" : "danger-button"} type="button" onClick={onClose}>{type === "rename" ? "保存名称" : "确认移除"}</button></footer></section></div>;
}

function GeneralSettings({ navigate, theme, setTheme }) {
  const [saved, setSaved] = useState(false);
  return (
    <ProductShell title="设置" active="settings" navigate={navigate}>
      <section className="settings-page">
        <header className="page-heading"><div><p className="eyebrow">偏好</p><h1>通用设置</h1><p>这些设置适用于当前管理员与此 FolioPath 实例。</p></div></header>
        <div className="settings-stack">
          <section><h2>外观与语言</h2><div className="setting-row"><div><strong>主题</strong><span>默认跟随系统，也可以手动覆盖</span></div><div className="theme-segment">{["light", "dark"].map((item) => <button type="button" className={theme === item ? "is-active" : ""} key={item} onClick={() => setTheme(item)}>{item === "light" ? <Sun size={17} /> : <Moon size={17} />}{item === "light" ? "浅色" : "深色"}</button>)}</div></div><div className="setting-row"><div><strong>界面语言</strong><span>首次默认跟随浏览器语言</span></div><select><option>简体中文</option><option>English</option></select></div></section>
          <section><h2>扫描与缓存</h2><div className="setting-row"><div><strong>定时完整扫描</strong><span>用于权威地核对文件系统</span></div><select><option>每 24 小时</option><option>每周</option><option>关闭</option></select></div><div className="setting-row"><div><strong>缩略图缓存上限</strong><span>达到水位线后按 LRU 清理可重建内容</span></div><label className="inline-field"><input type="number" defaultValue="10" />GiB</label></div></section>
          <section><h2>账户</h2><div className="setting-row"><div><strong>管理员会话</strong><span>退出后需要重新输入凭据</span></div><button className="secondary-button" type="button" onClick={() => navigate("/login")}><SignOut size={17} />退出登录</button></div></section>
        </div>
        <div className="sticky-save"><span>{saved && <><Check size={17} />设置已保存</>}</span><button className="primary-button" type="button" onClick={() => setSaved(true)}>保存更改</button></div>
      </section>
    </ProductShell>
  );
}

function UnavailablePage({ navigate, theme, setTheme }) {
  return <PublicFrame theme={theme} setTheme={setTheme}><section className="unavailable-card"><div className="unavailable-icon"><Database size={34} /></div><p className="eyebrow">服务暂不可用</p><h1>FolioPath 无法完成启动</h1><p>应用数据目录当前不可写，系统已安全停止。媒体目录没有被修改。</p><div className="diagnostic"><strong>检查项目</strong><span>/app/data 的挂载与写入权限</span><span>SQLite 数据库是否位于本地文件系统</span><span>容器日志中的启动错误</span></div><button className="primary-button" type="button">重新检查</button><button className="text-button" type="button" onClick={() => navigate("/setup/admin")}>返回原型起点</button></section></PublicFrame>;
}

export function PrototypeApp() {
  const [route, navigate] = usePrototypeRoute();
  const [theme, setTheme] = usePrototypeTheme();
  const path = route.split("?")[0];
  let page;
  if (path === "/setup/admin") page = <PublicFrame theme={theme} setTheme={setTheme}><AuthCard mode="setup" navigate={navigate} /></PublicFrame>;
  else if (path === "/login") page = <PublicFrame theme={theme} setTheme={setTheme}><AuthCard mode="login" navigate={navigate} /></PublicFrame>;
  else if (path === "/welcome") page = <WelcomePage navigate={navigate} />;
  else if (path === "/settings/libraries/new") page = <LibraryWizard navigate={navigate} />;
  else if (path === "/libraries/family/search") page = <SearchPage navigate={navigate} />;
  else if (path === "/libraries/family/search/empty") page = <SearchPage empty navigate={navigate} />;
  else if (path === "/libraries/family/media/pagoda") page = <ViewerPage navigate={navigate} />;
  else if (path === "/libraries/family/media/offline") page = <ViewerPage navigate={navigate} unavailable />;
  else if (path === "/status/scanning") page = <StatusPage type="scanning" navigate={navigate} />;
  else if (path === "/status/offline") page = <StatusPage type="offline" navigate={navigate} />;
  else if (path === "/status/error") page = <StatusPage type="error" navigate={navigate} />;
  else if (path === "/settings/libraries") page = <LibrariesSettings navigate={navigate} />;
  else if (path === "/settings/general") page = <GeneralSettings navigate={navigate} theme={theme} setTheme={setTheme} />;
  else if (path === "/system/unavailable") page = <UnavailablePage navigate={navigate} theme={theme} setTheme={setTheme} />;
  else page = <App />;
  return <>{page}<PrototypeNavigator route={route} navigate={navigate} theme={theme} setTheme={setTheme} /></>;
}
