import { ArrowClockwise } from "@phosphor-icons/react";
import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { ManagementShell } from "../../components/patterns/ManagementShell/ManagementShell";
import { Button, Dialog, EmptyState, ErrorState, InlineStatus, LoadingState } from "../../components/ui";
import type { AuthenticatedSession } from "../../lib/api/auth";
import { useLocale } from "../../lib/i18n/LocaleProvider";
import { paths } from "../../routes/paths";
import { DeletionSettingsPanel } from "./DeletionSettingsPanel";
import { FileActionDialog, type FileActionIntent } from "./FileActionDialog";
import { deletionItemErrorMessage } from "./presentation";
import { useDeletionOperationQuery, useDeletionOperationsQuery } from "./queries";
import styles from "./Deletion.module.css";

export function FileManagementPage({ session, logoutPending, onLogout }: { session: AuthenticatedSession; logoutPending?: boolean; onLogout?: () => Promise<void> }) {
  const { t } = useLocale();
  const [params, setParams] = useSearchParams();
  const view = params.get("view") === "settings" ? "settings" : "tasks";
  const tabs = ["tasks", "settings"] as const;
  return <ManagementShell active="files" homeHref={paths.root} generalHref={paths.generalSettings} accountHref={paths.accountSettings} storageHref={paths.storageSettings} librariesHref={paths.libraries} searchHref={paths.search} identity={session.administrator.displayName} logoutPending={logoutPending} onLogout={onLogout}>
    <div className={styles.page}><header className={styles.header}><h1>{t(`deletion.${view}`)}</h1>{view !== "settings" && <p>{t("deletion.noTasksHint")}</p>}</header>
      <div className={styles.tabs} role="tablist" aria-label={t("deletion.files")}>{tabs.map(tab => <Button key={tab} role="tab" aria-selected={tab === view} variant={tab === view ? "secondary" : "quiet"} onClick={() => setParams({ view: tab })}>{t(`deletion.${tab}`)}</Button>)}</div>
      {view === "settings" ? <DeletionSettingsPanel session={session} /> : <FileTasksPanel session={session} />}
    </div>
  </ManagementShell>;
}
export function FileTasksPanel({ session }: { session: AuthenticatedSession }) {
  const { locale, t } = useLocale();
  const [params, setParams] = useSearchParams();
  const cursor = params.get("cursor") ?? "";
  const query = useDeletionOperationsQuery(cursor);
  const [intent, setIntent] = useState<FileActionIntent>();
  const [detailId, setDetailId] = useState("");
  const [detailCursor, setDetailCursor] = useState("");
  const detail = useDeletionOperationQuery(detailId, detailCursor);
  function navigatePage(value: string) { setParams({ view: "tasks", ...(value ? { cursor: value } : {}) }); }
  return <section className={styles.stack} role="tabpanel">
    <div className={styles.panel}>
    <div className={styles.filters}><Button onClick={() => void query.refetch()}><ArrowClockwise size={18} aria-hidden="true" />{t("deletion.refresh")}</Button></div>
    {query.isPending && <LoadingState label={t("deletion.checking")} />}
    {query.isError && <ErrorState message={t("deletion.loadFailed")} onRetry={() => void query.refetch()} />}
    {query.isSuccess && !query.data.items.length && <EmptyState title={t("deletion.noTasks")} description={t("deletion.noTasksHint")} />}
    <ul className={styles.list}>{query.data?.items.map(operation => <li key={operation.id} className={styles.row}>
      <div className={styles.identity}><strong>{t(`deletion.kind.${operation.kind}`)} · {t(`deletion.state.${operation.state}`)}</strong><span className={styles.muted}>{new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(new Date(operation.createdAt))}</span>{operation.items.find(item => item.state === "running") && <span className={styles.muted}>{operation.items.find(item => item.state === "running")!.name}</span>}<span>{t("deletion.progress").replace("{done}", String(operation.completedItems)).replace("{total}", String(operation.totalItems)).replace("{failed}", String(operation.failedItems))}</span>{["partial", "needs_attention", "failed"].includes(operation.state) && <InlineStatus tone="warning">{t("deletion.partialHint")}</InlineStatus>}</div>
      <div className={styles.actions}><Button size="small" onClick={() => { setDetailCursor(""); setDetailId(operation.id); }}>{t("deletion.details")}</Button>{operation.canStop && <Button size="small" onClick={() => setIntent({ kind: "stop", operation })}>{t("deletion.pause")}</Button>}{operation.canResume && <Button size="small" onClick={() => setIntent({ kind: "resume", operation })}>{t("deletion.resume")}</Button>}</div>
    </li>)}</ul>
    {(cursor || query.data?.nextCursor) && <div className={styles.pagination}>{cursor && <Button onClick={() => navigatePage("")}>{t("deletion.firstPage")}</Button>}{query.data?.nextCursor && <Button onClick={() => navigatePage(query.data!.nextCursor!)}>{t("deletion.more")}</Button>}</div>}
    </div>
    <Dialog open={Boolean(detailId)} onOpenChange={open => { if (!open) setDetailId(""); }} title={t("deletion.details")}>
      <div className={styles.stack}>{detail.isPending && <LoadingState label={t("deletion.checking")} />}{detail.isError && <ErrorState message={t("deletion.loadFailed")} onRetry={() => void detail.refetch()} />}<ul className={styles.list}>{detail.data?.items.map(item => <li key={item.id} className={styles.identity}><strong>{item.name}</strong><span className={styles.muted}>{item.libraryName} · {item.relativePath}</span><span>{t(`deletion.item.${item.state}`)}</span>{item.removedEntries > 0 && <span className={styles.muted}>{t("deletion.removed").replace("{count}", String(item.removedEntries))}</span>}{item.errorCode && <InlineStatus tone="warning">{t(deletionItemErrorMessage(item.errorCode))}</InlineStatus>}</li>)}</ul><div className={styles.toolbar}>{detailCursor && <Button onClick={() => setDetailCursor("")}>{t("deletion.firstPage")}</Button>}{detail.data?.itemsNextCursor && <Button onClick={() => setDetailCursor(detail.data!.itemsNextCursor!)}>{t("deletion.more")}</Button>}</div></div>
    </Dialog>
    {intent && <FileActionDialog intent={intent} session={session} onAccepted={() => undefined} onClose={() => setIntent(undefined)} />}
  </section>;
}
