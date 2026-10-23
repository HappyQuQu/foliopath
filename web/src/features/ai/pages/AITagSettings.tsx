import { Check, X } from "@phosphor-icons/react";
import { useId, useState } from "react";
import { useNavigate } from "react-router-dom";

import { Button, Dialog, EmptyState, ErrorState, FormField, IconButton, InlineStatus, LoadingState, useToast } from "../../../components/ui";
import type { AIModelOperation, AITagVocabularySnapshot } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { ApiError } from "../../../lib/api/errors";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useTagsQuery } from "../../curation";
import { useAITagSuggestionsQuery } from "../queries";
import { useAITagJobCoverageRefresh, useAITagVocabularyQuery, useReplaceAITagVocabularyMutation, useRequestAITagSuggestionJobMutation } from "../tagQueries";
import styles from "./AITagSettings.module.css";

export interface AITagSettingsProps {
  session: AuthenticatedSession;
  library?: LibrarySummary;
  semanticEnabled: boolean;
  semanticGenerationReady?: boolean;
  serviceReady: boolean;
  onOperation: (operation: AIModelOperation) => void;
  mode: "vocabulary" | "jobs";
  disabled?: boolean;
}

export function AITagSettings(props: AITagSettingsProps) {
  const { t } = useLocale();
  if (props.mode === "vocabulary") return <TagVocabularySettings session={props.session} />;
  if (!props.library) return <InlineStatus>{t("ai.tags.noLibrary")}</InlineStatus>;
  return <TagSuggestionJobs key={props.library.id} {...props} library={props.library} />;
}

function TagVocabularySettings({ session }: { session: AuthenticatedSession }) {
  const { t } = useLocale();
  const query = useAITagVocabularyQuery();
  const headingId = useId();
  return <section className={styles.card} aria-labelledby={headingId}>
    <header className={styles.header}><h2 id={headingId}>{t("ai.tags.vocabularyTitle")}</h2><p>{t("ai.tags.description")}</p></header>
    {query.isError && <ErrorState message={t("ai.tags.loadFailed")} onRetry={() => void query.refetch()} />}
    {query.data ? <TagVocabularyEditor session={session} snapshot={query.data} refresh={() => void query.refetch()} unavailable={query.isError} />
      : query.isPending && <LoadingState label={t("ai.loadingStatus")} />}
  </section>;
}

