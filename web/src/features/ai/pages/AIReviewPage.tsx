import { useQueries } from "@tanstack/react-query";
import { ArrowRight, SlidersHorizontal, Sparkle, UsersThree } from "@phosphor-icons/react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { ContextSidebar } from "../../../components/patterns/ContextSidebar/ContextSidebar";
import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import { Button, Dialog, DialogCloseButton, EmptyState, ErrorState, InlineStatus, Input, LoadingState, Select, useToast } from "../../../components/ui";
import type { AITagSuggestion } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { assetContentUrl } from "../../../lib/api/catalog";
import { getAssetCuration } from "../../../lib/api/curation";
import { ApiError } from "../../../lib/api/errors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { DirectoryNavigation, defaultBrowseUrlState } from "../../browse";
import { useLibrariesQuery } from "../../libraries";
import { curationKeys } from "../../curation";
import {
  useAITagSuggestionsQuery, useCreatePersonMutation,
  useFaceClustersQuery, usePeopleQuery, useReviewAITagSuggestionsMutation,
} from "../queries";
import { useAssetQuery } from "../../media";
import styles from "./AIReviewPage.module.css";
import { PersonPanel } from "./PersonPanel";
import { ClusterPanel } from "./ClusterPanel";
import { usePeopleDirectoryState } from "./usePeopleDirectoryState";
import { usePersonCreationIntent } from "./usePersonCreationIntent";
import { ReviewPhotoPreview } from "./ReviewPhotoPreview";
import { ReviewErrorState, ReviewPagination } from "./ReviewFeedback";
import { PhotoFacesPanel } from "./PhotoFacesPanel";
import { FaceReviewUndo, type FaceReviewReceipt } from "./FaceReviewUndo";

type ReviewView = "tags" | "people" | "clusters" | "faces";

