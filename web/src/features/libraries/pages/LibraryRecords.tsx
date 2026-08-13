import {
  ArrowCounterClockwise,
  Broom,
  CaretDown,
  CaretUp,
  CircleNotch,
  ClockCounterClockwise,
  WarningCircle,
} from "@phosphor-icons/react";
import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import {
  Button,
  EmptyState,
  ErrorState,
  InlineStatus,
  LoadingState,
  Select,
} from "../../../components/ui";
import type { MediaFailure } from "../../../lib/api/diagnostics";
import type { LibrarySummary } from "../../../lib/api/libraries";
import type { ScanStatus } from "../../../lib/api/scans";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import {
  clearClearedMediaFailureRevision,
  readClearedMediaFailureRevision,
  writeAcknowledgedMediaFailureRevision,
  writeClearedMediaFailureRevision,
} from "../../../lib/storage/preferences";
import { useMediaFailureQuery, useMediaFailuresQuery } from "../../diagnostics";
import {
  mergeLibraryScanRecords,
  useLibraryScanRecordQueries,
} from "../library-record-queries";
import styles from "./LibraryRecords.module.css";

export function LibraryScanRecords({
  libraries,
}: {
  libraries: LibrarySummary[];
}) {
  const { locale, t } = useLocale();
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedLibrary = searchParams.get("libraryId") ?? "all";
  const selectedStatus = searchParams.get("scanStatus") ?? "all";
  const queries = useLibraryScanRecordQueries(libraries);
  const records = mergeLibraryScanRecords(
    libraries,
    queries.map((query) => query.data),
  ).filter(
    (record) =>
      (selectedLibrary === "all" || record.libraryId === selectedLibrary) &&
      (selectedStatus === "all" || record.scan.status === selectedStatus),
  );
  const pending = queries.some((query) => query.isPending);
  const failed = queries.some((query) => query.isError);
  const date = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }),
    [locale],
  );

  return (
    <div role="tabpanel">
      <RecordFilters
        libraries={libraries}
        onChange={(key, value) => updateParam(searchParams, setSearchParams, key, value)}
        values={{ libraryId: selectedLibrary, scanStatus: selectedStatus }}
        view="scans"
      />
      {pending && <LoadingState label={t("libraryRecords.scansLoading")} />}
      {failed && (
        <ErrorState
          message={t("libraryRecords.scansLoadFailed")}
          onRetry={() => {
            for (const query of queries) void query.refetch();
          }}
        />
      )}
      {!pending && !failed && records.length === 0 && (
        <EmptyState
          description={t("libraryRecords.scansEmptyDescription")}
          title={t("libraryRecords.scansEmpty")}
        />
      )}
      {records.length > 0 && (
        <section className={styles.list} aria-label={t("libraryRecords.scansTab")}>
          {records.map((record) => {
            const active = record.scan.status === "queued" || record.scan.status === "running";
            return (
              <article className={styles.row} key={record.id}>
                {active ? (
                  <CircleNotch aria-hidden="true" className={styles.spinning} size={22} />
                ) : (
                  <ClockCounterClockwise aria-hidden="true" size={22} />
                )}
                <div className={styles.body}>
                  <div className={styles.heading}>
                    <strong>{t("logs.operation.fullScan")}</strong>
                    <InlineStatus tone={scanTone(record.scan.status)}>
                      {t(`logs.operation.status.${record.scan.status}` as MessageKey)}
                    </InlineStatus>
                  </div>
                  <p>
                    {t("logs.operation.scanSummary")
                      .replace("{assets}", String(record.scan.discoveredAssets))
                      .replace("{directories}", String(record.scan.discoveredDirectories))
                      .replace("{errors}", String(record.scan.errorCount))}
                  </p>
                  <span>
                    {record.libraryName} · {t(`logs.operation.trigger.${record.scan.trigger}` as MessageKey)} · {date.format(new Date(record.scan.startedAt ?? record.scan.createdAt))}
                  </span>
                </div>
              </article>
            );
          })}
        </section>
      )}
    </div>
  );
}

