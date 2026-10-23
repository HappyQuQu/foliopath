// Disposable proposal: Storybook-only, bounded synthetic state and no API calls.
import { ArrowLeft, ArrowRight, UsersThree } from "@phosphor-icons/react";
import { useEffect, useRef, useState } from "react";

import azure01 from "../../../../prototypes/apple-redesign/assets/ai-media/azure-01.png";
import azure02 from "../../../../prototypes/apple-redesign/assets/ai-media/azure-02.png";
import azure03 from "../../../../prototypes/apple-redesign/assets/ai-media/azure-03.png";
import crimson01 from "../../../../prototypes/apple-redesign/assets/ai-media/crimson-01.png";
import crimson02 from "../../../../prototypes/apple-redesign/assets/ai-media/crimson-02.png";
import crimson03 from "../../../../prototypes/apple-redesign/assets/ai-media/crimson-03.png";
import forest01 from "../../../../prototypes/apple-redesign/assets/ai-media/forest-01.png";
import forest02 from "../../../../prototypes/apple-redesign/assets/ai-media/forest-02.png";
import forest03 from "../../../../prototypes/apple-redesign/assets/ai-media/forest-03.png";
import { Button, Dialog, EmptyState, InlineStatus, Input, Select } from "../../components/ui";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import styles from "./PeopleWorkspacePrototype.module.css";

const photos = [
  { id: "azure-01", src: azure01 }, { id: "azure-02", src: azure02 }, { id: "azure-03", src: azure03 },
  { id: "crimson-01", src: crimson01 }, { id: "crimson-02", src: crimson02 }, { id: "crimson-03", src: crimson03 },
  { id: "forest-01", src: forest01 }, { id: "forest-02", src: forest02 }, { id: "forest-03", src: forest03 },
];
type Person = { id: string; name: string; unnamedNumber: number; photoIds: string[] };
const initialPeople: Person[] = [
  { id: "azure", name: "林夏", unnamedNumber: 0, photoIds: ["azure-01", "azure-02", "crimson-03"] },
  { id: "azure-duplicate", name: "", unnamedNumber: 1, photoIds: ["azure-03"] },
  { id: "crimson", name: "明月", unnamedNumber: 0, photoIds: ["crimson-01", "crimson-02"] },
  { id: "forest", name: "", unnamedNumber: 2, photoIds: ["forest-01", "forest-02", "forest-03"] },
];
type Notice = { kind: "named" | "merged" | "removed"; count?: number } | null;

