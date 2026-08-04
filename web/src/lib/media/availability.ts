import type { MediaAvailabilityKind } from "../../components/patterns/MediaAvailabilityState/MediaAvailabilityState";
import type { MediaAvailabilityPresentation } from "../../components/patterns/MediaAvailabilityState/MediaAvailabilityState";
import type { Asset } from "../api/catalog";
import type { MessageKey } from "../i18n/LocaleProvider";

export interface ReadyMediaStoryboard {
  cellHeight: number;
  cellWidth: number;
  columns: number;
  frameCount: 4 | 10;
  rows: number;
  url: string;
}

export function mediaAvailability(
  asset: Asset,
): MediaAvailabilityKind | undefined {
  if (asset.sourceAvailability !== "available") {
    return asset.sourceAvailability;
  }
  if (asset.probeStatus === "failed") return "invalid";
  if (asset.probeStatus === "unsupported") return "unsupported";
  return undefined;
}

export function mediaPosterUrl(asset: Asset): string | undefined {
  return asset.thumbnail.status === "ready" && asset.thumbnail.url
    ? asset.thumbnail.url
    : undefined;
}

export function mediaStoryboard(
  asset: Asset,
): ReadyMediaStoryboard | undefined {
  const storyboard = asset.storyboard;
  if (
    storyboard.status !== "ready" ||
    !storyboard.url ||
    (storyboard.frameCount !== 4 && storyboard.frameCount !== 10) ||
    storyboard.columns === null ||
    storyboard.rows === null ||
    storyboard.cellWidth === null ||
    storyboard.cellHeight === null ||
    storyboard.columns < 1 ||
    storyboard.columns > 5 ||
    storyboard.rows < 1 ||
    storyboard.rows > 2 ||
    storyboard.cellWidth < 1 ||
    storyboard.cellWidth > 320 ||
    storyboard.cellHeight < 1 ||
    storyboard.cellHeight > 320 ||
    storyboard.columns * storyboard.rows < storyboard.frameCount
  ) {
    return undefined;
  }
  return {
    cellHeight: storyboard.cellHeight,
    cellWidth: storyboard.cellWidth,
    columns: storyboard.columns,
    frameCount: storyboard.frameCount,
    rows: storyboard.rows,
    url: storyboard.url,
  };
}

export function mediaDerivedStatePending(asset: Asset): boolean {
  return (
    asset.thumbnail.status === "pending" ||
    asset.storyboard.status === "pending"
  );
}

const availabilityMessageKeys: Record<
  MediaAvailabilityKind,
  { description: MessageKey; title: MessageKey }
> = {
  deleted: {
    description: "mediaState.deletedDescription",
    title: "mediaState.deletedTitle",
  },
  invalid: {
    description: "mediaState.invalidDescription",
    title: "mediaState.invalidTitle",
  },
  loadFailed: {
    description: "mediaState.loadFailedDescription",
    title: "mediaState.loadFailedTitle",
  },
  missing: {
    description: "mediaState.missingDescription",
    title: "mediaState.missingTitle",
  },
  offline: {
    description: "mediaState.offlineDescription",
    title: "mediaState.offlineTitle",
  },
  unreadable: {
    description: "mediaState.unreadableDescription",
    title: "mediaState.unreadableTitle",
  },
  unsupported: {
    description: "mediaState.unsupportedDescription",
    title: "mediaState.unsupportedTitle",
  },
  unsupportedCodec: {
    description: "mediaState.unsupportedCodecDescription",
    title: "mediaState.unsupportedCodecTitle",
  },
};

export function mediaAvailabilityPresentation(
  kind: MediaAvailabilityKind,
  t: (key: MessageKey) => string,
  onRetry?: () => void,
): MediaAvailabilityPresentation {
  const keys = availabilityMessageKeys[kind];
  const retryable = kind !== "deleted" && kind !== "unsupportedCodec";
  return {
    description: t(keys.description),
    kind,
    title: t(keys.title),
    ...(retryable && onRetry
      ? { actionLabel: t("mediaState.retry"), onAction: onRetry }
      : {}),
  };
}
