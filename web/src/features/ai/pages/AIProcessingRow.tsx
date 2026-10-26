import { CaretDown } from "@phosphor-icons/react";
import { useId, useState, type ReactNode, type Ref } from "react";
import { Button } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import styles from "./AIProcessingRow.module.css";

// One presentation owner for the contract-backed processing capabilities.
export function AIProcessingRow({ title, description, icon, children, action, maintenance, headingRef, details }: {
  title: string;
  description: string;
  icon: ReactNode;
  children: ReactNode;
  action?: ReactNode;
  maintenance?: ReactNode;
  headingRef?: Ref<HTMLHeadingElement>;
  details?: ReactNode;
}) {
  const { t } = useLocale();
  const heading = useId();
  const content = useId();
  const [expanded, setExpanded] = useState(false);
  return <section className={styles.row} aria-labelledby={heading}>
    <header className={styles.identity}>
      <span className={styles.icon}>{icon}</span>
      <div><h2 id={heading} ref={headingRef} tabIndex={headingRef ? -1 : undefined}>{title}</h2><p>{description}</p></div>
    </header>
    <div className={styles.status}>{children}</div>
    <div className={styles.actions}>
      {action}
      {maintenance && <Button variant="quiet" aria-label={`${t("ai.tasks.more")}：${title}`} aria-expanded={expanded} aria-controls={content} onClick={() => setExpanded(!expanded)}>
        {t("ai.tasks.more")}<CaretDown aria-hidden="true" size={14} weight="bold" />
      </Button>}
    </div>
    {details}
    {maintenance && <div className={styles.maintenance} id={content} hidden={!expanded}>{maintenance}</div>}
  </section>;
}