export function LibraryProcessingResults({
  libraries,
}: {
  libraries: LibrarySummary[];
}) {
  const { locale, t } = useLocale();
  const [clearedRevision, setClearedRevision] = useState(
    readClearedMediaFailureRevision,
  );
  const [searchParams, setSearchParams] = useSearchParams();
  const libraryId = searchParams.get("libraryId") ?? "all";
  const rawVariant = searchParams.get("variant");
  const variant = rawVariant === "grid" || rawVariant === "storyboard" ? rawVariant : "all";
  const rawError = searchParams.get("errorCode");
  const errorCode = isFailureCode(rawError) ? rawError : "all";
  const query = useMediaFailuresQuery({
    ...(libraryId === "all" ? {} : { libraryId }),
    ...(variant === "all" ? {} : { variant }),
    ...(errorCode === "all" ? {} : { errorCode }),
  });
  const globalQuery = useMediaFailuresQuery();
  const globalRevision = globalQuery.data?.pages[0]?.revision ?? undefined;
  const allItems = query.data?.pages.flatMap((page) => page.items) ?? [];
  const items = clearedRevision
    ? allItems.filter((failure) => mediaFailureIsAfterRevision(failure, clearedRevision))
    : allItems;
  const date = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }),
    [locale],
  );

  return (
    <div role="tabpanel">
      <div className={styles.resultActions}>
        <Button
          disabled={!globalRevision || globalRevision === clearedRevision}
          onClick={() => {
            if (!globalRevision) return;
            setClearedRevision(globalRevision);
            writeClearedMediaFailureRevision(globalRevision);
            writeAcknowledgedMediaFailureRevision(globalRevision);
          }}
          size="small"
          variant="secondary"
        >
          <Broom aria-hidden="true" size={16} />
          {t("libraryRecords.clearResults")}
        </Button>
        {clearedRevision && (
          <Button
            onClick={() => {
              setClearedRevision(undefined);
              clearClearedMediaFailureRevision();
            }}
            size="small"
            variant="quiet"
          >
            <ArrowCounterClockwise aria-hidden="true" size={16} />
            {t("libraryRecords.restoreResults")}
          </Button>
        )}
      </div>
      <RecordFilters
        libraries={libraries}
        onChange={(key, value) => updateParam(searchParams, setSearchParams, key, value)}
        values={{ errorCode, libraryId, variant }}
        view="results"
      />
      {query.isPending && <LoadingState label={t("libraryRecords.resultsLoading")} />}
      {query.isError && (
        <ErrorState
          message={t("libraryRecords.resultsLoadFailed")}
          onRetry={() => void query.refetch()}
        />
      )}
      {query.isSuccess && items.length === 0 && (
        <EmptyState
          description={t(clearedRevision && allItems.length > 0
            ? "libraryRecords.resultsClearedDescription"
            : "libraryRecords.resultsEmptyDescription")}
          title={t(clearedRevision && allItems.length > 0
            ? "libraryRecords.resultsCleared"
            : "libraryRecords.resultsEmpty")}
        />
      )}
      {items.length > 0 && (
        <section className={styles.list} aria-label={t("libraryRecords.resultsTab")}>
          {items.map((failure) => (
            <FailureRow date={date} failure={failure} key={failure.id} />
          ))}
          {query.hasNextPage && (
            <div className={styles.loadMore}>
              <Button
                loading={query.isFetchingNextPage}
                onClick={() => void query.fetchNextPage()}
                variant="secondary"
              >
                {t("logs.loadMore")}
              </Button>
            </div>
          )}
        </section>
      )}
    </div>
  );
}

function RecordFilters({
  libraries,
  onChange,
  values,
  view,
}: {
  libraries: LibrarySummary[];
  onChange: (key: string, value: string) => void;
  values: Record<string, string>;
  view: "scans" | "results";
}) {
  const { t } = useLocale();
  return (
    <div className={styles.filters}>
      <label>
        <span>{t("libraryRecords.libraryFilter")}</span>
        <Select onChange={(event) => onChange("libraryId", event.currentTarget.value)} value={values.libraryId}>
          <option value="all">{t("libraryRecords.allLibraries")}</option>
          {libraries.map((library) => <option key={library.id} value={library.id}>{library.name}</option>)}
        </Select>
      </label>
      {view === "scans" ? (
        <label>
          <span>{t("libraryRecords.statusFilter")}</span>
          <Select onChange={(event) => onChange("scanStatus", event.currentTarget.value)} value={values.scanStatus}>
            <option value="all">{t("libraryRecords.allStatuses")}</option>
            {(["queued", "running", "succeeded", "failed", "cancelled", "offline", "interrupted"] as const).map((status) => (
              <option key={status} value={status}>{t(`logs.operation.status.${status}` as MessageKey)}</option>
            ))}
          </Select>
        </label>
      ) : (
        <>
          <label>
            <span>{t("libraryRecords.typeFilter")}</span>
            <Select onChange={(event) => onChange("variant", event.currentTarget.value)} value={values.variant}>
              <option value="all">{t("libraryRecords.allTypes")}</option>
              <option value="grid">{t("logs.variant.grid")}</option>
              <option value="storyboard">{t("logs.variant.storyboard")}</option>
            </Select>
          </label>
          <label>
            <span>{t("libraryRecords.reasonFilter")}</span>
            <Select onChange={(event) => onChange("errorCode", event.currentTarget.value)} value={values.errorCode}>
              <option value="all">{t("libraryRecords.allReasons")}</option>
              {failureCodes.map((code) => <option key={code} value={code}>{t(`logs.error.${code}` as MessageKey)}</option>)}
            </Select>
          </label>
        </>
      )}
    </div>
  );
}

