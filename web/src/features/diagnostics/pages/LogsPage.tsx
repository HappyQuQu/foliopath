import {
  CheckCircle,
  Info,
  WarningCircle,
  XCircle,
} from "@phosphor-icons/react";
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { ManagementShell } from "../../../components/patterns/ManagementShell/ManagementShell";
import {
  Button,
  EmptyState,
  ErrorState,
  InlineStatus,
  LoadingState,
  Select,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type {
  SystemEvent,
  SystemEventLevel,
} from "../../../lib/api/system-logs";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { paths } from "../../../routes/paths";
import { useSystemEventsQuery } from "../queries";
import styles from "./LogsPage.module.css";

const levels: Array<SystemEventLevel | "all"> = [
  "all",
  "error",
  "warning",
  "info",
];

export function LogsPage({
  logoutPending,
  onLogout,
  session,
}: {
  logoutPending?: boolean | undefined;
  onLogout?: (() => Promise<void>) | undefined;
  session: AuthenticatedSession;
}) {
  const { locale, t } = useLocale();
  const [searchParams, setSearchParams] = useSearchParams();
  const rawLevel = searchParams.get("level");
  const level =
    rawLevel === "error" || rawLevel === "warning" || rawLevel === "info"
      ? rawLevel
      : "all";
  const module = searchParams.get("module") ?? "all";
  const query = useSystemEventsQuery({
    ...(level === "all" ? {} : { level }),
    ...(module === "all" ? {} : { module }),
  });
  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const date = useMemo(
    () =>
      new Intl.DateTimeFormat(locale, {
        dateStyle: "medium",
        timeStyle: "medium",
      }),
    [locale],
  );

  function updateFilters(next: { level?: string; module?: string }) {
    const params = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(next)) {
      if (!value || value === "all") params.delete(key);
      else params.set(key, value);
    }
    setSearchParams(params, { replace: true });
  }

  return (
    <ManagementShell
      active="logs"
      accountHref={paths.accountSettings}
      aboutHref={paths.aboutSettings}
      generalHref={paths.generalSettings}
      homeHref={paths.root}
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      logsHref={paths.logsSettings}
      logoutPending={logoutPending}
      onLogout={onLogout}
      searchHref={paths.search}
      storageHref={paths.generalSettings}
    >
      <div className={styles.main}>
        <header className={styles.hero}>
          <p>{t("management.title")}</p>
          <h1>{t("systemLogs.title")}</h1>
          <span>{t("systemLogs.description")}</span>
        </header>

        <div className={styles.filters}>
          <div className={styles.levels} aria-label={t("systemLogs.levelFilter")}>
            {levels.map((item) => (
              <Button
                aria-pressed={level === item}
                key={item}
                onClick={() => updateFilters({ level: item })}
                size="small"
                variant={level === item ? "secondary" : "quiet"}
              >
                {t(`systemLogs.level.${item}` as MessageKey)}
              </Button>
            ))}
          </div>
          <label>
            <span>{t("systemLogs.moduleFilter")}</span>
            <Select
              onChange={(event) => updateFilters({ module: event.currentTarget.value })}
              value={module}
            >
              <option value="all">{t("systemLogs.module.all")}</option>
              <option value="application">{t("systemLogs.module.application")}</option>
              <option value="http">{t("systemLogs.module.http")}</option>
            </Select>
          </label>
        </div>

        {query.isPending && <LoadingState label={t("systemLogs.loading")} />}
        {query.isError && (
          <ErrorState
            message={t("systemLogs.loadFailed")}
            onRetry={() => void query.refetch()}
          />
        )}
        {query.isSuccess && items.length === 0 && (
          <EmptyState
            description={t("systemLogs.emptyDescription")}
            title={t("systemLogs.empty")}
          />
        )}
        {items.length > 0 && (
          <section className={styles.list} aria-label={t("systemLogs.title")}>
            {items.map((event) => (
              <SystemEventRow date={date} event={event} key={event.id} />
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
    </ManagementShell>
  );
}

function SystemEventRow({
  date,
  event,
}: {
  date: Intl.DateTimeFormat;
  event: SystemEvent;
}) {
  const { t } = useLocale();
  const Icon =
    event.level === "error"
      ? XCircle
      : event.level === "warning"
        ? WarningCircle
        : event.eventCode === "http.admin_operation"
          ? CheckCircle
          : Info;
  return (
    <article className={`${styles.row} ${styles[event.level]}`}>
      <Icon aria-hidden="true" size={22} weight="duotone" />
      <div className={styles.body}>
        <div className={styles.heading}>
          <strong>{t(eventMessageKey(event.eventCode))}</strong>
          <InlineStatus tone={eventTone(event.level)}>
            {t(`systemLogs.level.${event.level}` as MessageKey)}
          </InlineStatus>
        </div>
        <p>{eventDescription(event, t)}</p>
        <span>
          {date.format(new Date(event.occurredAt))} · {t(`systemLogs.module.${event.module}` as MessageKey)}
        </span>
        {(event.requestId || event.routePattern || event.statusCode) && (
          <details className={styles.details}>
            <summary>{t("systemLogs.technicalDetails")}</summary>
            <dl>
              {event.requestId && (
                <div><dt>{t("systemLogs.requestId")}</dt><dd>{event.requestId}</dd></div>
              )}
              {event.routePattern && (
                <div><dt>{t("systemLogs.route")}</dt><dd>{event.method} {event.routePattern}</dd></div>
              )}
              {event.statusCode && (
                <div><dt>{t("systemLogs.status")}</dt><dd>{event.statusCode}</dd></div>
              )}
              {event.durationMs !== null && (
                <div><dt>{t("systemLogs.duration")}</dt><dd>{event.durationMs} ms</dd></div>
              )}
            </dl>
          </details>
        )}
      </div>
    </article>
  );
}

function eventMessageKey(code: string): MessageKey {
  switch (code) {
    case "application.started": return "systemLogs.event.applicationStarted";
    case "application.stopped": return "systemLogs.event.applicationStopped";
    case "http.request_failed": return "systemLogs.event.requestFailed";
    case "http.request_rejected": return "systemLogs.event.requestRejected";
    default: return "systemLogs.event.adminOperation";
  }
}

function eventDescription(
  event: SystemEvent,
  t: (key: MessageKey) => string,
): string {
  if (event.eventCode === "http.request_failed") {
    return t("systemLogs.description.requestFailed");
  }
  if (event.eventCode === "http.request_rejected") {
    return t("systemLogs.description.requestRejected");
  }
  if (event.eventCode === "http.admin_operation") {
    return t("systemLogs.description.adminOperation");
  }
  return t("systemLogs.description.applicationLifecycle");
}

function eventTone(level: SystemEventLevel): "danger" | "info" | "warning" {
  if (level === "error") return "danger";
  if (level === "warning") return "warning";
  return "info";
}