export function PeopleWorkspacePrototype({ active = true }: { active?: boolean }) {
  const { locale } = useLocale();
  const copy = (zh: string, en: string) => locale === "en" ? en : zh;
  const [people, setPeople] = useState(initialPeople);
  const [filter, setFilter] = useState<"all" | "unnamed">("all");
  const [personId, setPersonId] = useState<string | null>(null);
  const [nameDraft, setNameDraft] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  const [mergeOpen, setMergeOpen] = useState(false);
  const [mergeTarget, setMergeTarget] = useState("");
  const [removeOpen, setRemoveOpen] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const person = people.find(item => item.id === personId);
  const target = people.find(item => item.id === mergeTarget);
  const personName = (item: Person) => item.name || copy(`未命名人物 ${item.unnamedNumber}`, `Unnamed person ${item.unnamedNumber}`);
  const photoCount = (count: number) => copy(`${count} 张照片`, `${count} ${count === 1 ? "photo" : "photos"}`);
  const visiblePeople = people.filter(item => filter === "all" || !item.name);
  const unassignedPhotos = photos.filter(photo => !people.some(item => item.photoIds.includes(photo.id)));
  const displayPhotos = photos.filter(photo => person?.photoIds.includes(photo.id));
  const normalizedName = nameDraft.trim().normalize("NFC");

  useEffect(() => {
    if (!active) return;
    heading.current?.focus({ preventScroll: true });
    window.scrollTo(0, 0);
  }, [personId, active]);
  useEffect(() => {
    if (!active || !notice || notice.kind === "named") return;
    const frame = requestAnimationFrame(() => heading.current?.focus({ preventScroll: true }));
    return () => cancelAnimationFrame(frame);
  }, [notice, active]);

  function openPerson(next: Person) {
    setPersonId(next.id); setNameDraft(next.name); setSelected([]); setNotice(null);
  }

  function saveName() {
    if (!person || !normalizedName || normalizedName.length > 100 || normalizedName === person.name) return;
    setPeople(current => current.map(item => item.id === person.id ? { ...item, name: normalizedName } : item));
    setNameDraft(normalizedName); setNotice({ kind: "named" });
  }

  function mergePeople() {
    if (!person || !target || target.id === person.id) return;
    const sourceId = person.id;
    setPeople(current => current.filter(item => item.id !== sourceId).map(item => item.id === target.id
      ? { ...item, photoIds: [...new Set([...item.photoIds, ...person.photoIds])] } : item));
    setPersonId(target.id); setNameDraft(target.name); setSelected([]);
    setMergeOpen(false); setMergeTarget(""); setNotice({ kind: "merged" });
  }

  function removePhotos() {
    if (!person || !selected.length) return;
    const count = person.photoIds.filter(id => selected.includes(id)).length;
    setPeople(current => current.map(item => item.id === person.id ? { ...item, photoIds: item.photoIds.filter(id => !selected.includes(id)) } : item));
    setSelected([]); setRemoveOpen(false); setNotice({ kind: "removed", count });
  }

  const noticeText = notice?.kind === "named" ? copy("名称已保存到本次演示。", "Name saved for this demo.")
    : notice?.kind === "merged" ? copy("演示人物已合并，所有照片仍保留。", "Demo people merged. All photos are preserved.")
    : copy(`已移出 ${notice?.count ?? 0} 张照片。照片仍保留在媒体库，只解除与此人物的关联。`, `Removed ${notice?.count ?? 0} photos from this person. Photos remain in the library; only the association is removed.`);

  return <div className={styles.page}>
    {person && <div className={styles.back}><Button variant="quiet" size="small" onClick={() => { setPersonId(null); setSelected([]); setNotice(null); }}><ArrowLeft size={16} aria-hidden="true" />{copy("返回人物", "Back to people")}</Button></div>}
    <header className={styles.header}>
      <div><p className={styles.eyebrow}>{copy("人物整理", "PEOPLE")}<span>{copy("合成原型数据", "Synthetic prototype data")}</span></p>
        <h1 ref={heading} tabIndex={-1}>{person ? personName(person) : copy("人物", "People")}</h1>
        <p>{person ? copy(`${photoCount(person.photoIds.length)} · 确认分组、命名，或移出不属于此人的照片。`, `${photoCount(person.photoIds.length)} · Name this person or remove incorrectly grouped photos.`) : copy("从照片找到熟悉的面孔，再由你确认与命名。", "Explore familiar faces, then confirm and name them.")}</p>
      </div>
      {person ? <Button disabled={people.length < 2} onClick={() => { setMergeTarget(""); setMergeOpen(true); }}>{copy("合并人物", "Merge people")}</Button>
        : <div className={styles.summary}><strong>{people.length}</strong><span>{copy("个人物分组", "people groups")}</span></div>}
    </header>
    <p className={styles.scope}>{copy("人物目录适用于全部媒体库；这里的分组、姓名和照片均为演示内容。", "The people directory spans all libraries. All groups, names and photos here are demo content.")}</p>
    {notice && <InlineStatus tone="success" onDismiss={() => setNotice(null)}>{noticeText}</InlineStatus>}

    {!person ? <>
      <div className={styles.toolbar}>
        <div className={styles.filters} role="group" aria-label={copy("人物筛选", "Filter people")}>
          <Button aria-pressed={filter === "all"} variant={filter === "all" ? "secondary" : "quiet"} onClick={() => setFilter("all")}>{copy("全部人物", "All people")}<span className={styles.count}>{people.length}</span></Button>
          <Button aria-pressed={filter === "unnamed"} variant={filter === "unnamed" ? "secondary" : "quiet"} onClick={() => setFilter("unnamed")}>{copy("未命名", "Unnamed")}<span className={styles.count}>{people.filter(item => !item.name).length}</span></Button>
        </div>
        <span className={styles.hint}>{copy("点击头像，查看并整理照片", "Open a person to review their photos")}</span>
      </div>
      {visiblePeople.length ? <div className={styles.peopleGrid}>{visiblePeople.map(item => {
        const avatar = photos.find(photo => photo.id === item.photoIds[0]);
        return <Button key={item.id} variant="quiet" className={styles.personCard} aria-label={copy(`查看${personName(item)}，${photoCount(item.photoIds.length)}`, `Open ${personName(item)}, ${photoCount(item.photoIds.length)}`)} onClick={() => openPerson(item)}>
          {avatar ? <img className={styles.avatar} src={avatar.src} alt={copy(`${personName(item)}的合成人物头像`, `Synthetic portrait for ${personName(item)}`)} /> : <span className={styles.avatarPlaceholder}><UsersThree size={40} aria-hidden="true" /></span>}
          <strong>{personName(item)}</strong><span className={styles.cardCount}>{photoCount(item.photoIds.length)}</span>
        </Button>;
      })}</div> : <EmptyState title={copy("没有未命名的人物", "No unnamed people")} description={copy("所有演示分组都已有名称。可以切换到全部人物继续整理。", "All demo groups are named. Switch to all people to continue organizing.")} />}
      {unassignedPhotos.length > 0 && <section className={styles.unassigned} aria-labelledby="prototype-unassigned-title"><h2 id="prototype-unassigned-title">{copy("已移出的照片", "Removed from people")} · {photoCount(unassignedPhotos.length)}</h2><p>{copy("这些照片没有删除，只是不再属于之前的人物分组。", "These photos were not deleted. They are no longer assigned to their previous person.")}</p><div className={styles.unassignedGrid}>{unassignedPhotos.map(photo => <figure key={photo.id}><img src={photo.src} alt={copy(`已移出的合成照片 ${photo.id}.png`, `Unassigned synthetic photo ${photo.id}.png`)} /><figcaption>{photo.id}.png</figcaption></figure>)}</div></section>}
    </> : <>
      <section className={styles.namePanel} aria-label={copy("人物名称", "Person name")}>
        <form onSubmit={event => { event.preventDefault(); saveName(); }}><label htmlFor="prototype-person-name">{copy("人物名称", "Person name")}<Input id="prototype-person-name" maxLength={100} placeholder={copy("为这个人物填写名称", "Give this person a name")} value={nameDraft} onChange={event => setNameDraft(event.currentTarget.value)} /></label><Button variant="primary" type="submit" disabled={!normalizedName || normalizedName === person.name}>{copy("保存名称", "Save name")}</Button></form>
        <span>{copy("同名不会自动合并；合并需要单独确认。", "Matching names do not merge people automatically.")}</span>
      </section>
      <div className={styles.photoToolbar}><div><h2>{copy("人物照片", "Photos of this person")}</h2><p>{copy("示例中包含需要人工核对的照片，不代表识别准确率。", "Some sample photos need review. This is not a recognition-quality result.")}</p></div>
        <div className={styles.selection}><span aria-live="polite">{copy(`已选 ${selected.length} 张`, `${selected.length} selected`)}</span><Button disabled={!selected.length} onClick={() => setRemoveOpen(true)}>{copy("移出人物", "Remove from person")}</Button></div>
      </div>
      {displayPhotos.length ? <div className={styles.photoGrid}>{displayPhotos.map(photo => <label key={photo.id} className={styles.photoCard} data-selected={selected.includes(photo.id) || undefined}><div className={styles.photoImage}><img src={photo.src} alt={copy(`合成成年人物照片 ${photo.id}.png`, `Synthetic adult portrait ${photo.id}.png`)} /><input type="checkbox" aria-label={copy(`选择 ${photo.id}.png`, `Select ${photo.id}.png`)} checked={selected.includes(photo.id)} onChange={event => setSelected(current => event.target.checked ? [...new Set([...current, photo.id])] : current.filter(id => id !== photo.id))} /></div><span>{photo.id}.png</span></label>)}</div>
        : <EmptyState title={copy("暂时没有关联照片", "No photos assigned")} description={copy("人物名称仍保留；移出的照片没有被删除。", "The person’s name is retained. Removed photos were not deleted.")} />}
    </>}

    <footer className={styles.footer}><p>{copy(`共 ${photos.length} 张 AI 合成的虚构成年人物照片。无真实识别、无 API 调用、无持久化；原始媒体不受影响。`, `${photos.length} AI-generated fictional adult portraits. No real recognition, API calls or persistent changes. Original media is unaffected.`)}</p><p>{copy("隐藏人物与自选封面需补充接口，未作为本原型的可用功能。", "Hiding people and choosing a cover require additional API contracts and are not available in this prototype.")}</p></footer>

    <Dialog open={mergeOpen && !!person} onOpenChange={setMergeOpen} title={copy("合并人物", "Merge people")} description={copy("确认属于同一个人后再合并，不会删除照片。", "Merge only after confirming these are the same person. No photos are deleted.")} actions={<><Button onClick={() => setMergeOpen(false)}>{copy("取消", "Cancel")}</Button><Button variant="primary" disabled={!person || !target || target.id === person.id} onClick={mergePeople}>{copy("确认合并", "Confirm merge")}</Button></>}>
      <label className={styles.mergeField}>{copy("合并到哪位人物", "Merge into") }<Select value={mergeTarget} onChange={event => setMergeTarget(event.currentTarget.value)}><option value="">{copy("选择要保留的人物", "Choose the person to keep")}</option>{people.filter(item => item.id !== person?.id).map(item => <option key={item.id} value={item.id}>{personName(item)} · {photoCount(item.photoIds.length)}</option>)}</Select></label>
      {person && target && <div className={styles.mergePreview}><strong>{personName(person)}</strong><ArrowRight size={20} aria-hidden="true" /><strong>{personName(target)}</strong><p>{copy(`保留「${personName(target)}」的名称，合计 ${new Set([...person.photoIds, ...target.photoIds]).size} 张照片；原人物条目将移除。`, `Keep the name “${personName(target)}” with ${new Set([...person.photoIds, ...target.photoIds]).size} photos. The source person entry will be removed.`)}</p></div>}
    </Dialog>
    <Dialog open={removeOpen && !!person} onOpenChange={setRemoveOpen} title={copy("移出选中的照片？", "Remove selected photos from this person?")} description={copy("只解除人物关联，不删除照片，不改动原始文件。", "Only remove the person association. Photos and original files are not deleted.")} actions={<><Button onClick={() => setRemoveOpen(false)}>{copy("取消", "Cancel")}</Button><Button variant="primary" disabled={!selected.length} onClick={removePhotos}>{copy("确认移出", "Confirm removal")}</Button></>}><p>{copy(`将从「${person ? personName(person) : ""}」中移出 ${selected.length} 张照片。`, `Remove ${selected.length} photos from ${person ? personName(person) : ""}.`)}</p></Dialog>
  </div>;
}