export function AIReviewPage({ logoutPending, onLogout, session }: {
  logoutPending?: boolean; onLogout?: () => Promise<void>; session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const [reviewEpoch, setReviewEpoch] = useState(0);
  const [undoReceipt, setUndoReceipt] = useState<FaceReviewReceipt | null>(null);
  const reviewCallbacks = { onReviewStart: () => setUndoReceipt(null), onReviewCommitted: (receipt: FaceReviewReceipt) => setUndoReceipt(receipt.result.undoable && receipt.result.action !== "undo" ? receipt : null) };
  const [params, setParams] = useSearchParams();
  const librariesQuery = useLibrariesQuery();
  const libraries = librariesQuery.data?.pages.flatMap((page) => page.items) ?? [];
  const faceAsset = useAssetQuery(params.get("view") === "faces" ? params.get("asset") ?? "" : "");
  const libraryId = faceAsset.data?.libraryId ?? params.get("libraryId") ?? libraries[0]?.id ?? "";
  const currentLibrary = libraries.find((library) => library.id === libraryId);
  const view = parseView(params.get("view"));
  const reviewingTags = view === "tags";
  const title = t(reviewingTags ? "intelligence.tags" : "intelligence.title");
  const personId = params.get("person") ?? "";
  const clusterId = params.get("cluster") ?? "";
  const assetId = params.get("asset") ?? "";
  const headingRef = useRef<HTMLHeadingElement>(null);
  const focusRoute = [view, personId, clusterId, assetId, params.get("cursor"), params.get("groupsCursor"), params.get("review"), params.get("status")].join(":");
  const previousFocusRoute = useRef(focusRoute);
  useLayoutEffect(() => {
    if (previousFocusRoute.current !== focusRoute && document.activeElement === document.body) {
      headingRef.current?.focus();
    }
    previousFocusRoute.current = focusRoute;
  }, [focusRoute]);
  useEffect(() => {
    if (view !== "people" || personId || !params.has("cursor")) return;
    const next = new URLSearchParams(params);
    if (!next.has("peopleCursor")) next.set("peopleCursor", next.get("cursor")!);
    next.delete("cursor");
    setParams(next, { replace: true });
  }, [params, personId, setParams, view]);
  useEffect(() => {
    if (view !== "clusters" || clusterId || !params.has("cursor")) return;
    const next = new URLSearchParams(params);
    if (!next.has("groupsCursor")) next.set("groupsCursor", next.get("cursor")!);
    next.delete("cursor");
    setParams(next, { replace: true });
  }, [clusterId, params, setParams, view]);
  useEffect(()=>{
    if(view!=="faces" || !faceAsset.data || params.get("libraryId")===faceAsset.data.libraryId)return;
    const next=new URLSearchParams(params);next.set("libraryId",faceAsset.data.libraryId);setParams(next,{replace:true});
  },[faceAsset.data,params,setParams,view]);
  const faceLibraryOffline = view === "clusters" && currentLibrary?.status === "offline";

  function navigate(nextView: ReviewView, extras: Record<string, string> = {}) {
    setParams({ view: nextView, ...(libraryId ? { libraryId } : {}),
      ...(nextView === "clusters" ? { ...(params.get("review") === "all" ? { review: "all" } : {}), ...(params.get("kind") === "edge" ? { kind: "edge" } : {}), ...(params.get("groupsCursor") ? { groupsCursor: params.get("groupsCursor")! } : {}) } : {}),
      ...(nextView === "people" ? { ...(params.get("q") ? { q: params.get("q")! } : {}), ...(params.get("peopleCursor") ? { peopleCursor: params.get("peopleCursor")! } : {}) } : {}), ...extras });
  }

  return <AppShell active="settings" homeHref={paths.root} identity={session.administrator.displayName}
    logoutPending={logoutPending} onLogout={onLogout} searchHref={paths.search}
    settingsHref={paths.generalSettings} title={title}
    navigationKey={`${view}:${view === "people" ? personId : `${libraryId}:${clusterId}:${assetId}`}`}
    navigationLabel={reviewingTags ? undefined : t("context.openPeople")}
    sidebarLabel={reviewingTags ? undefined : t("context.people")}
    sidebarContent={<DirectoryNavigation
      activeQuickAccess={reviewingTags ? "suggestions" : "people"}
      browseState={defaultBrowseUrlState()}
      {...(currentLibrary ? { currentLibraryName: currentLibrary.name } : {})}
      hideDirectoryTree
      {...(view === "people" ? { scopeLabel: t("search.allLibraries") } : {})}
      libraries={libraries}
      libraryId={libraryId}
      onLibraryChange={(nextLibraryId) => setParams({ view: reviewingTags ? "tags" : "clusters", libraryId: nextLibraryId })}
      quickAccessDetail={!reviewingTags ? <ContextSidebar title={t("context.people")} description={t(view === "people" ? "intelligence.peopleGlobalScope" : "context.groupScope")}>
        <div className={styles.sidebarViews} role="group" aria-label={t("intelligence.title")}>
          {(["people", "clusters", ...(assetId || view === "faces" ? ["faces" as const] : [])] as const).map(item => <Button key={item} size="small" aria-pressed={view === item} variant={view === item ? "secondary" : "quiet"} onClick={() => navigate(item)}>{t(`intelligence.${item}`)}</Button>)}
        </div>
        {view === "people" && <PeopleSidebar selectedId={personId} onOpen={id => navigate("people", { person: id })} />}
        {view === "clusters" && <fieldset><legend>{t("intelligence.peopleGroups")}</legend>{(["core", "edge"] as const).map(kind => <Button key={kind} size="small" aria-pressed={(params.get("kind") === "edge" ? "edge" : "core") === kind} variant={(params.get("kind") === "edge" ? "edge" : "core") === kind ? "secondary" : "quiet"} onClick={() => { const next = new URLSearchParams(params); next.delete("cursor"); next.delete("groupsCursor"); next.delete("cluster"); if (kind === "core") next.delete("kind"); else next.set("kind", kind); setParams(next); }}>{t(`intelligence.${kind}`)}</Button>)}</fieldset>}
      </ContextSidebar> : undefined}
      selectedPathIds={new Set<string>()}
    />}
    topbarContent={<span>{title}</span>}>
    <div className={styles.main}>
      <header className={styles.heading}><div><p className={styles.eyebrow}><Sparkle aria-hidden="true" size={16} />{t("ai.overview.eyebrow")}</p><h1 ref={headingRef} tabIndex={-1}>{title}</h1><p>{t(reviewingTags ? "intelligence.tagsDescription" : "intelligence.description")}</p></div>
        <Link className={styles.settingsLink} to={`${paths.aiSettings}${libraryId ? `?library=${encodeURIComponent(libraryId)}` : ""}`}><SlidersHorizontal aria-hidden="true" size={18} />{t("intelligence.manageModels")}<ArrowRight aria-hidden="true" size={16} /></Link>
      </header>
      {!reviewingTags && undoReceipt && <FaceReviewUndo key={undoReceipt.result.reviewId} receipt={undoReceipt} session={session} fallbackFocusRef={headingRef} onDismiss={() => setUndoReceipt(null)} onStateResolved={() => setReviewEpoch(value => value + 1)} />}
      {view === "people" && <p className={styles.scopeNote}>{t("intelligence.peopleGlobalScope")}</p>}
      {libraries.length > 0 && view === "tags" && <label className={styles.libraryFilter}>{t("intelligence.library")}
        <Select value={libraryId} onChange={(event) => setParams({ view, libraryId: event.target.value })}>
          {libraries.map((library) => <option value={library.id} key={library.id}>{library.name}</option>)}
        </Select></label>}
      {librariesQuery.isError && <ErrorState message={t("ai.loadFailed")} onRetry={() => void librariesQuery.refetch()} />}
      {!librariesQuery.isPending && !librariesQuery.isError && !libraries.length && <EmptyState title={t("ai.noLibraries")} description={t("ai.noLibrariesDescription")} />}
      {faceLibraryOffline && <ErrorState message={t("intelligence.offline")} onRetry={() => void librariesQuery.refetch()} />}
      {view === "tags" && <TagReviewPanel key={`${libraryId}:${params.get("status") ?? "pending"}:${params.get("cursor") ?? ""}`} libraryId={libraryId} session={session} />}
      {view === "people" && (personId ? <PersonPanel {...reviewCallbacks} key={`${personId}:${reviewEpoch}`} personId={personId} session={session} onBack={() => navigate("people")} /> : <PeoplePanel session={session} onOpen={(id) => navigate("people", { person: id })} />)}
      {view === "clusters" && !faceLibraryOffline && (clusterId ? <ClusterPanel {...reviewCallbacks} key={`${libraryId}:${clusterId}:${params.get("cursor") ?? ""}:${reviewEpoch}`} clusterId={clusterId} libraryId={libraryId} session={session} onBack={() => navigate("clusters")} /> : <ClustersPanel libraryId={libraryId} onOpen={(id) => navigate("clusters", { cluster: id })} />)}
      {view === "faces" && !faceLibraryOffline && <PhotoFacesPanel {...reviewCallbacks} key={`${assetId}:${reviewEpoch}`} assetId={assetId} onAsset={(id) => navigate("faces", { asset: id })} session={session} />}
      {!reviewingTags && <details className={styles.privacy}><summary>{t("intelligence.privacyTitle")}</summary><p>{t("intelligence.privacy")}</p></details>}
    </div>
  </AppShell>;
}

function TagReviewPanel({ libraryId, session }: { libraryId: string; session: AuthenticatedSession }) {
  const { t } = useLocale(); const toast = useToast();
  const [params, setParams] = useSearchParams();
  const status = parseSuggestionStatus(params.get("status"));
  const [selected, setSelected] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const currentFilterRef = useRef<HTMLButtonElement>(null);
  const query = useAITagSuggestionsQuery(libraryId, status, params.get("cursor") ?? "");
  const items = query.data?.items ?? [];
  const assetIds = [...new Set(items.map(item => item.asset.id))];
  const curations = useQueries({ queries: assetIds.map((id) => ({ enabled: status === "pending", queryKey: curationKeys.asset(id), queryFn: () => getAssetCuration(id) })) });
  const review = useReviewAITagSuggestionsMutation();
  const curationFor = (item: AITagSuggestion) => curations[assetIds.indexOf(item.asset.id)];
  const failedCurations = curations.filter(curation => curation.isError);
  const selectionReady = items.every((item) => !selected.includes(item.id) || (curationFor(item)?.data && !curationFor(item)?.isFetching && !curationFor(item)?.isError));
  async function submit(action: "accept" | "dismiss") {
    if (submitting || !selectionReady) return;
    const ready = items.filter(item => selected.includes(item.id) && curationFor(item)?.data);
    if (!ready.length) return;
    const submitFocus = document.activeElement;
    const saved = new Set<string>();
    const blockedAssets = new Set<string>();
    const revisions = new Map(assetIds.map((id, index) => [id, curations[index]?.data?.revision ?? 0]));
    setSubmitting(true);
    try {
      // Accepting a tag changes this photo's curation revision. Keep only one
      // suggestion per photo in each batch, and never auto-retry a conflict.
      let remainingItems = ready;
      while (remainingItems.length) {
        const batchAssets = new Set<string>();
        const batch = remainingItems.filter(item => {
          if (blockedAssets.has(item.asset.id) || (action === "accept" && batchAssets.has(item.asset.id))) return false;
          batchAssets.add(item.asset.id);
          return true;
        });
        if (!batch.length) break;
        const result = await review.mutateAsync({ action, csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), items: batch.map(suggestion => ({ suggestion, curationRevision: revisions.get(suggestion.asset.id)! })) });
        const outcomes = new Map(result.items.map(item => [item.suggestionId, item.outcome]));
        for (const suggestion of batch) {
          const outcome = outcomes.get(suggestion.id);
          if (outcome === "accepted" || outcome === "dismissed") saved.add(suggestion.id);
          else blockedAssets.add(suggestion.asset.id);
        }
        setSelected(ids => ids.filter(id => !saved.has(id)));
        const submitted = new Set(batch.map(item => item.id));
        remainingItems = remainingItems.filter(item => !submitted.has(item.id) && !blockedAssets.has(item.asset.id));
        if (action === "accept" && remainingItems.length) {
          for (const assetId of new Set(remainingItems.map(item => item.asset.id))) {
            const refreshed = await curations[assetIds.indexOf(assetId)]!.refetch();
            if (!refreshed.data || refreshed.isError) blockedAssets.add(assetId);
            else revisions.set(assetId, refreshed.data.revision);
          }
        }
      }
      const remaining = selected.filter((id) => !saved.has(id)).length;
      // The selected-only action bar disappears after the last successful item.
      // Restore a stable nearby control, but do not steal focus moved meanwhile.
      if (!remaining && (document.activeElement === submitFocus || document.activeElement === document.body)) {
        currentFilterRef.current?.focus();
      }
      toast.show({ message: remaining ? t("intelligence.reviewPartial").replace("{saved}", String(saved.size)).replace("{remaining}", String(remaining)) : t("intelligence.saved"), tone: remaining ? "danger" : "success" });
    }
    catch (error) { toast.show({ message: error instanceof ApiError && error.code === "precondition_failed" ? t("intelligence.revisionConflict") : t("intelligence.actionFailed"), tone: "danger" }); }
    finally { try { await Promise.all([query.refetch(), ...curations.map((curation) => curation.refetch())]); } finally { setSubmitting(false); } }
  }
  if (!libraryId) return null;
  if (query.isPending) return <LoadingState label={t("common.loading")} />;
  if (query.isError) return <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} />;
  const coverage = query.data.coverage;
  const groups = assetIds.map(assetId => items.filter(item => item.asset.id === assetId));
  const allSelected = items.length > 0 && items.every(item => selected.includes(item.id));
  return <section aria-label={t("intelligence.tags")}>
    <div className={styles.tagToolbar}>
      <div className={styles.filters} role="group" aria-label={t("intelligence.reviewStatus")}>{(["pending", "accepted", "dismissed"] as const).map((item) => <Button key={item} ref={status === item ? currentFilterRef : undefined} aria-pressed={status === item} aria-disabled={submitting || undefined} variant={status === item ? "secondary" : "quiet"} onClick={() => { if (submitting) return; const next = new URLSearchParams(params); next.delete("cursor"); if (item === "pending") next.delete("status"); else next.set("status", item); setParams(next); setSelected([]); }}>{t(`intelligence.${item}`)}</Button>)}</div>
      <Link className={styles.settingsLink} to={`${paths.tags}?libraryId=${encodeURIComponent(libraryId)}`}>{t("intelligence.manageTags")}<ArrowRight aria-hidden="true" size={16} /></Link>
    </div>
    <div className={styles.tagCoverage}>
      <div><strong>{t("intelligence.tagAnalysis")}</strong><span>{t("intelligence.coverage").replace("{completed}", String(coverage.completed)).replace("{eligible}", String(coverage.eligible)).replace("{failed}", String(coverage.failed)).replace("{degraded}", String(coverage.degraded))}</span></div>
      <progress aria-label={t("intelligence.tagAnalysis")} max={Math.max(1, coverage.eligible)} value={coverage.completed} />
    </div>
    {items.length > 0 && !coverage.complete && <InlineStatus tone="warning">{t("intelligence.tagIncomplete")}</InlineStatus>}
    {items.length > 0 && <p className={styles.meta}>{t(status === "pending" ? "intelligence.tagReviewHint" : status === "accepted" ? "intelligence.acceptedHint" : "intelligence.dismissedHint")}</p>}
    {status === "pending" && failedCurations.length > 0 && <ErrorState message={t("intelligence.tagMetadataFailed")} onRetry={() => { void Promise.all(failedCurations.map(curation => curation.refetch())); }} />}
    {status === "pending" && items.length > 0 && <div className={styles.tagSelection}>
      <label className={styles.selectPage}><input type="checkbox" checked={allSelected} disabled={submitting} onChange={event => setSelected(event.target.checked ? items.map(item => item.id) : [])} />{t("intelligence.selectPage")}</label>
      <span aria-live="polite">{t("intelligence.selectedCount").replace("{count}", String(selected.length))}</span>
      {selected.length > 0 && <div className={styles.tagActions}><Button variant="primary" disabled={review.isPending || submitting || !selectionReady} onClick={() => void submit("accept")}>{t("intelligence.accept")}</Button><Button disabled={review.isPending || submitting || !selectionReady} variant="secondary" onClick={() => void submit("dismiss")}>{t("intelligence.dismiss")}</Button></div>}
    </div>}
    {items.length === 0 ? <EmptyState title={t(status === "pending" ? "intelligence.noPendingSuggestions" : status === "accepted" ? "intelligence.noAcceptedSuggestions" : "intelligence.noDismissedSuggestions")} description={t(status !== "pending" ? "intelligence.emptyReviewed" : coverage.eligible === 0 ? "intelligence.noTagImages" : coverage.complete ? "intelligence.noMatchingSuggestions" : "intelligence.tagIncomplete")} /> : <>
      <p className={styles.resultSummary}>{t("intelligence.pageSuggestions").replace("{count}", String(items.length)).replace("{photos}", String(groups.length))}</p>
      <div className={styles.tagGrid}>{groups.map(group => <SuggestionCard items={group} key={group[0]!.asset.id} selected={selected} disabled={submitting} onSelect={(id, checked) => setSelected(current => checked ? [...new Set([...current, id])].slice(0, 100) : current.filter(value => value !== id))} />)}</div>
    </>}
    <ReviewPagination nextCursor={query.data.nextCursor} />
  </section>;
}

