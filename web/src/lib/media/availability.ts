import type { MediaAvailabilityKind } from "../../components/patterns/MediaAvailabilityState/MediaAvailabilityState";
import type { MediaAvailabilityPresentation } from "../../components/patterns/MediaAvailabilityState/MediaAvailabilityState";
import type { Asset } from "../api/catalog";
import type { MessageKey } from "../i18n/LocaleProvider";

export function mediaAvailability(asset: Asset): MediaAvailabilityKind | undefined {
  if (asset.sourceAvailability !== "available") {
    return asset.sourceAvailability;
  }
  if (asset.probeStatus === "failed") return "invalid";
  if (asset.probeStatus === "unsupported") return "unsupported";
  if (asset.kind === "video" && asset.playbackStatus === "unsupported_codec") {
    return "unsupportedCodec";
  }
  return undefined;
}

export function mediaPosterUrl(asset: Asset): string | undefined {
  return asset.thumbnail.status === "ready" && asset.thumbnail.url
    ? asset.thumbnail.url
    : undefined;
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
