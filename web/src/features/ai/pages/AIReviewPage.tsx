import { useQueries, useQuery } from "@tanstack/react-query";
import { ArrowRight, ImagesSquare, SlidersHorizontal, Sparkle, UsersThree } from "@phosphor-icons/react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { ContextSidebar } from "../../../components/patterns/ContextSidebar/ContextSidebar";
import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import { Button, EmptyState, ErrorState, InlineStatus, Input, LoadingState, Select, useToast } from "../../../components/ui";
import type { AITagSuggestion, AssetFace, FaceReviewRequest, Person } from "../../../lib/api/ai";
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
  useAITagSuggestionsQuery, useApplyFaceReviewMutation, useCreatePersonMutation,
  useFaceClusterQuery, useFaceClustersQuery, usePeopleQuery, usePersonAssetsQuery,
  usePersonQuery, useRenamePersonMutation, useReviewAITagSuggestionsMutation,
} from "../queries";
import { getPerson, listAssetFaces } from "../../../lib/api/ai";
import styles from "./AIReviewPage.module.css";

type ReviewView = "tags" | "people" | "clusters" | "faces";

export function AIReviewPage({ logoutPending, onLogout, session }: {
  logoutPending?: boolean; onLogout?: () => Promise<void>; session: AuthenticatedSession;
}) {
  const { t } = useLocale();
  const [params, setParams] = useSearchParams();
  const librariesQuery = useLibrariesQuery();
  const libraries = librariesQuery.data?.pages.flatMap((page) => page.items) ?? [];
  const libraryId = params.get("libraryId") ?? libraries[0]?.id ?? "";
  const currentLibrary = libraries.find((library) => library.id === libraryId);
  const view = parseView(params.get("view"));
  const reviewingTags = view === "tags";
  const title = t(reviewingTags ? "intelligence.tags" : "intelligence.title");
  const personId = params.get("person") ?? "";
  const clusterId = params.get("cluster") ?? "";
  const assetId = params.get("asset") ?? "";
  const headingRef = useRef<HTMLHeadingElement>(null);
  const focusRoute = [view, personId, clusterId, assetId, params.get("cursor"), params.get("status")].join(":");
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
  const faceLibraryOffline = (view === "clusters" || view === "faces") && currentLibrary?.status === "offline";

  function navigate(nextView: ReviewView, extras: Record<string, string> = {}) {
    setParams({ view: nextView, ...(libraryId ? { libraryId } : {}),
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
        {view === "clusters" && <fieldset><legend>{t("intelligence.peopleGroups")}</legend>{(["core", "edge"] as const).map(kind => <Button key={kind} size="small" aria-pressed={(params.get("kind") === "edge" ? "edge" : "core") === kind} variant={(params.get("kind") === "edge" ? "edge" : "core") === kind ? "secondary" : "quiet"} onClick={() => { const next = new URLSearchParams(params); next.delete("cursor"); next.delete("cluster"); if (kind === "core") next.delete("kind"); else next.set("kind", kind); setParams(next); }}>{t(`intelligence.${kind}`)}</Button>)}</fieldset>}
      </ContextSidebar> : undefined}
      selectedPathIds={new Set<string>()}
    />}
    topbarContent={<span>{title}</span>}>
    <div className={styles.main}>
      <header className={styles.heading}><div><p className={styles.eyebrow}><Sparkle aria-hidden="true" size={16} />{t("ai.overview.eyebrow")}</p><h1 ref={headingRef} tabIndex={-1}>{title}</h1><p>{t(reviewingTags ? "intelligence.tagsDescription" : "intelligence.description")}</p></div>
        <Link className={styles.settingsLink} to={`${paths.aiSettings}${libraryId ? `?library=${encodeURIComponent(libraryId)}` : ""}`}><SlidersHorizontal aria-hidden="true" size={18} />{t("intelligence.manageModels")}<ArrowRight aria-hidden="true" size={16} /></Link>
      </header>
      {view === "people" && <p className={styles.scopeNote}>{t("intelligence.peopleGlobalScope")}</p>}
      {libraries.length > 0 && view === "tags" && <label className={styles.libraryFilter}>{t("intelligence.library")}
        <Select value={libraryId} onChange={(event) => setParams({ view, libraryId: event.target.value })}>
          {libraries.map((library) => <option value={library.id} key={library.id}>{library.name}</option>)}
        </Select></label>}
      {librariesQuery.isError && <ErrorState message={t("ai.loadFailed")} onRetry={() => void librariesQuery.refetch()} />}
      {!librariesQuery.isPending && !librariesQuery.isError && !libraries.length && <EmptyState title={t("ai.noLibraries")} description={t("ai.noLibrariesDescription")} />}
      {faceLibraryOffline && <ErrorState message={t("intelligence.offline")} onRetry={() => void librariesQuery.refetch()} />}
      {view === "tags" && <TagReviewPanel key={`${libraryId}:${params.get("status") ?? "pending"}:${params.get("cursor") ?? ""}`} libraryId={libraryId} session={session} />}
      {view === "people" && (personId ? <PersonPanel key={personId} personId={personId} session={session} onBack={() => navigate("people")} /> : <PeoplePanel session={session} onOpen={(id) => navigate("people", { person: id })} />)}
      {view === "clusters" && !faceLibraryOffline && (clusterId ? <ClusterPanel key={`${libraryId}:${clusterId}:${params.get("cursor") ?? ""}`} clusterId={clusterId} libraryId={libraryId} session={session} onBack={() => navigate("clusters")} /> : <ClustersPanel libraryId={libraryId} onOpen={(id) => navigate("clusters", { cluster: id })} />)}
      {view === "faces" && !faceLibraryOffline && <FacesPanel key={assetId} assetId={assetId} onAsset={(id) => navigate("faces", { asset: id })} session={session} />}
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

function PeopleSidebar({ selectedId, onOpen }: { selectedId: string; onOpen: (id: string) => void }) {
  const { t } = useLocale();
  const [params, setParams] = useSearchParams();
  const q = params.get("q") ?? "";
  const query = usePeopleQuery(q.trim(), params.get("peopleCursor") ?? "");
  return <>
    <label>{t("intelligence.searchPeople")}<Input type="search" value={q} maxLength={100} onChange={event => {
      const next = new URLSearchParams(params);
      next.delete("peopleCursor");
      if (event.target.value) next.set("q", event.target.value); else next.delete("q");
      setParams(next, { replace: true });
    }} /></label>
    {query.isPending ? <LoadingState label={t("common.loading")} /> : query.isError ? <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} cursorParam="peopleCursor" /> : query.data.items.length === 0 ? <p>{t("intelligence.noPeople")}</p> : <nav aria-label={t("context.peopleList")}>
      {query.data.items.map(person => <Button key={person.id} size="small" variant={person.id === selectedId ? "secondary" : "quiet"} aria-current={person.id === selectedId ? "page" : undefined} onClick={() => onOpen(person.id)}>
        <span>{person.name}</span><small>{person.assetCount}</small>
      </Button>)}
    </nav>}
    <ReviewPagination nextCursor={query.data?.nextCursor} cursorParam="peopleCursor" />
  </>;
}

function PeoplePanel({ onOpen, session }: { onOpen: (id: string) => void; session: AuthenticatedSession }) {
  const [name, setName] = useState(""); const create = useCreatePersonMutation(); const toast = useToast();
  async function createEmpty() {
    try {
      const person = await create.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), name: name.trim() });
      onOpen(person.id);
    } catch { toast.show({ message: t("intelligence.actionFailed"), tone: "danger" }); }
  }
  const { t } = useLocale(); const [params] = useSearchParams(); const queryText = params.get("q") ?? ""; const query = usePeopleQuery(queryText.trim(), params.get("peopleCursor") ?? "");
  if (query.isError) return <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return <section><form className={styles.actions} onSubmit={(event) => { event.preventDefault(); if (name.trim() && !create.isPending) void createEmpty(); }}>
    <Input aria-label={t("intelligence.name")} maxLength={100} value={name} onChange={(event) => setName(event.target.value)} />
    <Button type="submit" disabled={!name.trim() || create.isPending}>{t("intelligence.createPerson")}</Button>
  </form><p className={styles.meta}>{t("intelligence.duplicateNames")}</p>
    {query.isPending ? <LoadingState label={t("common.loading")} /> : query.data.items.length === 0 ? <EmptyState title={t("intelligence.noPeople")} description={t("intelligence.emptyPersonAssets")} /> : <div className={styles.grid}>{query.data.items.map((person) => <button className={styles.card} key={person.id} onClick={() => onOpen(person.id)}><h2>{person.name}</h2><p>{t("intelligence.personCounts").replace("{faces}", String(person.confirmedFaceCount)).replace("{assets}", String(person.assetCount))}</p><span className={styles.meta}>…{person.id.slice(-6)}</span></button>)}</div>}
  </section>;
}

