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
import { LoadingState } from "../../../components/ui";
import { assetContentUrl } from "../../../lib/api/catalog";
import { ApiError } from "../../../lib/api/errors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import {
  mediaAvailability,
  mediaAvailabilityPresentation,
  mediaPosterUrl,
} from "../../../lib/media/availability";
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
            summary:
              asset.kind === "video"
                ? t("browse.kindVideo")
                : asset.kind === "animated"
                  ? t("browse.kindAnimated")
                  : t("browse.kindImage"),
            ...(mediaPosterUrl(asset)
              ? { posterUrl: mediaPosterUrl(asset) }
              : {}),
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
  const usableViewerItem =
    viewerItem && asset?.libraryId === libraryId
      ? viewerItem
      : {
          contentUrl: "",
          details: [],
          id: assetId,
          kind: "image" as const,
          name:
            assetQuery.error instanceof ApiError &&
            assetQuery.error.code === "asset_not_found"
              ? t("mediaState.deletedTitle")
              : t("mediaState.loadFailedTitle"),
        };
  const availabilityKind =
    asset && asset.libraryId === libraryId
      ? mediaAvailability(asset)
      : assetQuery.error instanceof ApiError &&
          assetQuery.error.code === "asset_not_found"
        ? "deleted"
        : "loadFailed";
  const availability = availabilityKind
    ? mediaAvailabilityPresentation(
        availabilityKind,
        t,
        availabilityKind === "deleted" ? undefined : () => void retryAsset(),
      )
    : undefined;

  return (
    <MediaViewer
      availability={availability}
      canGoNext={Boolean(next)}
      canGoPrevious={Boolean(previous)}
      item={usableViewerItem}
      labels={{
        close: t("viewer.close"),
        closeInformation: t("viewer.closeInformation"),
        exitFullscreen: t("viewer.exitFullscreen"),
        fit: t("viewer.fit"),
        fullscreen: t("viewer.fullscreen"),
        imageFailed: t("viewer.imageFailed"),
        info: t("viewer.info"),
        information: t("viewer.information"),
        loadFailedDescription: t("mediaState.loadFailedDescription"),
        next: t("browse.nextMedia"),
        originalSize: t("viewer.originalSize"),
        previous: t("browse.previousMedia"),
        retry: t("mediaState.retry"),
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
