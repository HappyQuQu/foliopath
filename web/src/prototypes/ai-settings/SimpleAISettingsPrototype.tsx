// Disposable proposal. Imported only by its Storybook story and tests, never by product routes.
import { ArrowRight, CaretDown, CheckCircle, Clock, GearSix, LockSimple, MagnifyingGlass, Tag, Users } from "@phosphor-icons/react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { ManagementShell } from "../../components/patterns/ManagementShell/ManagementShell";
import { Button, Dialog, InlineStatus, Select, Switch } from "../../components/ui";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import styles from "./SimpleAISettingsPrototype.module.css";

type LibraryKey = "family" | "travel" | "archive";
type Feature = "search" | "people";
type LibraryState = { search: boolean; people: boolean; autoTags: boolean; total: number; completed: number; peopleCompleted: number; offline: boolean; queued: boolean; peopleQueued: boolean };
type Props = {
  // Older disposable stories can still open the settings view through this value.
  initialView?: "settings" | "tags";
  initialLibrary?: LibraryKey;
  onOpenLibrary?: () => void;
};
const initialLibraries: Record<LibraryKey, LibraryState> = {
  family: { search: true, people: true, autoTags: false, total: 2480, completed: 1860, peopleCompleted: 420, offline: false, queued: false, peopleQueued: false },
  travel: { search: true, people: false, autoTags: false, total: 640, completed: 640, peopleCompleted: 0, offline: false, queued: false, peopleQueued: false },
  archive: { search: true, people: true, autoTags: false, total: 1200, completed: 900, peopleCompleted: 300, offline: true, queued: false, peopleQueued: false },
};