function ClustersPanel({ libraryId, onOpen }: { libraryId: string; onOpen: (id: string) => void }) {
  const { t } = useLocale(); const [params] = useSearchParams(); const kind = params.get("kind") === "edge" ? "edge" : "core"; const query = useFaceClustersQuery(libraryId, kind, params.get("cursor") ?? "");
  if (!libraryId || query.isPending) return <LoadingState label={t("common.loading")} />;
  if (query.isError) return <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return <section aria-labelledby="people-groups-title"><div className={styles.clusterToolbar}><div><h2 id="people-groups-title">{t("intelligence.peopleGroups")}</h2><p>{t("intelligence.peopleGroupsDescription")}</p></div></div>
    {!query.data.coverage.complete && <InlineStatus tone="warning">{t("intelligence.incomplete")}</InlineStatus>}{!query.data.groupAssignmentAllowed && kind === "core" && <InlineStatus tone="warning">{t("intelligence.groupAssignUnavailable")}</InlineStatus>}
    {query.data.items.length === 0 ? <EmptyState title={t("intelligence.none")} description={t("intelligence.incomplete")} /> : <><p className={styles.resultSummary}>{t("intelligence.pageGroups").replace("{count}", String(query.data.items.length))}</p><div className={styles.clusterGrid}>{query.data.items.map((cluster) => <button aria-label={`${t(`intelligence.${cluster.kind}`)}，${t("intelligence.members").replace("{count}", String(cluster.memberCount))}`} className={styles.clusterCard} key={cluster.id} onClick={() => onOpen(cluster.id)}><ClusterPreview assetIds={cluster.previewAssetIds} /><span className={styles.clusterBody}><span className={styles.clusterTitle}><UsersThree aria-hidden="true" size={19} />{t(`intelligence.${cluster.kind}`)}</span><span>{t("intelligence.members").replace("{count}", String(cluster.memberCount))}</span><span className={styles.openHint}>{t("intelligence.openGroup")}<ArrowRight aria-hidden="true" size={15} /></span></span></button>)}</div></>}
  <ReviewPagination nextCursor={query.data?.nextCursor} /></section>;
}

