import { ErrorState, LoadingState } from "../../components/ui";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { useAssetExifQuery } from "./queries";
import styles from "./AssetExifPanel.module.css";

export function AssetExifPanel({
  assetId,
  kind,
}: {
  assetId: string;
  kind: "image" | "animated" | "video";
}) {
  const { t } = useLocale();
  const query = useAssetExifQuery(assetId, kind !== "video");
  if (kind === "video") return null;

  return (
    <section className={styles.panel} aria-label={t("exif.title")}>
      <h3>{t("exif.title")}</h3>
      {query.isPending ? (
        <LoadingState density="compact" label={t("exif.loading")} />
      ) : query.isError ? (
        <ErrorState
          density="compact"
          message={t("exif.failed")}
          onRetry={() => void query.refetch()}
        />
      ) : query.data.fields.length === 0 ? (
        <p>{t("exif.empty")}</p>
      ) : (
        <dl>
          {query.data.fields.map(({ key, value }) => (
            <div key={key}>
              <dt>{t(`exif.${key}`)}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
      )}
    </section>
  );
}
