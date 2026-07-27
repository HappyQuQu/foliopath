import { useMemo } from "react";
import {
  useLocation,
  useNavigate,
  useSearchParams,
} from "react-router-dom";

import {
  MediaViewer,
  type MediaViewerItem,
} from "../../../components/patterns/MediaViewer/MediaViewer";
import { mediaPreviewDetails } from "../../../components/patterns/MediaPreview/mediaPreviewDetails";
import { Button, ErrorState, LoadingState } from "../../../components/ui";
import { assetContentUrl } from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  readViewerLocationState,
  safeViewerReturnPath,
  type ViewerReturnState,
} from "../../../lib/navigation/viewer";
import { paths } from "../../../routes/paths";
import { useAssetQuery } from "../queries";
import styles from "./MediaViewerPage.module.css";

export function MediaViewerPage({
  assetId,
  libraryId,
}: {
  assetId: string;
  libraryId: string;
}) {
  const { locale, t } = useLocale();
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const assetQuery = useAssetQuery(assetId);
  const { refetch: retryAsset } = assetQuery;
  const viewerState = readViewerLocationState(location.state);
  const returnTo = safeViewerReturnPath(
    viewerState?.returnTo ?? searchParams.get("from"),
    libraryId,
  );
  const sequence = viewerState?.sequence ?? [];
  const currentIndex = sequence.findIndex((item) => item.id === assetId);
  const previous = currentIndex > 0 ? sequence[currentIndex - 1] : undefined;
  const next =
    currentIndex >= 0 && currentIndex < sequence.length - 1
      ? sequence[currentIndex + 1]
      : undefined;
  const asset = assetQuery.data;
  const viewerItem = useMemo<MediaViewerItem | undefined>(
    () =>
      asset
        ? {
            contentUrl: assetContentUrl(asset.id),
            details: mediaPreviewDetails(asset, locale, {
              animated: t("browse.kindAnimated"),
              dimensions: t("browse.detailDimensions"),
              duration: t("browse.detailDuration"),
              image: t("browse.kindImage"),
              modified: t("browse.detailModified"),
              path: t("browse.detailPath"),
              size: t("browse.detailSize"),
              type: t("browse.detailType"),
              unknown: t("browse.detailUnknown"),
              video: t("browse.kindVideo"),
            }),
            id: asset.id,
            kind: asset.kind,
            name: asset.name,
          }
        : undefined,
    [asset, locale, t],
  );

  function closeViewer() {
    const state: ViewerReturnState = { restoreFocusAssetId: assetId };
    navigate(returnTo, { replace: true, state });
  }

  function moveTo(candidate: { id: string; libraryId: string } | undefined) {
    if (!candidate) return;
    navigate(paths.media(candidate.libraryId, candidate.id, returnTo), {
      replace: true,
      state: viewerState,
    });
  }

  if (assetQuery.isPending) {
    return (
      <ViewerState>
        <LoadingState label={t("viewer.loading")} />
      </ViewerState>
    );
  }
  if (assetQuery.isError || !viewerItem || asset?.libraryId !== libraryId) {
    return (
      <ViewerState>
        <ErrorState
          message={t("viewer.loadFailed")}
          onRetry={() => void retryAsset()}
        />
        <Button onClick={closeViewer} variant="quiet">
          {t("viewer.close")}
        </Button>
      </ViewerState>
    );
  }

  return (
    <MediaViewer
      canGoNext={Boolean(next)}
      canGoPrevious={Boolean(previous)}
      item={viewerItem}
      labels={{
        close: t("viewer.close"),
        exitFullscreen: t("viewer.exitFullscreen"),
        fit: t("viewer.fit"),
        fullscreen: t("viewer.fullscreen"),
        imageFailed: t("viewer.imageFailed"),
        info: t("viewer.info"),
        information: t("viewer.information"),
        next: t("browse.nextMedia"),
        originalSize: t("viewer.originalSize"),
        previous: t("browse.previousMedia"),
        shortcutHint: t("viewer.shortcutHint"),
        videoFailed: t("viewer.videoFailed"),
        zoomIn: t("viewer.zoomIn"),
        zoomOut: t("viewer.zoomOut"),
      }}
      onClose={closeViewer}
      onNext={() => moveTo(next)}
      onPrevious={() => moveTo(previous)}
      position={
        currentIndex >= 0
          ? t("browse.previewPosition")
              .replace("{current}", String(currentIndex + 1))
              .replace("{total}", String(sequence.length))
          : t("viewer.directPosition")
      }
    />
  );
}

function ViewerState({ children }: { children: React.ReactNode }) {
  return <main className={styles.state}>{children}</main>;
}