function ClusterPreview({ assetIds }: { assetIds: string[] }) {
  const previewIds = assetIds.slice(0, 4);
  if (!previewIds.length) return <span className={styles.clusterPlaceholder}><ImagesSquare aria-hidden="true" size={36} /></span>;
  return <span className={`${styles.clusterPreview} ${styles[`previewCount${previewIds.length}`] ?? ""}`} aria-hidden="true">
    {previewIds.map((assetId) => <img alt="" key={assetId} loading="lazy" src={`/api/v1/assets/${encodeURIComponent(assetId)}/thumbnail?variant=grid`} />)}
  </span>;
}

function ClusterPanel({ clusterId, libraryId, onBack, session }: { clusterId: string; libraryId: string; onBack: () => void; session: AuthenticatedSession }) {
  const { t } = useLocale(); const toast = useToast(); const [params] = useSearchParams(); const query = useFaceClusterQuery(libraryId, clusterId, params.get("cursor") ?? ""); const corePolicy = useFaceClustersQuery(libraryId, "core"); const [peopleSearch, setPeopleSearch] = useState(""); const people = usePeopleQuery(peopleSearch.trim()); const create = useCreatePersonMutation(); const review = useApplyFaceReviewMutation(); const [selected, setSelected] = useState<string[]>([]); const [name, setName] = useState(""); const [personId, setPersonId] = useState("");
  if (query.isPending) return <LoadingState label={t("common.loading")} />; if (query.isError) return <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} />;
  async function createFromCluster() { try { await create.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), name, sourceCluster: query.data!.cluster }); toast.show({ message: t("intelligence.saved"), tone: "success" }); } catch { toast.show({ message: t("intelligence.actionFailed"), tone: "danger" }); void query.refetch(); } }
  async function applyToSelected(action: "exclude" | "assign") {
    const members = query.data!.items.filter((item) => selected.includes(item.faceId));
    try {
      for (const member of members) {
        const request: FaceReviewRequest = action === "exclude"
          ? { action: "exclude_face", faceId: member.faceId, expectedFaceRevision: member.revision }
          : { action: "assign_face", faceId: member.faceId, expectedFaceRevision: member.revision, personId, expectedPersonRevision: (await getPerson(personId)).revision };
        await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: request });
        setSelected((ids) => ids.filter((id) => id !== member.faceId));
      }
      toast.show({ message: t("intelligence.saved"), tone: "success" });
    } catch { toast.show({ message: t("intelligence.revisionConflict"), tone: "danger" }); }
    finally { void query.refetch(); void people.refetch(); }
  }
  return <section><Button variant="quiet" onClick={onBack}>{t("intelligence.back")}</Button><h2>{t(`intelligence.${query.data.cluster.kind}`)} · {t("intelligence.members").replace("{count}", String(query.data.cluster.memberCount))}</h2>
    {!corePolicy.data?.groupAssignmentAllowed && <InlineStatus tone="warning">{t("intelligence.groupAssignUnavailable")}</InlineStatus>}
    <div className={styles.actions}><Input aria-label={t("intelligence.name")} value={name} onChange={(e) => setName(e.target.value)} /><Button disabled={!name.trim() || query.data.cluster.kind !== "core" || !corePolicy.data?.groupAssignmentAllowed} onClick={() => void createFromCluster()}>{t("intelligence.createPerson")}</Button><Input aria-label={t("intelligence.searchPeople")} placeholder={t("intelligence.searchPeople")} value={peopleSearch} onChange={(event) => { setPeopleSearch(event.target.value); setPersonId(""); }} /><Select aria-label={t("intelligence.move")} value={personId} onChange={(e) => setPersonId(e.target.value)}><option value="">{t("intelligence.move")}</option>{people.data?.items.map((p) => <option value={p.id} key={p.id}>{p.name} · …{p.id.slice(-6)}</option>)}</Select><Button disabled={!selected.length || !personId || review.isPending} onClick={() => void applyToSelected("assign")}>{t("intelligence.mergeInto")}</Button><Button disabled={!selected.length || review.isPending} variant="secondary" onClick={() => void applyToSelected("exclude")}>{t("intelligence.exclude")}</Button></div>
    <div className={styles.grid}>{query.data.items.map((member) => <label className={styles.card} key={member.faceId}><input type="checkbox" checked={selected.includes(member.faceId)} onChange={(e) => setSelected((ids) => e.target.checked ? [...ids, member.faceId] : ids.filter((id) => id !== member.faceId))} /><div className={styles.faceStage}><img src={assetContentUrl(member.assetId)} alt="" /><span aria-hidden="true" className={styles.faceBox} style={{ left: `${member.region.xPercent}%`, top: `${member.region.yPercent}%`, width: `${member.region.widthPercent}%`, height: `${member.region.heightPercent}%` }} /></div><span>{t(`intelligence.${member.kind}`)} · …{member.faceId.slice(-6)}</span></label>)}</div>
  <ReviewPagination nextCursor={query.data?.nextCursor} /></section>;
}