export function SimpleAISettingsPrototype({ initialLibrary = "family", onOpenLibrary }: Props) {
  const { locale } = useLocale();
  const en = locale === "en";
  const copy = (zh: string, english: string) => en ? english : zh;
  const [libraryId, setLibraryId] = useState<LibraryKey>(initialLibrary);
  const [libraries, setLibraries] = useState(initialLibraries);
  const [notice, setNotice] = useState("");
  const [progressOpen, setProgressOpen] = useState(false);
  const [confirm, setConfirm] = useState<"rebuild" | "clear" | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus({ preventScroll: true }); window.scrollTo(0, 0); }, []);
  const library = libraries[libraryId];
  const libraryName = (id: LibraryKey) => id === "family" ? copy("家庭影像", "Family photos") : id === "travel" ? copy("旅行记录", "Travel") : copy("离线归档", "Offline archive");
  const name = libraryName(libraryId);
  const updateLibrary = (patch: Partial<LibraryState>) => setLibraries(current => ({ ...current, [libraryId]: { ...current[libraryId], ...patch } }));
  function toggle(feature: Feature, enabled: boolean) {
    updateLibrary({ [feature]: enabled, ...(enabled ? feature === "search" ? { queued: true } : { peopleQueued: true } : {}) });
    setNotice(enabled ? copy(`演示：已为「${name}」开启功能并提交补充分析；不代表分析完成。`, `Demo: enabled for ${name} and queued missing analysis; not completed.`) : copy(`演示：已关闭「${name}」的功能，保留已有分析结果和人工整理。`, `Demo: disabled for ${name}; existing results and curation are retained.`));
  }
  function toggleAutoTags(enabled: boolean) {
    updateLibrary({ autoTags: enabled });
    setNotice(copy(`提案演示：已${enabled ? "开启" : "关闭"}「${name}」的自动标签意愿。尚无后端契约，未执行识别或写入标签。`, `Proposal demo: automatic tags ${enabled ? "enabled" : "disabled"} for ${name} in memory only. No backend contract, recognition, or tag writes.`));
  }
  function switchLibrary(value: LibraryKey) { setLibraryId(value); setNotice(""); setConfirm(null); }
  const completed = (feature: Feature) => feature === "search" ? library.completed : library.peopleCompleted;
  const stateLabel = (feature: Feature) => !library[feature] ? copy("未开启", "Off") : library.offline ? copy("媒体库离线", "Library offline") : (feature === "search" ? library.queued : library.peopleQueued) ? copy("等待处理", "Queued") : completed(feature) === 0 ? copy("尚未分析", "Not analyzed") : completed(feature) < library.total ? copy("分析中", "Analyzing") : copy("已就绪", "Ready");
  const tagAction = onOpenLibrary
    ? <Button variant="quiet" size="small" onClick={() => onOpenLibrary()}>{copy("查看标签", "View tags")} <ArrowRight aria-hidden size={16} /></Button>
    : <a className={styles.tagLink} href="/iframe.html?id=prototypes-intelligence-photo-workspace--tags&viewMode=story">{copy("查看标签", "View tags")} <ArrowRight aria-hidden size={16} /></a>;

  return <ManagementShell active="ai" accountHref="#prototype-only" generalHref="#prototype-only" homeHref="#prototype-only" identity={copy("原型预览", "Prototype")} librariesHref="#prototype-only" logsHref="#prototype-only" aboutHref="#prototype-only" aiHref="/settings/ai" peopleHref="#prototype-only" searchHref="/settings/ai" storageHref="#prototype-only">
    <div className={styles.page}>
      <header className={styles.header}>
        <div><div className={styles.eyebrow}>{copy("管理设置", "SETTINGS")} <span>{copy("交互原型 · 演示数据", "Prototype · sample data")}</span></div>
          <h1 ref={heading} tabIndex={-1}>{copy("智能分析", "Intelligent analysis")}</h1>
          <p>{copy("让照片更好找，让人物和标签更好整理。", "Find photos more easily. Keep people and tags organized.")}</p>
        </div>
        <Button variant="secondary" onClick={() => setProgressOpen(true)}><Clock aria-hidden size={17} /> {copy("处理进度", "Processing status")}</Button>
      </header>

      <div className={styles.scope}>
        <label htmlFor="prototype-library">{copy("当前媒体库", "Current library")}<Select aria-label={copy("当前媒体库", "Current library")} id="prototype-library" value={libraryId} onChange={event => switchLibrary(event.target.value as LibraryKey)}>{Object.keys(libraries).map(id => <option key={id} value={id}>{libraryName(id as LibraryKey)}</option>)}</Select></label>
        <span>{copy("开关仅影响所选媒体库", "Switches apply to this library only")}</span>
      </div>
      {library.offline && <InlineStatus tone="warning">{copy("媒体库离线，暂时不能更改或运行分析；已有结果仍保留。", "This library is offline. Analysis changes are unavailable; existing results are retained.")}</InlineStatus>}
      {notice && <InlineStatus onDismiss={() => setNotice("")}>{notice}</InlineStatus>}

      <section className={styles.panel} aria-label={copy("分析功能", "Analysis features")}>
        <FeatureRow icon={<MagnifyingGlass size={23} />} title={copy("智能搜索", "Smart search")} description={copy("用描述查找照片，例如“海边的日落”。", "Find photos with descriptions, such as “sunset by the sea”.")} status={stateLabel("search")}>
          <Switch aria-label={copy(`开启${name}的智能搜索`, `Enable smart search for ${name}`)} checked={library.search} disabled={library.offline} onChange={event => toggle("search", event.target.checked)} />
        </FeatureRow>
        <FeatureRow icon={<Users size={23} />} title={copy("人物识别", "People recognition")} description={copy("发现相似人脸，由你确认分组和命名。", "Find similar faces for you to confirm and name.")} status={stateLabel("people")}>
          <Switch aria-label={copy(`开启${name}的人物识别`, `Enable people recognition for ${name}`)} checked={library.people} disabled={library.offline} onChange={event => toggle("people", event.target.checked)} />
        </FeatureRow>
        <FeatureRow icon={<Tag size={23} />} title={copy("自动标签", "Automatic tags")} description={copy("无需预选标签。识别后自动添加到照片，打开照片可查看、手动增删。", "No tags to preselect. Add tags after recognition, then view, add, or remove them in the photo viewer.")} status={copy("提案 · 未接入", "Proposal · not connected")} action={<><p className={styles.proposalHint}>{copy("仅交互提案：尚无后端契约，此开关不会运行分析或修改照片标签。", "Interaction proposal only: no backend contract. This switch does not run analysis or change photo tags.")}</p>{tagAction}</>}>
          <Switch aria-label={copy(`开启${name}的自动标签`, `Enable automatic tags for ${name}`)} checked={library.autoTags} disabled={library.offline} onChange={event => toggleAutoTags(event.target.checked)} />
        </FeatureRow>
      </section>
      <p className={styles.footnote}>{copy("智能搜索与人物识别开启后补充分析；关闭不会清除已有结果。自动标签仅演示开关。", "Enabling search or people recognition queues missing analysis. Disabling retains existing results. Automatic tags is a switch demo only.")}</p>
      <details className={styles.advanced}>
        <summary><GearSix aria-hidden size={18} /><span>{copy("高级维护", "Advanced maintenance")}</span><span className={styles.summaryHint}>{copy("模型信息、重新分析与清理", "Models, rebuild and cleanup")}</span><CaretDown aria-hidden size={16} /></summary>
        <div className={styles.advancedBody}>
          <div className={styles.modelInfo}><h2>{copy("共享 AI 服务", "Shared AI service")}</h2><span>{copy("由 Docker Compose 管理 · 只读", "Managed by Docker Compose · read-only")}</span></div>
          <dl className={styles.modelList}><div><dt>{copy("搜索模型", "Search model")}</dt><dd>ViT-B-32__openai</dd></div><div><dt>{copy("人物模型", "People model")}</dt><dd>buffalo_l</dd></div></dl>
          <div className={styles.maintenance}><div><h2>{copy("重新分析", "Rebuild analysis")}</h2><p>{copy(`重新处理「${name}」的图片搜索索引。`, `Rebuild the image search index for ${name}.`)}</p></div><Button disabled={!library.search || library.offline} onClick={() => setConfirm("rebuild")}>{copy("重新分析", "Rebuild")}</Button></div>
          <div className={styles.maintenance}><div><h2>{copy("清理搜索索引", "Clear search index")}</h2><p>{copy("只移除可重建索引，不删除原始照片、人物命名或人工标签。", "Remove rebuildable search data only. Photos, named people and manual tags are kept.")}</p></div><Button disabled={library.offline} onClick={() => setConfirm("clear")}>{copy("清理索引", "Clear index")}</Button></div>
        </div>
      </details>

      <footer className={styles.footer}><span><CheckCircle aria-hidden size={16} />{copy("AI 服务已连接（演示）", "AI service connected (demo)")}</span><span><LockSimple aria-hidden size={15} />{copy("模型由 Compose 管理，原始媒体保持只读", "Models managed by Compose. Original media is read-only.")}</span></footer>
      <aside className={styles.prototypeNote} id="prototype-only">{copy("原型预览，不保存真实设置。切库、开关和维护仅演示交互；自动标签尚无后端契约，查看标签进入独立演示。其他导航不在本次范围内。", "Prototype only: library switches and maintenance use in-memory sample data. Automatic tags has no backend contract; View tags opens a separate demo. Other navigation is outside this preview.")}</aside>
    </div>

    <Dialog open={progressOpen} onOpenChange={setProgressOpen} title={copy(`${name} · 处理进度`, `${name} · Processing status`)} description={copy("演示数据，只展示当前媒体库。", "Sample data for the selected library only.")} actions={<Button onClick={() => setProgressOpen(false)}>{copy("关闭", "Close")}</Button>}>
      <div className={styles.progressBody}>{(["search", "people"] as Feature[]).map(feature => <div className={styles.progressItem} key={feature}><div><strong>{feature === "search" ? copy("智能搜索", "Smart search") : copy("人物识别", "People recognition")}</strong><span>{stateLabel(feature)}</span></div><progress aria-label={feature === "search" ? copy("智能搜索处理进度", "Smart search progress") : copy("人物识别处理进度", "People progress")} max={library.total} value={completed(feature)} /><span>{copy(`已处理 ${completed(feature).toLocaleString()} / ${library.total.toLocaleString()} 张`, `${completed(feature).toLocaleString()} / ${library.total.toLocaleString()} processed`)}</span></div>)}<p className={styles.proposalHint}>{copy("自动标签：仅交互提案，未接入分析，没有处理进度。", "Automatic tags: interaction proposal only. No analysis integration or processing progress.")}</p></div>
    </Dialog>
    <Dialog open={confirm !== null} onOpenChange={open => { if (!open) setConfirm(null); }} title={confirm === "clear" ? copy(`清理「${name}」的搜索索引？`, `Clear the search index for ${name}?`) : copy(`重新分析「${name}」？`, `Rebuild analysis for ${name}?`)} description={copy("仅影响当前媒体库的搜索索引，保留原始照片和人工整理。", "Affects only this library’s search index. Originals and manual curation are preserved.")} actions={<><Button onClick={() => setConfirm(null)}>{copy("取消", "Cancel")}</Button><Button variant="primary" onClick={() => { if (confirm === "clear") updateLibrary({ completed: 0, queued: false }); else updateLibrary({ queued: true }); setNotice(copy("演示操作已确认，没有修改真实索引或文件。", "Demo action confirmed. No real index or files were changed.")); setConfirm(null); }}>{copy("确认演示操作", "Confirm demo action")}</Button></>}><p>{copy("此原型不会执行实际清理或重建。", "This prototype does not perform real deletion or rebuilding.")}</p></Dialog>
  </ManagementShell>;
}

function FeatureRow({ icon, title, description, status, action, children }: { icon: ReactNode; title: string; description: string; status?: string; action?: ReactNode; children: ReactNode }) {
  return <div className={styles.featureRow}><span className={styles.featureIcon} aria-hidden>{icon}</span><div className={styles.featureText}><h2>{title}</h2><p>{description}</p>{action && <div className={styles.featureAction}>{action}</div>}</div><div className={styles.control}>{status && <span className={styles.status}>{status}</span>}{children}</div></div>;
}