function SuggestionCard({ items, onSelect, selected, disabled }: { items: AITagSuggestion[]; onSelect: (id: string, selected: boolean) => void; selected: string[]; disabled: boolean }) {
  const { t } = useLocale();
  const asset = items[0]!.asset;
  return <article className={styles.suggestionCard}>
    <img className={styles.suggestionImage} loading="lazy" src={asset.thumbnail.url ?? assetContentUrl(asset.id)} alt={asset.name} />
    <div className={styles.suggestionBody}><h2 title={asset.name}>{asset.name}</h2>
      <div className={styles.suggestedTags}>{items.map(item => <label key={item.id} className={styles.suggestedTag} data-selected={selected.includes(item.id) || undefined}>
        {item.status === "pending" && <input type="checkbox" checked={selected.includes(item.id)} disabled={disabled} aria-label={`${t("intelligence.select").replace("{name}", asset.name)} · ${item.tag.name}`} onChange={event => onSelect(item.id, event.target.checked)} />}
        <strong>{item.tag.name}</strong><span>{t("intelligence.confidence").replace("{value}", String(Math.round(item.confidence * 100)))}</span>
      </label>)}</div>
    </div>
  </article>;
}

function PeopleSearchField({ label }: { label: string }) {
 const state = usePeopleDirectoryState();
 return <label>{label}<Input type="search" value={state.search} maxLength={100} onChange={event => state.setSearch(event.target.value)} /></label>;
}
function PeopleSidebar({ selectedId, onOpen }: { selectedId: string; onOpen: (id: string) => void }) {
  const { t } = useLocale(); const state = usePeopleDirectoryState(); const query = usePeopleQuery(state.search.trim(), state.cursor);
  return <>
    <PeopleSearchField label={t("intelligence.searchPeople")} />
    {query.isPending ? <LoadingState label={t("common.loading")} /> : query.isError ? <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} cursorParam="peopleCursor" /> : query.data.items.length === 0 ? <p>{t(state.search.trim() ? "peopleDirectory.noMatch" : "intelligence.noPeople")}</p> : <nav aria-label={t("context.peopleList")}>
      {query.data.items.map(person => <Button key={person.id} size="small" variant={person.id === selectedId ? "secondary" : "quiet"} aria-current={person.id === selectedId ? "page" : undefined} onClick={() => onOpen(person.id)}>
        <span>{person.name}</span><small>{person.assetCount}</small>
      </Button>)}
    </nav>}
    <ReviewPagination nextCursor={query.data?.nextCursor} cursorParam="peopleCursor" />
  </>;
}