function FacesPanel({ assetId, onAsset, session }: { assetId: string; onAsset: (id: string) => void; session: AuthenticatedSession }) {
  const { t } = useLocale(); const [draft, setDraft] = useState(assetId); const [selected, setSelected] = useState(""); const [personId, setPersonId] = useState(""); const toast = useToast(); const [peopleSearch, setPeopleSearch] = useState(""); const people = usePeopleQuery(peopleSearch.trim()); const review = useApplyFaceReviewMutation();
  const facesQuery = useQuery({ enabled: Boolean(assetId), queryKey: ["ai", "asset-faces", assetId], queryFn: () => listAssetFaces(assetId) }); const faces = facesQuery.data?.items ?? [];
  async function apply(action: "exclude" | "assign") {
    const face = faces.find((item) => item.faceId === selected);
    const person = people.data?.items.find((item) => item.id === personId);
    if (!face) return;
    try {
      const source = action === "assign" && face.personId && face.personId !== personId ? await getPerson(face.personId) : undefined;
      const request: FaceReviewRequest = action === "exclude"
        ? { action: "exclude_face", faceId: face.faceId, expectedFaceRevision: face.revision }
        : { action: "assign_face", faceId: face.faceId, expectedFaceRevision: face.revision, personId, expectedPersonRevision: person?.revision ?? 0,
          ...(source ? { sourcePersonId: source.id, expectedSourceRevision: source.revision } : {}) };
      await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: request });
      toast.show({ message: t("intelligence.saved"), tone: "success" });
    } catch { toast.show({ message: t("intelligence.actionFailed"), tone: "danger" }); }
    finally { void facesQuery.refetch(); void people.refetch(); }
  }
  return <section><div className={styles.actions}><Input aria-label={t("intelligence.assetId")} value={draft} onChange={(e) => setDraft(e.target.value)} /><Button disabled={!draft.trim()} onClick={() => onAsset(draft.trim())}>{t("intelligence.showFaces")}</Button></div>
    {facesQuery.isError && <ReviewErrorState error={facesQuery.error} onRetry={() => void facesQuery.refetch()} />}{assetId && facesQuery.isPending && <LoadingState label={t("common.loading")} />}{assetId && !facesQuery.isPending && !facesQuery.isError && faces.length === 0 && <EmptyState title={t("intelligence.none")} description={t("intelligence.emptyFaces")} />}{assetId && faces.length > 0 && <><p>{t("intelligence.faceHint")}</p><div className={styles.faceStage} aria-label={t("intelligence.faceSelector")}><img src={assetContentUrl(assetId)} alt="" />{faces.map((face) => <FaceBox face={face} key={face.faceId} selected={selected === face.faceId} onSelect={() => setSelected(face.faceId)} label={t("intelligence.faceNumber").replace("{number}", String(face.ordinal)).replace("{state}", face.state)} />)}</div><div className={styles.faceList} aria-label={t("intelligence.faceAlternative")}>{faces.map((face) => <Button key={face.faceId} variant={selected === face.faceId ? "secondary" : "quiet"} onClick={() => setSelected(face.faceId)}>{t("intelligence.faceNumber").replace("{number}", String(face.ordinal)).replace("{state}", face.state)}</Button>)}</div><div className={styles.actions}><Input aria-label={t("intelligence.searchPeople")} placeholder={t("intelligence.searchPeople")} value={peopleSearch} onChange={(event) => { setPeopleSearch(event.target.value); setPersonId(""); }} /><Select aria-label={t("intelligence.move")} value={personId} onChange={(e) => setPersonId(e.target.value)}><option value="">{t("intelligence.move")}</option>{people.data?.items.map((p) => <option value={p.id} key={p.id}>{p.name} · …{p.id.slice(-6)}</option>)}</Select><Button disabled={!selected || !personId || review.isPending} onClick={() => void apply("assign")}>{t("intelligence.move")}</Button><Button disabled={!selected || review.isPending} variant="secondary" onClick={() => void apply("exclude")}>{t("intelligence.exclude")}</Button></div></>}
  </section>;
}

