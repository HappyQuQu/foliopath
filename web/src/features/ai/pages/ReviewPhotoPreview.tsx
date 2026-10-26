import { ImagesSquare } from "@phosphor-icons/react";
import { useState } from "react";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import styles from "./ReviewPhotoPreview.module.css";

// Shared full-photo preview for people and groups. Never a face crop or identity assertion.
export function ReviewPhotoPreview({ assetIds }: { assetIds: string[] }) {
  const ids = [...new Set(assetIds)].slice(0, 4);
  return <PreviewFrame key={ids.join(":")} ids={ids} />;
}
function PreviewFrame({ ids }: { ids: string[] }) {
  const [failed, setFailed] = useState<string[]>([]); const { t } = useLocale();
  if (ids.length === 0 || ids.every(id => failed.includes(id))) return <span className={styles.placeholder}><ImagesSquare size={32} aria-hidden="true" /><span>{t("peopleDirectory.noPreview")}</span></span>;
  return <span aria-hidden="true" className={`${styles.preview} ${styles[`count${ids.length}`]}`}>
    {ids.map(id => failed.includes(id) ? <span key={id} className={styles.failed}><ImagesSquare size={24} /></span> : <img key={id} alt="" loading="lazy" src={`/api/v1/assets/${encodeURIComponent(id)}/thumbnail?variant=grid`} onError={() => setFailed(value => value.includes(id) ? value : [...value, id])} />)}
  </span>;
}
