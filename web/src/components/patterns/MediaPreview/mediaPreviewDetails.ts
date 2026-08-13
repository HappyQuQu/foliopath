export interface MediaPreviewDetailAsset {
  durationMs: number | null;
  height: number | null;
  kind: "image" | "animated" | "video";
  mimeType: string;
  modifiedAt: string;
  relativePath: string;
  sizeBytes: number;
  width: number | null;
}

export interface MediaPreviewDetailLabels {
  animated: string;
  dimensions: string;
  duration: string;
  image: string;
  modified: string;
  path: string;
  size: string;
  type: string;
  unknown: string;
  video: string;
}

export function mediaPreviewDetails(
  asset: MediaPreviewDetailAsset,
  locale: string,
  labels: MediaPreviewDetailLabels,
): Array<{
  label: string;
  layout?: "default" | "path";
  value: string;
}> {
  const number = new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
  });
  const modified = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(asset.modifiedAt));
  const dimensions =
    asset.width && asset.height
      ? `${number.format(asset.width)} × ${number.format(asset.height)} px`
      : labels.unknown;
  const kind =
    asset.kind === "video"
      ? labels.video
      : asset.kind === "animated"
        ? labels.animated
        : labels.image;
  const details = [
    { label: labels.type, value: `${kind} · ${asset.mimeType}` },
    { label: labels.path, layout: "path" as const, value: asset.relativePath },
    { label: labels.modified, value: modified },
    { label: labels.dimensions, value: dimensions },
    { label: labels.size, value: formatBytes(asset.sizeBytes, locale) },
  ];

  if (asset.durationMs !== null) {
    details.push({
      label: labels.duration,
      value: formatDuration(asset.durationMs),
    });
  }
  return details;
}

function formatBytes(bytes: number, locale: string): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = "B";
  for (const nextUnit of units) {
    value /= 1024;
    unit = nextUnit;
    if (value < 1024) break;
  }
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
  }).format(value)} ${unit}`;
}

function formatDuration(durationMs: number): string {
  const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`
    : `${minutes}:${String(seconds).padStart(2, "0")}`;
}