function TagVocabularyEditor({ session, snapshot, refresh, unavailable }: {
  session: AuthenticatedSession;
  snapshot: AITagVocabularySnapshot;
  refresh: () => void;
  unavailable: boolean;
}) {
  const { locale, t } = useLocale();
  const navigate = useNavigate();
  const toast = useToast();
  const tags = useTagsQuery();
  const save = useReplaceAITagVocabularyMutation();
  const selectedHeading = useId();
  const [draft, setDraft] = useState(snapshot);
  const [base, setBase] = useState(snapshot);
  const [filter, setFilter] = useState("");
  const [pageIndex, setPageIndex] = useState(0);
  const [failure, setFailure] = useState<MessageKey | null>(null);
  const selected = new Set(draft.entries.map((entry) => entry.tagId));
  const changed = draft.entries.length !== base.entries.length || base.entries.some((entry) => !selected.has(entry.tagId));
  const conflict = snapshot.etag !== base.etag;
  const page = tags.data?.pages[pageIndex];
  const visible = page?.items.filter((tag) => tag.name.toLocaleLowerCase(locale).includes(filter.toLocaleLowerCase(locale))) ?? [];
  const lastLoaded = (tags.data?.pages.length ?? 1) - 1;
  const hasNext = pageIndex < lastLoaded || Boolean(tags.hasNextPage);

  function restore() {
    setDraft(snapshot);
    setBase(snapshot);
    setFailure(null);
  }

  function choose(tagId: string, name: string) {
    if (save.isPending) return;
    setDraft((current) => ({ ...current, entries: current.entries.some((entry) => entry.tagId === tagId)
      ? current.entries.filter((entry) => entry.tagId !== tagId)
      : current.entries.length < 1000 ? [...current.entries, { tagId, name }] : current.entries }));
    setFailure(null);
  }

  async function publish() {
    if (!changed || conflict || save.isPending || unavailable) return;
    try {
      const value = await save.mutateAsync({ csrfToken: session.csrfToken, etag: base.etag, tagIds: draft.entries.map((entry) => entry.tagId) });
      setDraft(value);
      setBase(value);
      setFailure(null);
      toast.show({ message: t("ai.tags.saved"), tone: "success" });
    } catch (error) {
      setFailure(tagActionError(error, "ai.tags.saveFailed"));
      if (error instanceof ApiError && ["precondition_failed", "tag_not_found"].includes(error.code)) {
        refresh();
        void tags.refetch();
      }
    }
  }

  async function nextPage() {
    if (pageIndex < lastLoaded) { setPageIndex(pageIndex + 1); return; }
    const result = await tags.fetchNextPage();
    if (!result.isFetchNextPageError && result.data && result.data.pages.length > pageIndex + 1) setPageIndex(pageIndex + 1);
  }

  return <div className={styles.editor}>
    <div className={styles.scope}><strong>{t("ai.tags.globalScope")}</strong><p>{t("ai.tags.globalDescription")}</p></div>
    <section aria-labelledby={selectedHeading} className={styles.selection}>
      <h3 id={selectedHeading}>{t("ai.tags.selected").replace("{count}", new Intl.NumberFormat(locale).format(draft.entries.length))}</h3>
      {draft.entries.length > 0 ? <ul className={styles.selectedTags}>{draft.entries.map((entry) => <li key={entry.tagId}>
        <span>{entry.name}</span><IconButton disabled={save.isPending} label={t("ai.tags.remove").replace("{name}", entry.name)} onClick={() => choose(entry.tagId, entry.name)}><X size={14} aria-hidden="true" /></IconButton>
      </li>)}</ul> : <InlineStatus>{t("ai.tags.emptyVocabulary")}</InlineStatus>}
    </section>
    <div className={styles.choicesHeader}><h3>{t("ai.tags.available")}</h3><Button variant="quiet" size="small" onClick={() => navigate(paths.tags)}>{t("ai.tags.manage")}</Button></div>
    {tags.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : tags.isError && !tags.data ? <ErrorState message={t("ai.tags.loadFailed")} onRetry={() => void tags.refetch()} /> : <>
      {!page?.items.length && !tags.hasNextPage ? <EmptyState title={t("ai.tags.noTags")} description={t("ai.tags.noTagsDescription")} /> : <>
        <FormField label={t("ai.tags.filter")} type="search" value={filter} onChange={(event) => setFilter(event.currentTarget.value)} />
        <ul className={styles.choices} aria-label={t("ai.tags.available")}>{visible.map((tag) => <li key={tag.id}>
          <Button size="small" aria-label={t("ai.tags.choice").replace("{name}", tag.name)} aria-pressed={selected.has(tag.id)} variant={selected.has(tag.id) ? "primary" : "secondary"}
            disabled={save.isPending || (!selected.has(tag.id) && selected.size >= 1000)} onClick={() => choose(tag.id, tag.name)}>
            {selected.has(tag.id) && <Check size={14} aria-hidden="true" />}{tag.name}
          </Button>
        </li>)}</ul>
        {!visible.length && <InlineStatus>{t("ai.tags.noMatches")}</InlineStatus>}
        {tags.isFetchNextPageError && <ErrorState message={t("ai.tags.loadFailed")} onRetry={() => void nextPage()} />}
        {(pageIndex > 0 || hasNext) && <div className={styles.actions}>
          <Button size="small" disabled={pageIndex === 0 || tags.isFetchingNextPage} onClick={() => setPageIndex(pageIndex - 1)}>{t("ai.tags.previousPage")}</Button>
          <Button size="small" disabled={!hasNext} loading={tags.isFetchingNextPage} onClick={() => void nextPage()}>{t("intelligence.nextPage")}</Button>
        </div>}
      </>}
    </>}
    <p className={styles.note}>{t("ai.tags.limit")} {t("ai.tags.languageHint")}</p>
    {conflict && <InlineStatus tone="warning">{t("ai.tags.changed")}<Button size="small" variant="quiet" onClick={restore}>{t("ai.tags.reload")}</Button></InlineStatus>}
    {failure && !conflict && <InlineStatus tone="danger">{t(failure)}</InlineStatus>}
    <div className={styles.actions}>
      <Button variant="primary" loading={save.isPending} disabled={!changed || conflict || unavailable} onClick={() => void publish()}>{t("ai.tags.save")}</Button>
      <Button variant="quiet" disabled={save.isPending || (!changed && !conflict)} onClick={restore}>{t("ai.tags.revert")}</Button>
      {changed && <span className={styles.note} role="status">{t("ai.tags.unsaved")}</span>}
    </div>
  </div>;
}

