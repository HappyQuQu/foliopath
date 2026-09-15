import { useState } from "react";
import { MagnifyingGlass, Users, Folder, CaretRight, CaretDown, LockSimple, Info, Play, CheckCircle } from "@phosphor-icons/react";
import { ManagementShell } from "../../../web/src/components/patterns/ManagementShell/ManagementShell";
import { Button } from "../../../web/src/components/ui/Button/Button";
import { Switch } from "../../../web/src/components/ui/Switch/Switch";
import { Dialog } from "../../../web/src/components/ui/Dialog/Dialog";
import { Select } from "../../../web/src/components/ui/Select/Select";
import { InlineStatus } from "../../../web/src/components/ui/InlineStatus/InlineStatus";
import { useLocale } from "../../../web/src/lib/i18n/LocaleProvider";

const initial = () => [
  { id: "family", semantic: true, face: false, completed: 1240, eligible: 2490, running: true },
  { id: "travel", semantic: false, face: false, completed: 0, eligible: 2650, running: false },
];

export function App() {
  const { locale } = useLocale();
  const t = (zh, en) => locale === "en" ? en : zh;
  const [libraries, setLibraries] = useState(initial);
  const [selected, setSelected] = useState("family");
  const [dialog, setDialog] = useState(null);
  const [service, setService] = useState("connected");
  const [notice, setNotice] = useState("");
  const library = libraries.find(item => item.id === selected);
  const connected = service === "connected";
  const libraryName = item => item.id === "family" ? t("家庭影像", "Family photos") : t("旅行记录", "Travel journals");
  function update(id, change) { setLibraries(current => current.map(item => item.id === id ? { ...item, ...change } : item)); }
  function toggle(kind, value) {
    update(selected, { [kind]: value, ...(kind === "semantic" && !value ? { running: false } : {}) });
    setNotice(t("仅更新了示例设置，不会启动真实分析。", "Only example settings changed. No real analysis was started."));
  }
  function status(item) {
    if (!item.semantic) return t("未开启分析", "Analysis disabled");
    if (!connected) return t("服务不可用 · 保留已有索引", "Service unavailable · index preserved");
    if (item.running) return t("分析中 · 示例", "Analyzing · example");
    return t("待继续分析", "Ready to continue");
  }
  const capabilities = [
    { key: "semantic", title: t("智能搜索", "Smart search"), text: t("用自然语言查找照片。", "Find photos using natural language."), model: "CLIP ViT-B-32", Icon: MagnifyingGlass },
    { key: "face", title: t("人物分析", "People analysis"), text: t("按媒体库单独开启。", "Enable separately for each library."), model: "buffalo_l", Icon: Users },
  ];
  return <ManagementShell active="ai" identity={t("管理员", "Administrator")} accountHref="/" generalHref="/" homeHref="/" librariesHref="/" searchHref="/" storageHref="/" aiHref="/" logsHref="/" aboutHref="/">
    <article className="intelligence">
      <header className="page-heading">
        <div><span className="preview-label">{t("设计预览 · 示例状态", "Design preview · example state")}</span><h1>{t("智能功能", "Intelligence")}</h1><p>{t("在本地分析照片，模型由部署配置管理。", "Analyze photos locally. Models are managed by deployment configuration.")}</p></div>
        <div className="heading-actions"><div className={`connection ${connected ? "connected" : "disconnected"}`}><CheckCircle size={15} weight="fill" aria-hidden="true" />{connected ? t("AI 服务已连接 / 本地处理", "AI connected / local processing") : service === "offline" ? t("AI 服务暂时不可用", "AI service unavailable") : t("尚未配置 AI 服务", "AI service not configured")}</div><small>{t("示例检查时间 2026-09-14 10:24", "Example check: 2026-09-14 10:24")}</small><Button variant="primary" onClick={() => setDialog("progress")}><span className="button-content"><Play size={18} weight="fill" aria-hidden="true" />{t("查看分析进度", "View analysis progress")}</span></Button></div>
      </header>
      {!connected && <InlineStatus tone="warning">{t("普通浏览不受影响，已有索引将保留。请检查部署配置后重试。", "Browsing is unaffected and existing indexes are preserved. Check your deployment configuration.")}</InlineStatus>}
      <section className="section" aria-labelledby="capabilities-heading">
        <div className="section-heading"><div><h2 id="capabilities-heading">{t("AI 能力", "AI capabilities")}</h2><p>{t("在所选媒体库启用后，照片将在本地处理。", "Enable capabilities for the selected library. Photos are processed locally.")}</p></div><label className="library-select"><span>{t("当前媒体库", "Current library")}</span><Select aria-label={t("当前媒体库", "Current library")} value={selected} onChange={event => setSelected(event.target.value)}>{libraries.map(item => <option key={item.id} value={item.id}>{libraryName(item)}</option>)}</Select></label></div>
        <div className="capabilities">{capabilities.map(({ key, title, text, model, Icon }) => <div className="capability-row" key={key}>
          <span className="capability-icon"><Icon aria-hidden="true" size={28} /></span><div className="capability-description"><h3>{title}</h3><p>{text}</p></div>
          <div className="model-label"><span>{t("使用的模型", "Configured model")}</span><strong>{model}</strong><small>{t("来自 Docker Compose", "From Docker Compose")}</small></div>
          <label className="switch-label"><Switch aria-label={`${libraryName(library)} · ${title}`} checked={library[key]} disabled={!connected} onChange={event => toggle(key, event.target.checked)} /><span>{library[key] ? t("已开启", "Enabled") : t("未开启", "Disabled")}</span></label>
        </div>)}</div>
        {notice && <div className="notice"><InlineStatus onDismiss={() => setNotice("")}>{notice}</InlineStatus></div>}
      </section>
      <section className="section" aria-labelledby="libraries-heading"><div className="section-heading"><div><h2 id="libraries-heading">{t("媒体库分析", "Library analysis")}</h2><p>{t("选择要分析的媒体库。已有分析结果不会因关闭开关而删除。", "Choose a library to analyze. Turning a capability off does not delete existing results.")}</p></div></div>
        <div className="library-list">{libraries.map(item => <button className="library-row" key={item.id} onClick={() => { setSelected(item.id); setDialog("library"); }} aria-label={`${t("管理", "Manage")} ${libraryName(item)}`}><span className="capability-icon"><Folder size={28} aria-hidden="true" /></span><span className="library-name"><strong>{libraryName(item)}</strong><span>{item.completed ? `${item.completed.toLocaleString()} / ${item.eligible.toLocaleString()} ${t("张已分析", "analyzed")}` : status(item)}</span></span><progress aria-label={`${libraryName(item)} ${t("分析进度", "analysis progress")}`} max={item.eligible} value={item.completed} /><span className="percentage">{Math.round(item.completed / item.eligible * 100)}%</span><CaretRight size={20} aria-hidden="true" /></button>)}</div>
      </section>
      <details className="deployment"><summary>{t("部署详情", "Deployment details")}<CaretDown aria-hidden="true" size={18} /></summary><div className="deployment-content"><dl><div><dt>{t("运行方式", "Runtime")}</dt><dd>{t("主应用 + 独立 AI 容器", "Main app + independent AI container")}</dd></div><div><dt>{t("模型配置", "Model configuration")}</dt><dd>Docker Compose</dd></div><div><dt>{t("模型修改", "Change models")}</dt><dd>{t("修改部署配置后重启 AI 服务；界面不提供模型商店。", "Update deployment configuration and restart AI. There is no model store.")}</dd></div></dl><InlineStatus>{t("主应用尚未接入新 AI 服务。本页不读取 Docker、模型或媒体文件，所有状态均为交互演示。", "The main app is not integrated with the new AI service yet. This page reads no Docker state, models or media; all states are demonstrations.")}</InlineStatus><label className="demo-scenario">{t("演示场景（仅原型）", "Demo scenario (prototype only)")}<Select value={service} onChange={event => setService(event.target.value)}><option value="connected">{t("已连接", "Connected")}</option><option value="offline">{t("服务不可用", "Unavailable")}</option><option value="unconfigured">{t("尚未配置", "Not configured")}</option></Select></label></div></details>
      <footer className="privacy"><span><LockSimple size={17} aria-hidden="true" />{t("原始文件保持只读。分析在本地完成，不修改原始媒体。", "Originals stay read-only. Analysis runs locally without modifying media.")}</span><span><Info size={17} aria-hidden="true" />{t("示例数据，仅用于界面展示", "Example data for UI preview only")}</span></footer>
    </article>
    <Dialog open={dialog !== null} onOpenChange={open => { if (!open) setDialog(null); }} title={dialog === "progress" ? t("分析进度", "Analysis progress") : libraryName(library)} description={t("交互演示，不会创建真实任务或修改原始文件。", "Interactive demo: no real jobs or media changes.")}>
      {dialog === "library" ? <div className="library-dialog"><p>{status(library)}</p><div className="dialog-switch">{capabilities.map(item => <label key={item.key}><span>{item.title}</span><Switch checked={library[item.key]} disabled={!connected} onChange={e => toggle(item.key, e.target.checked)} aria-label={`${t("媒体库", "Library")} ${item.title}`} /></label>)}</div><Button variant="primary" onClick={() => setDialog("progress")}>{t("查看分析进度", "View analysis progress")}</Button></div> : <div className="task-list">{libraries.map(item => <div className="task-row" key={item.id}><div><h3>{libraryName(item)}</h3><p>{status(item)}</p></div><progress max={item.eligible} value={item.completed} aria-label={libraryName(item)} /><p>{item.completed.toLocaleString()} / {item.eligible.toLocaleString()}</p><Button disabled={!connected || !item.semantic} onClick={() => update(item.id, { running: !item.running })}>{item.running ? t("停止示例分析", "Stop example analysis") : t("分析剩余照片（演示）", "Analyze remaining (demo)")}</Button></div>)}</div>}
    </Dialog>
  </ManagementShell>;
}
