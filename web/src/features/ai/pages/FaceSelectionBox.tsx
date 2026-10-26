import type { AssetFace } from "../../../lib/api/ai";
import styles from "./FaceSelectionBox.module.css";

// One spatial control for single-photo and named-person review. The accompanying
// text selector remains necessary for small, coarse regions and keyboard users.
export function FaceSelectionBox({ face, label, onSelect, selected, disabled, marker }: { face: Pick<AssetFace, "region">; label: string; onSelect: () => void; selected: boolean; disabled?: boolean; marker?: string }) {
  return <button type="button" className={styles.box} disabled={disabled} aria-label={label} aria-pressed={selected} onClick={onSelect} style={{ left: `${face.region.xPercent}%`, top: `${face.region.yPercent}%`, width: `${face.region.widthPercent}%`, height: `${face.region.heightPercent}%` }}>{marker && <span className={styles.marker} aria-hidden="true">{marker}</span>}</button>;
}
