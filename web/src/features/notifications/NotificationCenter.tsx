import {
  Bell,
  CheckCircle,
  CircleNotch,
  DownloadSimple,
  WarningCircle,
} from "@phosphor-icons/react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { Button, IconButton } from "../../components/ui";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import {
  readAcknowledgedMediaFailureRevision,
  readDismissedCompletedNotifications,
  writeAcknowledgedMediaFailureRevision,
  writeDismissedCompletedNotifications,
} from "../../lib/storage/preferences";
import { paths } from "../../routes/paths";
import { useMediaFailuresQuery } from "../diagnostics";
import {
  mediaProcessingIsActive,
  useLibrariesMediaProcessingProgressQueries,
  useLibrariesQuery,
} from "../libraries";
import { useReleaseInformationQuery } from "../release-info";
import styles from "./NotificationCenter.module.css";

export function NotificationCenter() {
  const { locale, t } = useLocale();
  const [open, setOpen] = useState(false);
  const [dismissed, setDismissed] = useState<Set<string>>(readDismissed);
  const [acknowledgedFailureRevision, setAcknowledgedFailureRevision] = useState(
    readAcknowledgedMediaFailureRevision,
  );
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const librariesQuery = useLibrariesQuery(true);
  const failuresQuery = useMediaFailuresQuery();
  const releasesQuery = useReleaseInformationQuery();
  const libraries = librariesQuery.data?.pages.flatMap((page) => page.items) ?? [];
  const mediaProgressQueries = useLibrariesMediaProcessingProgressQueries(libraries);
  const activeScans = libraries.filter((library) => library.status === "scanning");
  const activeMedia = libraries.flatMap((library, index) => {
    const progress = mediaProgressQueries[index]?.data;
    return mediaProcessingIsActive(progress) && progress
      ? [{ library, progress }]
      : [];
  });
  const completed = libraries
    .filter(
      (library) =>
        library.lastSuccessfulScanAt &&
        library.latestScanId &&
        !dismissed.has(library.latestScanId) &&
        library.status !== "scanning",
    )
    .sort((left, right) =>
      (right.lastSuccessfulScanAt ?? "").localeCompare(left.lastSuccessfulScanAt ?? ""),
    )
    .slice(0, 5);
  const failures = failuresQuery.data?.pages[0]?.items ?? [];
  const failureRevision = failuresQuery.data?.pages[0]?.revision ?? undefined;
  const hasNewFailures = Boolean(
    failureRevision && failureRevision !== acknowledgedFailureRevision,
  );
  const activeLibraryIds = new Set([
    ...activeScans.map((library) => library.id),
    ...activeMedia.map(({ library }) => library.id),
  ]);
  const badge = activeLibraryIds.size + (hasNewFailures ? 1 : 0) + (releasesQuery.data?.updateAvailable ? 1 : 0);
  const date = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short" }),
    [locale],
  );

  useEffect(() => {
    if (!open) return;
    function close(event: KeyboardEvent | PointerEvent) {
      if (event instanceof KeyboardEvent) {
        if (event.key !== "Escape") return;
        setOpen(false);
        triggerRef.current?.focus();
        return;
      }
      if (event.target instanceof Node && !rootRef.current?.contains(event.target)) {
        setOpen(false);
      }
    }
    document.addEventListener("keydown", close);
    document.addEventListener("pointerdown", close);
    return () => {
      document.removeEventListener("keydown", close);
      document.removeEventListener("pointerdown", close);
    };
  }, [open]);

  const completedIDs = useMemo(
    () => completed.flatMap((library) => (library.latestScanId ? [library.latestScanId] : [])),
    [completed],
  );

  function clearCompleted() {
    const next = new Set(dismissed);
    for (const id of completedIDs) next.add(id);
    setDismissed(next);
    writeDismissedCompletedNotifications([...next]);
  }

  function acknowledgeFailures() {
    if (!failureRevision) return;
    setAcknowledgedFailureRevision(failureRevision);
    writeAcknowledgedMediaFailureRevision(failureRevision);
  }

  return (
    <div className={styles.root} ref={rootRef}>
      <IconButton
        aria-expanded={open}
        aria-haspopup="dialog"
        label={t("notifications.title")}
        onClick={() => {
          setAcknowledgedFailureRevision(readAcknowledgedMediaFailureRevision());
          setOpen((value) => !value);
        }}
        ref={triggerRef}
      >
        <Bell aria-hidden="true" size={20} />
        {badge > 0 && <span className={styles.badge}>{badge > 9 ? "9+" : badge}</span>}
      </IconButton>
      {open && (
        <section className={styles.panel} aria-label={t("notifications.title")}>
          <header>
            <h2>{t("notifications.title")}</h2>
            {completed.length > 0 && (
              <Button onClick={clearCompleted} size="small" variant="quiet">
                {t("notifications.clearCompleted")}
              </Button>
            )}
          </header>

          {releasesQuery.data?.updateAvailable && (
            <Link className={styles.item} onClick={() => setOpen(false)} to={paths.aboutSettings}>
              <DownloadSimple aria-hidden="true" size={20} />
              <span>
                <strong>{t("notifications.updateAvailable")}</strong>
                <small>{releasesQuery.data.latestVersion}</small>
                <time dateTime={releasesQuery.data.checkedAt}>
                  {date.format(new Date(releasesQuery.data.checkedAt))}
                </time>
              </span>
            </Link>
          )}

          {activeScans.map((library) => (
            <Link
              className={styles.item}
              key={library.id}
              onClick={() => setOpen(false)}
              to={paths.libraryInlineStatus(library.id)}
            >
              <CircleNotch aria-hidden="true" className={styles.spinning} size={20} />
              <span>
                <strong>{t("notifications.scanActive")}</strong>
                <small>{library.name}</small>
              </span>
            </Link>
          ))}

          {activeMedia.map(({ library, progress }) => {
            const total = progress.thumbnails.total + progress.videoPreviews.total;
            const processed = progress.thumbnails.processed + progress.videoPreviews.processed;
            const queued = progress.thumbnails.queued + progress.videoPreviews.queued;
            const running = progress.thumbnails.running + progress.videoPreviews.running;
            const failed = progress.thumbnails.failed + progress.videoPreviews.failed;
            return (
              <Link
                className={styles.item}
                key={`media-${library.id}`}
                onClick={() => setOpen(false)}
                to={paths.libraryInlineStatus(library.id)}
              >
                <CircleNotch aria-hidden="true" className={styles.spinning} size={20} />
                <span>
                  <strong>{t("notifications.mediaActive")}</strong>
                  <small>{library.name}</small>
                  <progress
                    aria-label={t("notifications.mediaProgress")
                      .replace("{processed}", String(processed))
                      .replace("{total}", String(total))}
                    max={Math.max(1, total)}
                    value={processed}
                  />
                  <small className={styles.progressMeta}>
                    {t("notifications.mediaProgressDetails")
                      .replace("{processed}", String(processed))
                      .replace("{total}", String(total))
                      .replace("{queued}", String(queued))
                      .replace("{running}", String(running))
                      .replace("{failed}", String(failed))}
                  </small>
                </span>
              </Link>
            );
          })}

          {hasNewFailures && (
            <div className={styles.actionItem}>
              <Link
                className={styles.item}
                onClick={() => {
                  acknowledgeFailures();
                  setOpen(false);
                }}
                to={paths.libraryProcessingResults()}
              >
                <WarningCircle aria-hidden="true" size={20} />
                <span>
                  <strong>{t("notifications.failures")}</strong>
                  <small>{t("notifications.failureCount").replace("{count}", String(failures.length))}</small>
                  {failures[0]?.finishedAt && (
                    <time dateTime={failures[0].finishedAt}>
                      {date.format(new Date(failures[0].finishedAt))}
                    </time>
                  )}
                </span>
              </Link>
              <Button onClick={acknowledgeFailures} size="small" variant="quiet">
                {t("notifications.markRead")}
              </Button>
            </div>
          )}

          {completed.map((library) => (
            <Link
              className={styles.item}
              key={library.latestScanId}
              onClick={() => setOpen(false)}
              to={paths.libraryScanRecords(library.id)}
            >
              <CheckCircle aria-hidden="true" size={20} />
              <span>
                <strong>{t("notifications.scanCompleted")}</strong>
                <small>{library.name}</small>
                {library.lastSuccessfulScanAt && (
                  <time dateTime={library.lastSuccessfulScanAt}>
                    {date.format(new Date(library.lastSuccessfulScanAt))}
                  </time>
                )}
              </span>
            </Link>
          ))}

          {badge === 0 && completed.length === 0 && (
            <p className={styles.empty}>{t("notifications.empty")}</p>
          )}
        </section>
      )}
    </div>
  );
}

function readDismissed() {
  return new Set(readDismissedCompletedNotifications());
}