function FaceBox({ face, label, onSelect, selected }: { face: AssetFace; label: string; onSelect: () => void; selected: boolean }) { return <button type="button" className={styles.faceBox} aria-label={label} aria-pressed={selected} onClick={onSelect} style={{ left: `${face.region.xPercent}%`, top: `${face.region.yPercent}%`, width: `${face.region.widthPercent}%`, height: `${face.region.heightPercent}%` }} />; }

function PersonPanel({ onBack, personId, session }: { onBack: () => void; personId: string; session: AuthenticatedSession }) {
  const { t } = useLocale(); const toast = useToast(); const person = usePersonQuery(personId); const [params] = useSearchParams(); const assets = usePersonAssetsQuery(personId, params.get("cursor") ?? ""); const [peopleSearch, setPeopleSearch] = useState(""); const people = usePeopleQuery(peopleSearch.trim()); const rename = useRenamePersonMutation(); const review = useApplyFaceReviewMutation(); const [name, setName] = useState(""); const [targetId, setTargetId] = useState("");
  const assetItems = useMemo(() => assets.data?.items ?? [], [assets.data]);
  const backRef = useRef<HTMLButtonElement>(null);
  const previousAssetCount = useRef(assetItems.length);
  useLayoutEffect(() => {
    if (assetItems.length < previousAssetCount.current && document.activeElement === document.body) backRef.current?.focus();
    previousAssetCount.current = assetItems.length;
  }, [assetItems.length]);
  const faceQueries = useQueries({ queries: assetItems.map(({ asset }) => ({ queryKey: ["ai", "asset-faces", asset.id], queryFn: () => listAssetFaces(asset.id) })) });
  if (person.isPending) return <LoadingState label={t("common.loading")} />; if (person.isError) return <ReviewErrorState error={person.error} onRetry={() => void person.refetch()} />;
  async function save() { try { await rename.mutateAsync({ csrfToken: session.csrfToken, etag: person.data!.etag, name: name.trim(), personId }); setName(""); toast.show({ message: t("intelligence.saved"), tone: "success" }); } catch (error) { toast.show({ message: error instanceof ApiError && error.code === "precondition_failed" ? t("intelligence.revisionConflict") : t("intelligence.actionFailed"), tone: "danger" }); void person.refetch(); } }
  async function faceAction(index: number, mode: "remove" | "move") { const face = faceQueries[index]?.data?.items.find((item) => item.personId === personId); const target = people.data?.items.find((item) => item.id === targetId); if (!face) return; const request: FaceReviewRequest = mode === "remove" ? { action: "split_face", faceId: face.faceId, expectedFaceRevision: face.revision, sourcePersonId: personId, expectedSourceRevision: person.data!.revision } : { action: "assign_face", faceId: face.faceId, expectedFaceRevision: face.revision, personId: targetId, expectedPersonRevision: target?.revision ?? 0, sourcePersonId: personId, expectedSourceRevision: person.data!.revision }; try { await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: request }); toast.show({ message: t("intelligence.saved"), tone: "success" }); void assets.refetch(); } catch { toast.show({ message: t("intelligence.revisionConflict"), tone: "danger" }); void person.refetch(); } }
  async function merge() { const target = people.data?.items.find((item) => item.id === targetId); if (!target) return; try { await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: { action: "merge_people", sourcePersonId: personId, targetPersonId: target.id, expectedSourceRevision: person.data!.revision, expectedTargetRevision: target.revision, conflictsAcknowledged: true } }); toast.show({ message: t("intelligence.saved"), tone: "success" }); onBack(); } catch { toast.show({ message: t("intelligence.revisionConflict"), tone: "danger" }); void person.refetch(); } }
  return <section><Button ref={backRef} variant="quiet" onClick={onBack}>{t("intelligence.back")}</Button><h2>{person.data.name} <span className={styles.meta}>…{person.data.id.slice(-6)}</span></h2><p>{t("intelligence.personCounts").replace("{faces}", String(person.data.confirmedFaceCount)).replace("{assets}", String(person.data.assetCount))}</p><div className={styles.actions}><Input placeholder={person.data.name} aria-label={t("intelligence.name")} value={name} onChange={(e) => setName(e.target.value)} /><Button disabled={!name.trim() || rename.isPending} onClick={() => void save()}>{t("intelligence.rename")}</Button><Input aria-label={t("intelligence.searchPeople")} placeholder={t("intelligence.searchPeople")} value={peopleSearch} onChange={(event) => { setPeopleSearch(event.target.value); setTargetId(""); }} /><Select aria-label={t("intelligence.move")} value={targetId} onChange={(e) => setTargetId(e.target.value)}><option value="">{t("intelligence.move")}</option>{people.data?.items.filter((item) => item.id !== personId).map((item) => <option value={item.id} key={item.id}>{item.name} · …{item.id.slice(-6)}</option>)}</Select><Button disabled={!targetId || review.isPending} variant="secondary" onClick={() => void merge()}>{t("intelligence.mergePeople")}</Button></div><h3>{t("intelligence.personAssets")}</h3>{assets.isPending ? <LoadingState label={t("common.loading")} /> : assets.isError ? <ReviewErrorState error={assets.error} onRetry={() => void assets.refetch()} /> : assetItems.length === 0 ? <EmptyState title={t("intelligence.none")} description={t("intelligence.emptyPersonAssets")} /> : <div className={styles.grid}>{assetItems.map(({ asset, faceIds }, index) => <article className={styles.card} key={asset.id}><Link className={styles.personLink} to={paths.media(asset.libraryId, asset.id)}><img className={styles.media} src={asset.thumbnail.url ?? assetContentUrl(asset.id)} alt={asset.name} /><strong>{asset.name}</strong><span className={styles.meta}>{t(faceIds.length === 1 ? "intelligence.assetFaceOne" : "intelligence.assetFaceMany").replace("{count}", String(faceIds.length))} · …{asset.id.slice(-6)}</span></Link><div className={styles.actions}><Button disabled={!targetId || !faceQueries[index]?.data} onClick={() => void faceAction(index, "move")}>{t("intelligence.move")}</Button><Button disabled={!faceQueries[index]?.data} variant="quiet" onClick={() => void faceAction(index, "remove")}>{t("intelligence.removeAssignment")}</Button></div></article>)}</div>}<ReviewPagination nextCursor={assets.data?.nextCursor} /></section>;
}

