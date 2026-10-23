// Disposable interaction proposal. No real models, media, API calls or persistence.
import { ArrowLeft, ArrowRight, Images, MagnifyingGlass, Sparkle, Tag, Users, X } from "@phosphor-icons/react";
import { useEffect, useRef, useState } from "react";
import { AppShell } from "../../components/patterns/AppShell/AppShell";
import { Button, EmptyState, IconButton, InlineStatus, Input, Select } from "../../components/ui";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { PeopleWorkspacePrototype } from "./PeopleWorkspacePrototype";
import { SimpleAISettingsPrototype } from "./SimpleAISettingsPrototype";
import azure1 from "../../../../prototypes/apple-redesign/assets/ai-media/azure-01.png";
import azure2 from "../../../../prototypes/apple-redesign/assets/ai-media/azure-02.png";
import azure3 from "../../../../prototypes/apple-redesign/assets/ai-media/azure-03.png";
import forest1 from "../../../../prototypes/apple-redesign/assets/ai-media/forest-01.png";
import forest2 from "../../../../prototypes/apple-redesign/assets/ai-media/forest-02.png";
import forest3 from "../../../../prototypes/apple-redesign/assets/ai-media/forest-03.png";
import crimson1 from "../../../../prototypes/apple-redesign/assets/ai-media/crimson-01.png";
import crimson2 from "../../../../prototypes/apple-redesign/assets/ai-media/crimson-02.png";
import crimson3 from "../../../../prototypes/apple-redesign/assets/ai-media/crimson-03.png";
import styles from "./PhotoWorkspacePrototype.module.css";

