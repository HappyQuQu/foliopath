import { ArrowRight, DownloadSimple, Images, MagnifyingGlass, UsersThree } from "@phosphor-icons/react";
import { Link } from "react-router-dom";
import { Button, InlineStatus } from "../../../components/ui";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import styles from "./AISettingsPage.module.css";

export function AIOverview({ runtimeAvailable, activeModel, modelCount, enabledLibraries, totalLibraries, partialLibraries, completed, eligible, onView }: {
  runtimeAvailable: boolean | undefined;
  activeModel: boolean;
  modelCount: number;
  enabledLibraries: number;
  totalLibraries: number;
  partialLibraries: boolean;
  completed: number;
  eligible: number;
  onView: (view: "libraries" | "models" | "faces" | "tasks") => void;
}) {
  const { locale, t } = useLocale();
  const number = new Intl.NumberFormat(locale);
  return <div className={styles.cardList}>
    <section className={styles.welcome}>
      <div><p className={styles.eyebrow}>{t("ai.overview.eyebrow")}</p><h2>{t("ai.overview.title")}</h2><p>{t("ai.overview.description")}</p></div>
      <Button variant="primary" onClick={() => onView(activeModel ? "libraries" : "models")}>
        {activeModel ? <Images aria-hidden="true" size={18} /> : <DownloadSimple aria-hidden="true" size={18} />}{t(activeModel ? "ai.overview.manageLibraries" : "ai.overview.getModels")}
      </Button>
    </section>
    <div className={styles.overviewStats}>
      <div><span>{t("ai.installedModels")}</span><strong>{number.format(modelCount)}</strong></div>
      <div><span>{t("ai.overview.enabledLibraries")}</span><strong>{number.format(enabledLibraries)} <small>/ {number.format(totalLibraries)}</small></strong></div>
      <div><span>{t("ai.overview.analyzed")}</span><strong>{number.format(completed)} <small>/ {number.format(eligible)}</small></strong></div>
    </div>
    {partialLibraries && <InlineStatus tone="info">{t("ai.overview.partialLibraries")}</InlineStatus>}
    {runtimeAvailable === false && <InlineStatus tone="warning">{t("ai.download.runtimeMissing")}</InlineStatus>}
    <section aria-labelledby="ai-features-title"><div className={styles.sectionHeading}><h2 id="ai-features-title">{t("ai.overview.features")}</h2><span>{t("ai.overview.local")}</span></div>
      <div className={styles.featureGrid}>
        <article className={styles.featureCard}><div className={styles.modelIcon}><MagnifyingGlass aria-hidden="true" size={24} /></div><h3>{t("ai.overview.searchTitle")}</h3><p>{t("ai.overview.searchDescription")}</p>
          <InlineStatus tone={activeModel ? "success" : "info"}>{t(activeModel ? "ai.modelActive" : "ai.overview.needsModel")}</InlineStatus>
          <Link className={styles.textLink} to={activeModel ? `${paths.search}?mode=semantic` : `${paths.aiSettings}?view=models`}>{t(activeModel ? "ai.overview.openSearch" : "ai.overview.getModels")}<ArrowRight aria-hidden="true" size={16} /></Link>
        </article>
        <article className={styles.featureCard}><div className={styles.modelIcon}><Images aria-hidden="true" size={24} /></div><h3>{t("ai.overview.tagsTitle")}</h3><p>{t("ai.overview.tagsDescription")}</p>
          <Link className={styles.textLink} to={`${paths.intelligence}?view=tags`}>{t("ai.reviewWorkspace")}<ArrowRight aria-hidden="true" size={16} /></Link>
        </article>
        <article className={styles.featureCard}><div className={styles.modelIcon}><UsersThree aria-hidden="true" size={24} /></div><h3>{t("ai.overview.peopleTitle")}</h3><p>{t("ai.overview.peopleDescription")}</p>
          <Button variant="quiet" onClick={() => onView("faces")}>{t("ai.overview.peopleSettings")}<ArrowRight aria-hidden="true" size={16} /></Button>
        </article>
      </div>
    </section>
    {!activeModel && <section className={styles.card}><header className={styles.cardHeader}><div><h2>{t("ai.overview.startTitle")}</h2><p>{t("ai.overview.startDescription")}</p></div></header>
      <ol className={styles.setupSteps}><li><strong>{t("ai.overview.step1")}</strong><span>{t("ai.overview.step1Detail")}</span></li><li><strong>{t("ai.overview.step2")}</strong><span>{t("ai.overview.step2Detail")}</span></li><li><strong>{t("ai.overview.step3")}</strong><span>{t("ai.overview.step3Detail")}</span></li></ol>
    </section>}
  </div>;
}