function parseView(value: string | null): ReviewView { return value === "tags" || value === "people" || value === "faces" ? value : "clusters"; }
function parseSuggestionStatus(value: string | null): "pending" | "accepted" | "dismissed" { return value === "accepted" || value === "dismissed" ? value : "pending"; }

function ReviewPagination({ nextCursor, cursorParam = "cursor" }: { nextCursor: string | null | undefined; cursorParam?: "cursor" | "peopleCursor" }) {
 const { t } = useLocale(); const [params, setParams] = useSearchParams();
 if (!nextCursor && !params.has(cursorParam)) return null;
 function go(cursor?: string) { const next = new URLSearchParams(params); if (cursor) next.set(cursorParam, cursor); else next.delete(cursorParam); setParams(next); }
 return <div className={styles.actions}>
  <Button variant="quiet" disabled={!params.has(cursorParam)} onClick={() => go()}>{t("intelligence.firstPage")}</Button>
  <Button disabled={!nextCursor} onClick={() => go(nextCursor!)}>{t("intelligence.nextPage")}</Button>
 </div>;
}

function ReviewErrorState({ error, onRetry, cursorParam = "cursor" }: { error: unknown; onRetry: () => void; cursorParam?: "cursor" | "peopleCursor" }) {
 const { t } = useLocale();
 const [params, setParams] = useSearchParams();
 const code = error instanceof ApiError ? error.code : "";
 const message = code === "face_not_ready" || code === "face_disabled" ? t("intelligence.notReady")
   : code === "library_offline" ? t("intelligence.offline")
   : code === "precondition_failed" || code === "face_constraint_conflict" || code === "face_cursor_stale" || code === "suggestion_cursor_stale" ? t("intelligence.revisionConflict")
   : t("intelligence.loadFailed");
 return <ErrorState message={message} onRetry={() => {
  if ((code === "face_cursor_stale" || code === "suggestion_cursor_stale") && params.has(cursorParam)) { const next = new URLSearchParams(params); next.delete(cursorParam); setParams(next); }
  else onRetry();
 }} />;
}