type View = "tags" | "search" | "people" | "settings";
type Source = "file" | "path" | "semantic";
type Photo = { id: string; src: string; name: string; path: string; ai: string[]; manual: string[]; excluded: string[] };
const dictionary: Record<string, string> = { 人像: "Portrait", 室内: "Indoors", 服装: "Costume", 蓝色: "Blue", 绿色: "Green", 红色: "Red", 植物: "Plants", 精选: "Favorite set" };
const photos: Photo[] = [azure1, azure2, azure3, forest1, forest2, forest3, crimson1, crimson2, crimson3].map((src, i) => ({
  id: `sample-${i + 1}`, src,
  name: i === 3 ? "绿色_温室_01.png" : `IMG_${4001 + i}.png`,
  path: i === 4 ? "绿色主题 / 温室" : i < 3 ? "室内 / 蓝色" : i < 6 ? "温室" : "室内 / 红色",
  ai: ["人像", "服装", ...(i < 3 ? ["室内", "蓝色"] : i < 6 ? ["绿色", "植物"] : ["室内", "红色"])],
  manual: i === 0 ? ["精选"] : [], excluded: [],
}));
export function visibleTags(photo: Photo) { return [...new Set([...photo.ai.filter(tag => !photo.excluded.includes(tag)), ...photo.manual])]; }
export function countTags(items: Photo[]) {
  const counts = new Map<string, number>();
  for (const photo of items) for (const tag of visibleTags(photo)) counts.set(tag, (counts.get(tag) ?? 0) + 1);
  return [...counts].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0], "zh-CN"));
}
function normalized(value: string) { return value.normalize("NFKC").trim().toLocaleLowerCase(); }
export function PhotoWorkspacePrototype({ initialView = "tags" }: { initialView?: View }) {
  const { locale } = useLocale(); const en = locale === "en";
  const copy = (zh: string, english: string) => en ? english : zh;
  const label = (tag: string) => en ? dictionary[tag] ?? tag : tag;
  const [view, setView] = useState<View>(initialView);
  const [libraryId, setLibraryId] = useState("sample");
  const [allPhotos, setAllPhotos] = useState<Record<string, Photo[]>>({ sample: photos, trip: photos.slice(3, 6).map(p => ({ ...p, manual: [], excluded: [] })) });
  const [selectedTag, setSelectedTag] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [tagInput, setTagInput] = useState("");
  const [tagSearch, setTagSearch] = useState("");
  const [input, setInput] = useState("绿色"); const [query, setQuery] = useState("绿色");
  const [source, setSource] = useState<"all" | "file" | "semantic">("all");
  const [notice, setNotice] = useState("");
  const titleRef = useRef<HTMLHeadingElement>(null);
  const photosInLibrary = allPhotos[libraryId]!;
  const counts = countTags(photosInLibrary);
  const selected = photosInLibrary.find(p => p.id === selectedId);
  const title = view === "search" ? copy("搜索照片", "Search photos") : view === "people" ? copy("人物", "People") : copy("标签", "Tags");
  useEffect(() => { titleRef.current?.focus({ preventScroll: true }); window.scrollTo(0, 0); }, [view, selectedId]);
  function navigate(next: View) { setView(next); setSelectedId(null); setNotice(""); }
  function updateSelected(transform: (p: Photo) => Photo) {
    setAllPhotos(current => ({ ...current, [libraryId]: current[libraryId]!.map(p => p.id === selectedId ? transform(p) : p) }));
  }
  function removeTag(tag: string) {
    titleRef.current?.focus({ preventScroll: true });
    updateSelected(p => ({ ...p, manual: p.manual.filter(t => t !== tag), excluded: [...new Set([...p.excluded, tag])] }));
    setNotice(copy(`已移除「${label(tag)}」。再次分析也不会自动加回这张图片（演示）。`, `Removed “${label(tag)}”. Re-analysis will not add it back to this photo (demo).`));
  }
  function addTag() {
    const value = tagInput.normalize("NFKC").trim(); if (!selected || !value) return;
    const existing = [...counts.map(([tag]) => tag), ...Object.keys(dictionary)].find(tag => normalized(tag) === normalized(value) || normalized(dictionary[tag] ?? "") === normalized(value));
    const tag = existing ?? value;
    if (visibleTags(selected).includes(tag)) { setNotice(copy("这张图片已有该标签，不会重复添加。", "This photo already has that tag.")); return; }
    updateSelected(p => ({ ...p, manual: [...p.manual, tag], excluded: p.excluded.filter(t => t !== tag) }));
    setTagInput(""); setNotice(copy(`已手动添加「${label(tag)}」（演示）。`, `Added “${label(tag)}” manually (demo).`));
  }
  function matches(photo: Photo): Source[] {
    const q = normalized(query); if (!q) return [];
    const result: Source[] = [];
    if (normalized(photo.name).includes(q)) result.push("file");
    if (normalized(photo.path).includes(q)) result.push("path");
    // Explicit finite fixture matching, not a semantic algorithm or model output.
    if (photo.ai.some(tag => normalized(tag).includes(q) || normalized(dictionary[tag] ?? "").includes(q))) result.push("semantic");
    return result;
  }
  const sourceLabel = (s: Source) => s === "file" ? copy("文件名", "Filename") : s === "path" ? copy("库内路径", "Library path") : copy("画面语义", "Visual meaning");
  const displayed = view === "search" ? photosInLibrary.filter(p => {
    const hits = matches(p); return source === "all" ? hits.length > 0 : source === "file" ? hits.includes("file") || hits.includes("path") : hits.includes("semantic");
  }) : photosInLibrary.filter(p => !selectedTag || visibleTags(p).includes(selectedTag));
  const cloud = counts.filter(([tag]) => normalized(label(tag)).includes(normalized(tagSearch))).slice(0, 40);
  const maxCount = Math.max(1, ...counts.map(([, count]) => count));
  if (view === "settings") return <SimpleAISettingsPrototype onOpenLibrary={() => navigate("tags")} />;
  const sidebar = <div className={styles.sidebar}>
    <label className={styles.libraryLabel} htmlFor="workspace-library">{copy("媒体库", "Library")}</label>
    <Select id="workspace-library" disabled={view === "people"} aria-label={copy("媒体库", "Library")} value={libraryId} onChange={event => { setLibraryId(event.target.value); setSelectedId(null); setSelectedTag(null); setNotice(""); }}>{<option value="sample">{copy("样片集", "Sample collection")}</option>}<option value="trip">{copy("温室样片", "Greenhouse samples")}</option></Select>
    <p className={styles.libraryStatus}>{view === "people" ? copy("人物目录跨媒体库展示", "People directory spans libraries") : copy(`演示库 · ${photosInLibrary.length} 张图片`, `Demo library · ${photosInLibrary.length} photos`)}</p>
    <nav className={styles.nav} aria-label={copy("快速访问", "Quick access")}>
      <h2>{copy("快速访问", "Quick access")}</h2>
      <Button variant="quiet" onClick={() => { navigate("tags"); setSelectedTag(null); }}><Images aria-hidden size={19} />{copy("全部媒体", "All media")}</Button>
      <Button variant="quiet" aria-pressed={view === "people"} onClick={() => navigate("people")}><Users aria-hidden size={19} />{copy("人物", "People")}</Button>
      <Button variant="quiet" aria-pressed={view === "tags"} onClick={() => navigate("tags")}><Tag aria-hidden size={19} />{copy("标签", "Tags")}</Button>
      <Button variant="quiet" aria-pressed={view === "search"} onClick={() => navigate("search")}><MagnifyingGlass aria-hidden size={19} />{copy("搜索", "Search")}</Button>
    </nav>
    <section className={styles.cloudSection} aria-label={copy("标签云", "Tag cloud")}>
      <div className={styles.cloudHeading}><h2>{copy("全部标签", "All tags")}</h2><span>{counts.length}</span></div>
      <p>{copy("字越大，关联的图片越多", "Bigger tags have more photos")}</p>
      <Input aria-label={copy("查找标签", "Find a tag")} placeholder={copy("查找标签…", "Find a tag…")} value={tagSearch} onChange={event => setTagSearch(event.target.value)} />
      <div className={styles.cloud}>{cloud.map(([tag, count]) => <Button key={tag} variant="quiet" aria-label={`${label(tag)} · ${count} ${copy("张图片", "photos")}`} aria-pressed={view === "tags" && selectedTag === tag} className={`${styles.cloudTag} ${count / maxCount > 0.7 ? styles.frequent : count / maxCount > 0.3 ? styles.regular : styles.rare}`} onClick={() => { navigate("tags"); setSelectedTag(current => current === tag ? null : tag); }}><span>{label(tag)}</span><small>{count}</small></Button>)}</div>
      {cloud.length === 0 && <p>{copy("没有匹配的标签", "No matching tags")}</p>}
      <p className={styles.cloudNote}>{copy("当前库计数 · 同一图片只计一次", "This library · each photo counted once")}</p>
    </section>
    <Button className={styles.settingsLink} variant="quiet" onClick={() => navigate("settings")}><Sparkle aria-hidden size={18} />{copy("分析设置", "Analysis settings")}<ArrowRight aria-hidden size={15} /></Button>
  </div>;
  return <AppShell active="browse" identity={copy("交互原型", "Prototype")} homeHref="#prototype-note" searchHref="#prototype-note" settingsHref="#prototype-note" title={title} sidebarContent={sidebar} topbarContent={<><span>{title}</span><span className={styles.demoBadge}>{copy("交互提案 · 非真实分析", "Proposal · not real analysis")}</span></>}>
    <div className={styles.page}>
      <div hidden={view !== "people"}><PeopleWorkspacePrototype active={view === "people"} /></div>
      {view !== "people" && <>
        {selected ? <>
          <Button variant="quiet" className={styles.back} onClick={() => { setSelectedId(null); setNotice(""); }}><ArrowLeft aria-hidden size={17} />{copy("返回照片", "Back to photos")}</Button>
          <h1 className={styles.viewerTitle} ref={titleRef} tabIndex={-1}>{selected.name}</h1>
          <div className={styles.viewer}>
            <div className={styles.imageStage}><img src={selected.src} alt={copy("合成虚构人物样片", "Synthetic fictional portrait")} /></div>
            <section className={styles.inspector} aria-label={copy("图片标签", "Photo tags")}>
              <div className={styles.sectionHeading}><Tag aria-hidden size={20} /><h2>{copy("标签", "Tags")}</h2></div>
              <p>{copy("AI 自动添加，你可以随时增删。", "Added automatically by AI. You can edit them anytime.")}</p>
              <div className={styles.photoTags}>{visibleTags(selected).map(tag => <div className={styles.tagChip} key={tag}><span>{label(tag)}</span><small>{selected.manual.includes(tag) ? copy("手动", "Manual") : "AI"}</small><IconButton label={copy(`删除标签 ${label(tag)}`, `Remove tag ${label(tag)}`)} onClick={() => removeTag(tag)}><X aria-hidden size={14} /></IconButton></div>)}</div>
              {visibleTags(selected).length === 0 && <p>{copy("暂无标签，添加一个试试。", "No tags yet. Add one below.")}</p>}
              <form className={styles.addTag} onSubmit={event => { event.preventDefault(); addTag(); }}><label htmlFor="new-photo-tag">{copy("添加标签", "Add a tag")}</label><div><Input id="new-photo-tag" maxLength={40} placeholder={copy("输入标签名称", "Type a tag name")} value={tagInput} onChange={event => setTagInput(event.target.value)} /><Button type="submit" disabled={!tagInput.trim()}>{copy("添加", "Add")}</Button></div></form>
              <p className={styles.help}>{copy("手动添加的保留；删掉的不会被下次 AI 分析自动加回。", "Manual tags are kept. Removed tags stay excluded from future AI analysis.")}</p>
              <Button size="small" onClick={() => { updateSelected(p => ({ ...p, ai: photos.find(item => item.id === p.id)!.ai })); setNotice(copy("已模拟重新分析：保留手动标签，跳过你删除的标签。", "Simulated re-analysis: manual tags kept; removed tags remain excluded.")); }}>{copy("模拟重新分析", "Simulate re-analysis")}</Button>
              {notice && <InlineStatus>{notice}</InlineStatus>}
              {view === "search" && <section className={styles.matchDetails}><h3>{copy("为什么匹配", "Why it matched")}</h3>{matches(selected).map(s => <div key={s}><strong>{sourceLabel(s)}</strong><p>{s === "file" ? selected.name : s === "path" ? selected.path : copy(`「${query}」的语义相似匹配（固定演示，非实际模型解释）`, `Semantic match for “${query}” (fixed demo, not a model explanation)`)}</p></div>)}</section>}
              <p className={styles.help}>{copy("演示标签保存在本页内存中；不会修改原图。", "Demo tags live in memory only. Originals are unchanged.")}</p>
            </section>
          </div>
        </> : <>
          <header className={styles.header}><div><div className={styles.eyebrow}>{copy("照片整理", "PHOTO ORGANIZATION")}</div><h1 ref={titleRef} tabIndex={-1}>{view === "search" ? title : selectedTag ? label(selectedTag) : copy("用标签发现照片", "Find photos by tag")}</h1><p>{view === "search" ? copy("知道为什么找到它，不再猜测命中来源。", "See why each photo matched.") : copy("AI 自动标注，按内容浏览；打开照片即可调整标签。", "Automatic tags for browsing. Open any photo to edit its tags.")}</p></div><Button variant="quiet" onClick={() => navigate("settings")}>{copy("分析设置", "Analysis settings")}<ArrowRight aria-hidden size={16} /></Button></header>
          {view === "search" ? <section className={styles.searchBox} aria-label={copy("搜索条件", "Search criteria")}>
            <form onSubmit={event => { event.preventDefault(); setQuery(input.trim()); }}><Input aria-label={copy("搜索内容", "Search query")} value={input} placeholder={copy("文件名、画面描述…", "Filename or visual description…")} onChange={event => setInput(event.target.value)} /><Button variant="primary" type="submit"><MagnifyingGlass aria-hidden size={17} />{copy("搜索", "Search")}</Button></form>
            <div className={styles.sourceFilters} aria-label={copy("匹配来源筛选", "Match source filters")}>{(["all", "file", "semantic"] as const).map(s => <Button size="small" key={s} variant={source === s ? "primary" : "quiet"} aria-pressed={source === s} onClick={() => setSource(s)}>{s === "all" ? copy("全部来源", "All sources") : s === "file" ? copy("文件名 / 路径", "Filename / path") : copy("画面语义", "Visual meaning")}</Button>)}<Button size="small" variant="quiet" disabled>{copy("图片文字 OCR · 未接入", "Image text OCR · unavailable")}</Button></div>
            <p>{copy("来源与结果均为固定样例。正式混合搜索和 OCR 需要补充后端能力。", "Fixed sample results and sources. Mixed-source search and OCR still need backend support.")}</p>
          </section> : <div className={styles.tagToolbar}><span><Sparkle aria-hidden size={17} />{copy("AI 标签已自动加入样片 · 不需要逐项确认", "AI tags already applied to samples · no review queue")}</span>{selectedTag && <Button variant="quiet" size="small" onClick={() => setSelectedTag(null)}>{copy("清除标签筛选", "Clear tag filter")}<X aria-hidden size={14} /></Button>}</div>}
          <div className={styles.resultsHeading}><h2>{copy(`${displayed.length} 张图片`, `${displayed.length} photos`)}</h2><span>{view === "search" ? copy("合成样例 · 多种命中仅显示一次", "Synthetic samples · multiple hits deduplicated") : copy("合成样片 · 数量由当前标签计算", "Synthetic samples · counts reflect current tags")}</span></div>
          {displayed.length === 0 ? <EmptyState title={copy("没有匹配的图片", "No matching photos")} description={copy("换一个标签、关键词或来源试试。此原型只包含有限样片。", "Try another tag, query or source. This prototype has a finite sample set.")} /> : <div className={styles.gallery}>{displayed.map(photo => <article className={styles.card} key={photo.id}>
            <button className={styles.photoButton} type="button" aria-label={copy(`打开图片 ${photo.name}`, `Open photo ${photo.name}`)} onClick={() => { setSelectedId(photo.id); setTagInput(""); setNotice(""); }}><img src={photo.src} alt={copy("合成虚构人物样片", "Synthetic fictional portrait")} /><span>{photo.name}</span></button>
            <div className={styles.cardMeta}>{view === "search" ? <><div className={styles.badges}>{matches(photo).map(s => <span key={s} className={s === "semantic" ? styles.semanticBadge : styles.fileBadge}>{s === "semantic" && <Sparkle aria-hidden size={12} />}{sourceLabel(s)}</span>)}</div><p>{matches(photo).includes("path") ? photo.path : matches(photo).includes("file") ? copy("文件名包含搜索词", "Filename contains the query") : copy("画面相似匹配 · 演示", "Visual similarity · demo")}</p></> : <div className={styles.badges}>{visibleTags(photo).slice(0, 3).map(tag => <span key={tag}>{label(tag)}</span>)}{visibleTags(photo).length > 3 && <small>+{visibleTags(photo).length - 3}</small>}</div>}</div>
          </article>)}</div>}
        </>}
      </>}
      <footer className={styles.prototypeNote} id="prototype-note">{copy("历史原型归档：图片、中文标签和匹配来源都是演示数据，不访问真实媒体库。标签云、自动打标与人工保护请查看正式组件预览；人物整理、混合搜索与 OCR 仍按各自待办验收。", "Historical design archive: sample images, localized tags and match sources are synthetic; no real library access. See production component stories for the tag cloud, automatic tagging and manual protection. People, mixed search and OCR retain separate acceptance gates.")}</footer>
    </div>
  </AppShell>;
}
