import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import type { MediaCollectionSelection } from "../../components/patterns/MediaCollection/MediaCollection";
import type { Asset } from "../../lib/api/catalog";
import { Button } from "../../components/ui";
import type { DeletionTarget } from "../../lib/api/deletion";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { paths } from "../../routes/paths";
import type { useDeletionFlow } from "./useDeletionFlow";
import styles from "./Deletion.module.css";

export function useDeletionSelection({ flow, scope, loadedTargets, assets, onEnter, onAccepted }: {
  flow: ReturnType<typeof useDeletionFlow>;
  scope: string;
  loadedTargets: DeletionTarget[];
  assets: Pick<Asset, "id" | "archiveMember" | "sourceAvailability">[];
  onEnter: () => void;
  onAccepted: () => void;
}) {
  const { t } = useLocale();
  const unavailableAssets = useMemo(() => new Map(assets.filter(item => item.archiveMember || item.sourceAvailability !== "available").map(item => [item.id, t(item.archiveMember ? "deletion.block.archive_member" : item.sourceAvailability === "offline" ? "deletion.block.offline" : item.sourceAvailability === "missing" ? "deletion.block.missing" : "deletion.block.unreadable")])), [assets, t]);
  const [selecting, setSelecting] = useState(false);
  const [targets, setTargets] = useState<DeletionTarget[]>([]);
  useEffect(() => { setTargets([]); setSelecting(false); }, [scope]);
  const assetIds = useMemo(() => new Set(targets.filter(item => item.type === "asset").map(item => item.id)), [targets]);
  function toggle(type: DeletionTarget["type"], id: string) {
    setTargets(current => current.some(item => item.type === type && item.id === id) ? current.filter(item => item.type !== type || item.id !== id) : current.length < 200 ? [...current, { type, id }] : current);
  }
  const toolbar = <div className={styles.selection}>
    <Button size="small" disabled={!flow.canDelete && !selecting} onClick={() => { onEnter(); setSelecting(value => !value); setTargets([]); }}>{t(selecting ? "deletion.exitSelection" : "deletion.select")}</Button>
    {selecting && <><span role="status">{t("deletion.selected").replace("{count}", String(targets.length))}</span>
      <Button size="small" onClick={() => setTargets(loadedTargets.filter(item => item.type !== "asset" || !unavailableAssets.has(item.id)).slice(0, 200))}>{t("deletion.selectLoaded")}</Button>
      <Button size="small" onClick={() => setTargets([])} disabled={!targets.length}>{t("deletion.clearSelection")}</Button>
      <Button size="small" variant="danger" disabled={!targets.length || !flow.canDelete} onClick={() => flow.start(targets, () => { setSelecting(false); setTargets([]); onAccepted(); })}>{t("deletion.delete")}</Button>
      <span className={styles.muted}>{t("deletion.limit")}</span>
    </>}
    {!flow.canDelete && <Link to={paths.deletionSettings}>{t("deletion.settings")}</Link>}
  </div>;
  const mediaSelection: MediaCollectionSelection | undefined = selecting ? { ids: assetIds, label: (name: string) => t("deletion.selectItem").replace("{name}", name), onToggle: (id: string) => toggle("asset", id), disabledDescription: id => unavailableAssets.get(id) ?? (targets.length >= 200 && !assetIds.has(id) ? t("deletion.limit") : undefined), disabled: (id: string) => unavailableAssets.has(id) || (targets.length >= 200 && !assetIds.has(id)) } : undefined;
  return { selecting, targets, toggle, toolbar, mediaSelection };
}