function PeoplePanel({ onOpen, session }: { onOpen: (id: string) => void; session: AuthenticatedSession }) {
  const { t } = useLocale(); const state = usePeopleDirectoryState(); const query = usePeopleQuery(state.search.trim(), state.cursor);
  const [name, setName] = useState(""); const [open, setOpen] = useState(false); const [uncertain, setUncertain] = useState(false); const [saving, setSaving] = useState(false);
  const busy = useRef(false); const heading = useRef<HTMLHeadingElement>(null); const create = useCreatePersonMutation(); const intent = usePersonCreationIntent();
  async function createEmpty() {
    if (busy.current || !name.trim()) return; busy.current = true; setSaving(true); setUncertain(false);
    try {
      const person = await create.mutateAsync({ csrfToken: session.csrfToken, ...intent.forName(name) });
      intent.complete(); setOpen(false); setName(""); onOpen(person.id);
    } catch { setUncertain(true); }
    finally { busy.current = false; setSaving(false); }
  }
  return <section aria-labelledby="named-people-title">
    <div className={styles.clusterToolbar}><div><h2 id="named-people-title" ref={heading} tabIndex={-1}>{t("peopleDirectory.title")}</h2><p>{t("peopleDirectory.description")}</p></div><Button onClick={() => setOpen(true)}>{t("intelligence.createPerson")}</Button></div>
    <div className={styles.peopleSearch}><PeopleSearchField label={t("peopleDirectory.search")} />{state.search && <Button variant="quiet" onClick={() => state.setSearch("")}>{t("peopleDirectory.clearSearch")}</Button>}</div>
    {query.isPending ? <LoadingState label={t("common.loading")} /> : query.isError ? <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} cursorParam="peopleCursor" /> : <>
      <p className={styles.resultSummary}>{t("peopleDirectory.loaded").replace("{count}", String(query.data.items.length))}</p>
      {query.data.items.length === 0 ? <EmptyState title={t(state.search.trim() ? "peopleDirectory.noMatch" : "intelligence.noPeople")} description={t(state.search.trim() ? "peopleDirectory.noMatchHint" : "intelligence.emptyPersonAssets")} /> : <div className={styles.clusterGrid}>{query.data.items.map(person => <button className={styles.clusterCard} key={person.id} onClick={() => onOpen(person.id)}><ReviewPhotoPreview assetIds={person.previewAssetId ? [person.previewAssetId] : []} /><span className={styles.clusterBody}><strong className={styles.clusterTitle}>{person.name}</strong><span>{t("intelligence.personCounts").replace("{faces}", String(person.confirmedFaceCount)).replace("{assets}", String(person.assetCount))}</span><span className={styles.openHint}>{t("peopleDirectory.open")}<ArrowRight aria-hidden="true" size={15} /></span></span></button>)}</div>}
      <ReviewPagination nextCursor={query.data.nextCursor} cursorParam="peopleCursor" />
    </>}
    <Dialog open={open} onOpenChange={value => { if (!busy.current) setOpen(value); }} fallbackFocusRef={heading} title={t("intelligence.createPerson")} description={t("groupReview.createHint")} actions={<><DialogCloseButton onClick={() => { if (!busy.current) setOpen(false); }} /><Button disabled={!name.trim() || saving} onClick={() => void createEmpty()}>{t("intelligence.createPerson")}</Button></>}>
      <label>{t("intelligence.name")}<Input maxLength={100} value={name} disabled={saving} onChange={event => { setName(event.target.value); setUncertain(false); }} /></label>
      <p className={styles.meta}>{t("intelligence.duplicateNames")}</p>{uncertain && <InlineStatus tone="warning">{t("peopleDirectory.creationUnconfirmed")}</InlineStatus>}
    </Dialog>
  </section>;
}