function TagSuggestionJobs({ session, library, semanticEnabled, semanticGenerationReady = false, serviceReady, onOperation, disabled = false }: AITagSettingsProps & { library: LibrarySummary }) {
  const { locale, t } = useLocale();
  const navigate = useNavigate();
  const toast = useToast();
  const vocabulary = useAITagVocabularyQuery();
  const suggestions = useAITagSuggestionsQuery(library.id, "pending");
  const job = useRequestAITagSuggestionJobMutation();
  const headingId = useId();
  const [confirmAll, setConfirmAll] = useState(false);
  const [failure, setFailure] = useState<MessageKey | null>(null);
  const [submittedOperationId, setSubmittedOperationId] = useState<string | null>(null);
  useAITagJobCoverageRefresh(library.id, submittedOperationId);
  const unavailable = library.status === "offline" || !semanticEnabled || !semanticGenerationReady || !serviceReady || vocabulary.isError || vocabulary.isPending || !vocabulary.data?.entries.length;
  const blocked = disabled || job.isPending || unavailable;
  const coverage = suggestions.data?.coverage;
  const number = new Intl.NumberFormat(locale);
  const reason: MessageKey | null = library.status === "offline" ? "ai.offlinePreserved" : !semanticEnabled ? "ai.tags.needsSemantic" : !serviceReady ? "ai.tags.needsService" : !semanticGenerationReady ? "ai.tags.needsGeneration" : vocabulary.data && !vocabulary.data.entries.length ? "ai.tags.emptyVocabulary" : null;

  async function generate(mode: "missing" | "all") {
    if (blocked) return;
    try {
      const operation = await job.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), libraryId: library.id, mode });
      setSubmittedOperationId(operation.id);
      onOperation(operation);
      setConfirmAll(false);
      setFailure(null);
      toast.show({ message: t("ai.tags.jobStarted"), tone: "success" });
    } catch (error) {
      setFailure(tagActionError(error, "ai.tags.jobFailed"));
      setConfirmAll(false);
    }
  }

  return <section className={`${styles.card} ${styles.jobs}`} aria-labelledby={headingId}>
    <header className={styles.header}><h2 id={headingId}>{t("ai.tags.title")}</h2><p>{t("ai.tags.description")}</p></header>
    <p className={styles.note}>{t("ai.tags.libraryScope").replace("{name}", library.name)} · {t("ai.tags.jobDescription")}</p>
    {vocabulary.isPending && <LoadingState label={t("ai.loadingStatus")} />}
    {vocabulary.isError && <ErrorState message={t("ai.tags.loadFailed")} onRetry={() => void vocabulary.refetch()} />}
    {reason && <InlineStatus tone="warning">{t(reason)}</InlineStatus>}
    {suggestions.isPending ? <LoadingState label={t("ai.loadingStatus")} /> : suggestions.isError ? <ErrorState message={t("ai.tags.coverageFailed")} onRetry={() => void suggestions.refetch()} /> : coverage && <div className={styles.coverage}>
      <progress aria-label={t("ai.tags.title")} max={Math.max(1, coverage.eligible)} value={coverage.completed} />
      <span>{t("ai.tags.coverage").replace("{completed}", number.format(coverage.completed)).replace("{eligible}", number.format(coverage.eligible)).replace("{failed}", number.format(coverage.failed))}</span>
    </div>}
    {failure && <InlineStatus tone="danger">{t(failure)}</InlineStatus>}
    <div className={styles.actions}>
      <Button variant="primary" disabled={blocked} loading={job.isPending} onClick={() => void generate("missing")}>{t("ai.tags.generate")}</Button>
      <Button variant="quiet" disabled={blocked} onClick={() => setConfirmAll(true)}>{t("ai.tags.regenerate")}</Button>
      <Button variant="quiet" onClick={() => navigate(`${paths.intelligence}?${new URLSearchParams({ view: "tags", libraryId: library.id }).toString()}`)}>{t("ai.tags.review")}</Button>
    </div>
    <Dialog open={confirmAll} onOpenChange={setConfirmAll} title={t("ai.tags.regenerateTitle")} actions={<>
      <Button variant="quiet" disabled={job.isPending} onClick={() => setConfirmAll(false)}>{t("common.cancel")}</Button>
      <Button variant="primary" disabled={blocked} loading={job.isPending} onClick={() => void generate("all")}>{t("ai.tags.confirmRegenerate")}</Button>
    </>}><p>{t("ai.tags.regenerateDescription").replace("{name}", library.name)}</p></Dialog>
  </section>;
}

function tagActionError(error: unknown, fallback: MessageKey): MessageKey {
  if (error instanceof ApiError) {
    if (error.code === "precondition_failed") return "ai.tags.changed";
    if (error.code === "tag_not_found") return "ai.tags.tagMissing";
    if (error.code === "model_unavailable") return "ai.tags.needsService";
    if (error.code === "ai_disabled") return "ai.tags.needsSemantic";
    if (error.code === "vocabulary_unavailable") return "ai.tags.emptyVocabulary";
  }
  return fallback;
}