function FailureRow({ date, failure }: { date: Intl.DateTimeFormat; failure: MediaFailure }) {
  const { t } = useLocale();
  const [expanded, setExpanded] = useState(false);
  const detail = useMediaFailureQuery(failure.id, expanded);
  const latest = failure.latestAttempt;
  return (
    <article className={styles.row}>
      <WarningCircle aria-hidden="true" size={22} />
      <div className={styles.body}>
        <div className={styles.heading}>
          <strong>{failure.relativePath}</strong>
          <InlineStatus tone="warning">{t(`logs.error.${failure.errorCode}` as MessageKey)}</InlineStatus>
        </div>
        <p>
          {latest?.reasonCode
            ? t(`diagnostics.reason.${latest.reasonCode}` as MessageKey)
            : t(`logs.errorDescription.${failure.errorCode}` as MessageKey)}
        </p>
        <div className={styles.metaRow}>
          <span>{failure.libraryName} · {t(`logs.variant.${failure.variant}` as MessageKey)} · {date.format(new Date(failure.finishedAt))} · {t("logs.attempts").replace("{count}", String(failure.attempts))}</span>
          <Button
            aria-expanded={expanded}
            onClick={() => setExpanded((value) => !value)}
            size="small"
            variant="quiet"
          >
            {expanded ? <CaretUp aria-hidden="true" size={16} /> : <CaretDown aria-hidden="true" size={16} />}
            {t(expanded ? "diagnostics.collapse" : "diagnostics.expand")}
          </Button>
        </div>
        {expanded && (
          <section className={styles.diagnosticDetail} aria-label={t("diagnostics.detailTitle")}>
            {detail.isPending && <LoadingState label={t("diagnostics.loading")} />}
            {detail.isError && <ErrorState message={t("diagnostics.loadFailed")} onRetry={() => void detail.refetch()} />}
            {detail.data && detail.data.attemptHistory.length === 0 && (
              <p className={styles.legacy}>{t("diagnostics.legacyFailure")}</p>
            )}
            {detail.data?.attemptHistory.map((attempt) => (
              <div className={styles.attempt} key={`${attempt.attemptNumber}-${attempt.finishedAt}`}>
                <strong>{t("diagnostics.attemptTitle").replace("{number}", String(attempt.attemptNumber))}</strong>
                <dl>
                  <div><dt>{t("diagnostics.stage")}</dt><dd>{attempt.stage ? t(`diagnostics.stage.${attempt.stage}` as MessageKey) : t("diagnostics.unknown")}</dd></div>
                  <div><dt>{t("diagnostics.reason")}</dt><dd>{attempt.reasonCode ? t(`diagnostics.reason.${attempt.reasonCode}` as MessageKey) : t("diagnostics.unknown")}</dd></div>
                  <div><dt>{t("diagnostics.tool")}</dt><dd>{attempt.tool ?? t("diagnostics.unknown")}</dd></div>
                  <div><dt>{t("diagnostics.duration")}</dt><dd>{formatDuration(attempt.durationMs)}</dd></div>
                  {attempt.exitCode !== null && <div><dt>{t("diagnostics.exitCode")}</dt><dd>{attempt.exitCode}</dd></div>}
                </dl>
              </div>
            ))}
          </section>
        )}
      </div>
    </article>
  );
}

function formatDuration(durationMS: number): string {
  if (durationMS < 1_000) return `${durationMS} ms`;
  return `${(durationMS / 1_000).toFixed(durationMS < 10_000 ? 1 : 0)} s`;
}

function scanTone(status: ScanStatus): "info" | "success" | "warning" {
  if (status === "succeeded") return "success";
  if (status === "queued" || status === "running") return "info";
  return "warning";
}

const failureCodes = [
  "invalid_media",
  "unsupported_media",
  "media_processing_failed",
  "media_processing_timeout",
  "source_unavailable",
  "cache_unavailable",
] as const;

function isFailureCode(value: string | null): value is (typeof failureCodes)[number] {
  return failureCodes.some((code) => code === value);
}

function mediaFailureIsAfterRevision(
  failure: MediaFailure,
  revision: string,
): boolean {
  const revisionMatch = /^mfailrev_([1-9][0-9]*)_([1-9][0-9]*)$/.exec(revision);
  const jobMatch = /^mjob_([1-9][0-9]*)$/.exec(failure.id);
  const finishedAt = Date.parse(failure.finishedAt);
  if (!revisionMatch || !jobMatch || !Number.isFinite(finishedAt)) return true;

  const clearedAt = Number(revisionMatch[1]);
  if (finishedAt !== clearedAt) return finishedAt > clearedAt;
  return Number(jobMatch[1]) > Number(revisionMatch[2]);
}

function updateParam(
  params: URLSearchParams,
  setParams: ReturnType<typeof useSearchParams>[1],
  key: string,
  value: string,
) {
  const next = new URLSearchParams(params);
  if (value === "all") next.delete(key);
  else next.set(key, value);
  setParams(next, { replace: true });
}