function ClustersPanel({ libraryId, onOpen }: { libraryId: string; onOpen: (id: string) => void }) {
  const { t } = useLocale(); const [params, setParams] = useSearchParams(); const kind = params.get("kind") === "edge" ? "edge" : "core";
  const review = params.get("review") === "all" ? "all" : "pending";
  const query = useFaceClustersQuery(libraryId, kind, params.get("groupsCursor") ?? params.get("cursor") ?? "", review);
  const filters = <div role="group" aria-label={t("groupDirectory.filter")} className={styles.groupFilters}>{(["pending", "all"] as const).map(value => <Button key={value} size="small" variant={review === value ? "secondary" : "quiet"} aria-pressed={review === value} onClick={() => { const next = new URLSearchParams(params); next.delete("cursor"); next.delete("groupsCursor"); if (value === "all") next.set("review", value); else next.delete("review"); setParams(next); }}>{t(`groupDirectory.${value}`)}</Button>)}</div>;
  if (!libraryId || query.isPending) return <LoadingState label={t("common.loading")} />;
  if (query.isError) return <>{filters}<ReviewErrorState error={query.error} onRetry={() => void query.refetch()} cursorParam="groupsCursor" /></>;
  const counts = query.data.counts;
  return <section aria-labelledby="people-groups-title"><div className={styles.clusterToolbar}><div><h2 id="people-groups-title">{t(review === "all" ? "groupDirectory.all" : "intelligence.peopleGroups")}</h2><p>{t("intelligence.peopleGroupsDescription")}</p></div>{filters}</div>
    <p className={styles.resultSummary} role="status">{t("groupDirectory.summary").replace("{pending}", String(counts.pendingGroups)).replace("{total}", String(counts.totalGroups)).replace("{faces}", String(counts.pendingFaces))}</p>
    {!query.data.coverage.complete && <InlineStatus tone="warning">{t("intelligence.incomplete")}</InlineStatus>}{!query.data.groupAssignmentAllowed && kind === "core" && <InlineStatus tone="warning">{t("intelligence.groupAssignUnavailable")}</InlineStatus>}
    {query.data.items.length === 0 ? <EmptyState title={t(review === "pending" ? "groupDirectory.noPending" : "intelligence.none")} description={t(review === "pending" ? "groupDirectory.noPendingHint" : "groupDirectory.noGroupsHint")} /> : <><p className={styles.resultSummary}>{t("intelligence.pageGroups").replace("{count}", String(query.data.items.length))}</p><div className={styles.clusterGrid}>{query.data.items.map((cluster) => <button aria-label={`${t(`intelligence.${cluster.kind}`)}，${t("intelligence.members").replace("{count}", String(cluster.memberCount))}`} className={styles.clusterCard} key={cluster.id} onClick={() => onOpen(cluster.id)}><ReviewPhotoPreview assetIds={cluster.previewAssetIds} /><span className={styles.clusterBody}><span className={styles.clusterTitle}><UsersThree aria-hidden="true" size={19} />{t(`intelligence.${cluster.kind}`)}</span><span>{t("groupDirectory.memberSummary").replace("{pending}", String(cluster.pendingMemberCount)).replace("{total}", String(cluster.memberCount))}</span><span className={styles.openHint}>{t("intelligence.openGroup")}<ArrowRight aria-hidden="true" size={15} /></span></span></button>)}</div></>}
  <ReviewPagination nextCursor={query.data.nextCursor} cursorParam="groupsCursor" /></section>;
}


function parseView(value: string | null): ReviewView { return value === "tags" || value === "people" || value === "faces" ? value : "clusters"; }
function parseSuggestionStatus(value: string | null): "pending" | "accepted" | "dismissed" { return value === "accepted" || value === "dismissed" ? value : "pending"; }
