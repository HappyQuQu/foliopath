import { useQueries, useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import { Button, EmptyState, ErrorState, InlineStatus, LoadingState, useToast } from "../../../components/ui";
import type { AITagSuggestion, AssetFace, FaceReviewRequest, Person } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { assetContentUrl } from "../../../lib/api/catalog";
import { getAssetCuration } from "../../../lib/api/curation";
import { ApiError } from "../../../lib/api/errors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useLibrariesQuery } from "../../libraries";
import {
  useAITagSuggestionsQuery, useApplyFaceReviewMutation, useCreatePersonMutation,
  useFaceClusterQuery, useFaceClustersQuery, usePeopleQuery, usePersonAssetsQuery,
  usePersonQuery, useRenamePersonMutation, useReviewAITagSuggestionsMutation,
} from "../queries";
import { listAssetFaces } from "../../../lib/api/ai";
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
  const view = parseView(params.get("view"));
  const personId = params.get("person") ?? "";
  const clusterId = params.get("cluster") ?? "";
  const assetId = params.get("asset") ?? "";

  function navigate(nextView: ReviewView, extras: Record<string, string> = {}) {
    setParams({ view: nextView, ...(libraryId ? { libraryId } : {}), ...extras });
  }

  return <AppShell active="settings" homeHref={paths.root} identity={session.administrator.displayName}
    logoutPending={logoutPending} onLogout={onLogout} searchHref={paths.search}
    settingsHref={paths.generalSettings} title={t("intelligence.title")}>
    <div className={styles.main}>
      <header className={styles.heading}><h1>{t("intelligence.title")}</h1><p>{t("intelligence.description")}</p></header>
      <section className={styles.privacy} aria-labelledby="privacy-heading"><h2 id="privacy-heading">{t("intelligence.privacyTitle")}</h2><p>{t("intelligence.privacy")}</p></section>
      <div className={styles.tabs} role="tablist" aria-label={t("intelligence.title")}>
        {(["tags", "people", "clusters", "faces"] as const).map((item) => <Button key={item} role="tab"
          aria-selected={view === item} variant={view === item ? "secondary" : "quiet"} onClick={() => navigate(item)}>{t(`intelligence.${item}`)}</Button>)}
      </div>
      {libraries.length > 0 && (view === "tags" || view === "clusters") && <label className={styles.filters}>{t("intelligence.library")}
        <select className={styles.input} value={libraryId} onChange={(event) => setParams({ view, libraryId: event.target.value })}>
          {libraries.map((library) => <option value={library.id} key={library.id}>{library.name}</option>)}
        </select></label>}
      {view === "tags" && <TagReviewPanel libraryId={libraryId} session={session} />}
      {view === "people" && (personId ? <PersonPanel personId={personId} session={session} onBack={() => navigate("people")} /> : <PeoplePanel onOpen={(id) => navigate("people", { person: id })} />)}
      {view === "clusters" && (clusterId ? <ClusterPanel clusterId={clusterId} libraryId={libraryId} session={session} onBack={() => navigate("clusters")} /> : <ClustersPanel libraryId={libraryId} onOpen={(id) => navigate("clusters", { cluster: id })} />)}
      {view === "faces" && <FacesPanel assetId={assetId} onAsset={(id) => navigate("faces", { asset: id })} session={session} />}
    </div>
  </AppShell>;
}

function TagReviewPanel({ libraryId, session }: { libraryId: string; session: AuthenticatedSession }) {
  const { t } = useLocale(); const toast = useToast();
  const [params, setParams] = useSearchParams();
  const status = parseSuggestionStatus(params.get("status"));
  const [selected, setSelected] = useState<string[]>([]);
  const query = useAITagSuggestionsQuery(libraryId, status);
  const items = query.data?.items ?? [];
  const curations = useQueries({ queries: items.map((item) => ({ queryKey: ["curation", "asset", item.asset.id], queryFn: () => getAssetCuration(item.asset.id) })) });
  const review = useReviewAITagSuggestionsMutation();
  async function submit(action: "accept" | "dismiss") {
    const ready = items.flatMap((suggestion, index) => selected.includes(suggestion.id) && curations[index]?.data
      ? [{ suggestion, curationRevision: curations[index].data.revision }] : []);
    if (!ready.length) return;
    try { await review.mutateAsync({ action, csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), items: ready }); setSelected([]); toast.show({ message: t("intelligence.saved"), tone: "success" }); }
    catch (error) { toast.show({ message: error instanceof ApiError && error.code === "precondition_failed" ? t("intelligence.revisionConflict") : t("intelligence.actionFailed"), tone: "danger" }); void query.refetch(); }
  }
  if (!libraryId || query.isPending) return <LoadingState label={t("common.loading")} />;
  if (query.isError) return <ErrorState message={t("intelligence.loadFailed")} onRetry={() => void query.refetch()} />;
  const coverage = query.data.coverage;
  return <section><div className={styles.filters}>{(["pending", "accepted", "dismissed"] as const).map((item) => <Button key={item} variant={status === item ? "secondary" : "quiet"} onClick={() => { const next = new URLSearchParams(params); if (item === "pending") next.delete("status"); else next.set("status", item); setParams(next); setSelected([]); }}>{t(`intelligence.${item}`)}</Button>)}</div>
    <p className={styles.meta}>{t("intelligence.coverage").replace("{completed}", String(coverage.completed)).replace("{eligible}", String(coverage.eligible)).replace("{failed}", String(coverage.failed)).replace("{degraded}", String(coverage.degraded))}</p>
    {!coverage.complete && <InlineStatus tone="warning">{t("intelligence.incomplete")}</InlineStatus>}
    {status === "pending" && selected.length > 0 && <div className={styles.actions}><Button disabled={review.isPending} onClick={() => void submit("accept")}>{t("intelligence.accept")}</Button><Button disabled={review.isPending} variant="secondary" onClick={() => void submit("dismiss")}>{t("intelligence.dismiss")}</Button></div>}
    {items.length === 0 ? <EmptyState title={t("intelligence.noSuggestions")} description={t("intelligence.incomplete")} /> : <div className={styles.grid}>{items.map((item) => <SuggestionCard item={item} key={item.id} selected={selected.includes(item.id)} onSelect={(checked) => setSelected((current) => checked ? [...current, item.id].slice(0, 100) : current.filter((id) => id !== item.id))} />)}</div>}
  </section>;
}

function SuggestionCard({ item, onSelect, selected }: { item: AITagSuggestion; onSelect: (selected: boolean) => void; selected: boolean }) {
  const { t } = useLocale();
  return <article className={styles.card}><img className={styles.media} src={item.asset.thumbnail.url ?? assetContentUrl(item.asset.id)} alt={item.asset.name} />
    <p className={styles.meta}>{t("intelligence.aiSuggestion")}</p><h2>{item.tag.name}</h2><p>{t("intelligence.confidence").replace("{value}", String(Math.round(item.confidence * 100)))}</p>
    {item.status === "pending" && <label className={styles.check}><input type="checkbox" checked={selected} onChange={(event) => onSelect(event.target.checked)} />{t("intelligence.select").replace("{name}", item.asset.name)}</label>}
  </article>;
}

function PeoplePanel({ onOpen }: { onOpen: (id: string) => void }) {
  const { t } = useLocale(); const [params, setParams] = useSearchParams(); const queryText = params.get("q") ?? ""; const query = usePeopleQuery(queryText.trim());
  if (query.isError) return <ErrorState message={t("intelligence.loadFailed")} onRetry={() => void query.refetch()} />;
  return <section><label>{t("intelligence.searchPeople")}<input className={styles.input} value={queryText} onChange={(e) => { const next = new URLSearchParams(params); if (e.target.value) next.set("q", e.target.value); else next.delete("q"); setParams(next, { replace: true }); }} /></label><p className={styles.meta}>{t("intelligence.duplicateNames")}</p>
    {query.isPending ? <LoadingState label={t("common.loading")} /> : query.data.items.length === 0 ? <EmptyState title={t("intelligence.noPeople")} description={t("intelligence.duplicateNames")} /> : <div className={styles.grid}>{query.data.items.map((person) => <button className={styles.card} key={person.id} onClick={() => onOpen(person.id)}><h2>{person.name}</h2><p>{t("intelligence.personCounts").replace("{faces}", String(person.confirmedFaceCount)).replace("{assets}", String(person.assetCount))}</p><span className={styles.meta}>…{person.id.slice(-6)}</span></button>)}</div>}
  </section>;
}

function ClustersPanel({ libraryId, onOpen }: { libraryId: string; onOpen: (id: string) => void }) {
  const { t } = useLocale(); const [params, setParams] = useSearchParams(); const kind = params.get("kind") === "edge" ? "edge" : "core"; const query = useFaceClustersQuery(libraryId, kind);
  if (!libraryId || query.isPending) return <LoadingState label={t("common.loading")} />;
  if (query.isError) return <ErrorState message={t("intelligence.loadFailed")} onRetry={() => void query.refetch()} />;
  return <section><div className={styles.filters}>{(["core", "edge"] as const).map((item) => <Button key={item} variant={kind === item ? "secondary" : "quiet"} onClick={() => { const next = new URLSearchParams(params); if (item === "core") next.delete("kind"); else next.set("kind", item); setParams(next); }}>{t(`intelligence.${item}`)}</Button>)}</div>
    {!query.data.coverage.complete && <InlineStatus tone="warning">{t("intelligence.incomplete")}</InlineStatus>}{!query.data.groupAssignmentAllowed && kind === "core" && <InlineStatus tone="warning">{t("intelligence.groupAssignUnavailable")}</InlineStatus>}
    {query.data.items.length === 0 ? <EmptyState title={t("intelligence.none")} description={t("intelligence.incomplete")} /> : <div className={styles.grid}>{query.data.items.map((cluster) => <button className={styles.card} key={cluster.id} onClick={() => onOpen(cluster.id)}><h2>{t(`intelligence.${cluster.kind}`)}</h2><p>{t("intelligence.members").replace("{count}", String(cluster.memberCount))}</p><span className={styles.meta}>…{cluster.id.slice(-6)}</span></button>)}</div>}
  </section>;
}

function ClusterPanel({ clusterId, libraryId, onBack, session }: { clusterId: string; libraryId: string; onBack: () => void; session: AuthenticatedSession }) {
  const { t } = useLocale(); const toast = useToast(); const query = useFaceClusterQuery(libraryId, clusterId); const corePolicy = useFaceClustersQuery(libraryId, "core"); const people = usePeopleQuery(""); const create = useCreatePersonMutation(); const review = useApplyFaceReviewMutation(); const [selected, setSelected] = useState<string[]>([]); const [name, setName] = useState(""); const [personId, setPersonId] = useState("");
  if (query.isPending) return <LoadingState label={t("common.loading")} />; if (query.isError) return <ErrorState message={t("intelligence.loadFailed")} onRetry={() => void query.refetch()} />;
  async function createFromCluster() { try { await create.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), name, sourceCluster: query.data!.cluster }); toast.show({ message: t("intelligence.saved"), tone: "success" }); } catch { toast.show({ message: t("intelligence.actionFailed"), tone: "danger" }); void query.refetch(); } }
  async function applyToSelected(action: "exclude" | "assign") { const members = query.data!.items.filter((item) => selected.includes(item.faceId)); for (const member of members) { const request: FaceReviewRequest = action === "exclude" ? { action: "ExcludeFaceReview", faceId: member.faceId, expectedFaceRevision: member.revision } : { action: "AssignFaceReview", faceId: member.faceId, expectedFaceRevision: member.revision, personId, expectedPersonRevision: people.data?.items.find((p) => p.id === personId)?.revision ?? 0 }; await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: request }); } setSelected([]); void query.refetch(); }
  return <section><Button variant="quiet" onClick={onBack}>{t("intelligence.back")}</Button><h2>{t(`intelligence.${query.data.cluster.kind}`)} · {t("intelligence.members").replace("{count}", String(query.data.cluster.memberCount))}</h2>
    {!corePolicy.data?.groupAssignmentAllowed && <InlineStatus tone="warning">{t("intelligence.groupAssignUnavailable")}</InlineStatus>}
    <div className={styles.actions}><input className={styles.input} aria-label={t("intelligence.name")} value={name} onChange={(e) => setName(e.target.value)} /><Button disabled={!name.trim() || query.data.cluster.kind !== "core" || !corePolicy.data?.groupAssignmentAllowed} onClick={() => void createFromCluster()}>{t("intelligence.createPerson")}</Button><select className={styles.input} value={personId} onChange={(e) => setPersonId(e.target.value)}><option value="">{t("intelligence.move")}</option>{people.data?.items.map((p) => <option value={p.id} key={p.id}>{p.name} · …{p.id.slice(-6)}</option>)}</select><Button disabled={!selected.length || !personId} onClick={() => void applyToSelected("assign")}>{t("intelligence.mergeInto")}</Button><Button disabled={!selected.length} variant="secondary" onClick={() => void applyToSelected("exclude")}>{t("intelligence.exclude")}</Button></div>
    <div className={styles.grid}>{query.data.items.map((member) => <label className={styles.card} key={member.faceId}><input type="checkbox" checked={selected.includes(member.faceId)} onChange={(e) => setSelected((ids) => e.target.checked ? [...ids, member.faceId] : ids.filter((id) => id !== member.faceId))} /><img className={styles.media} src={assetContentUrl(member.assetId)} alt="" /><span>{t(`intelligence.${member.kind}`)} · …{member.faceId.slice(-6)}</span></label>)}</div>
  </section>;
}

function FacesPanel({ assetId, onAsset, session }: { assetId: string; onAsset: (id: string) => void; session: AuthenticatedSession }) {
  const { t } = useLocale(); const [draft, setDraft] = useState(assetId); const [selected, setSelected] = useState(""); const [personId, setPersonId] = useState(""); const toast = useToast(); const people = usePeopleQuery(""); const review = useApplyFaceReviewMutation();
  const facesQuery = useQuery({ enabled: Boolean(assetId), queryKey: ["ai", "asset-faces", assetId], queryFn: () => listAssetFaces(assetId) }); const faces = facesQuery.data?.items ?? [];
  async function apply(action: "exclude" | "assign") { const face = faces.find((item) => item.faceId === selected); const person = people.data?.items.find((item) => item.id === personId); if (!face) return; const request: FaceReviewRequest = action === "exclude" ? { action: "ExcludeFaceReview", faceId: face.faceId, expectedFaceRevision: face.revision } : { action: "AssignFaceReview", faceId: face.faceId, expectedFaceRevision: face.revision, personId, expectedPersonRevision: person?.revision ?? 0 }; try { await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: request }); toast.show({ message: t("intelligence.saved"), tone: "success" }); void facesQuery.refetch(); } catch { toast.show({ message: t("intelligence.actionFailed"), tone: "danger" }); } }
  return <section><div className={styles.actions}><input className={styles.input} aria-label={t("intelligence.assetId")} value={draft} onChange={(e) => setDraft(e.target.value)} /><Button disabled={!draft.trim()} onClick={() => onAsset(draft.trim())}>{t("intelligence.showFaces")}</Button></div>
    {facesQuery.isError && <ErrorState message={t("intelligence.loadFailed")} onRetry={() => void facesQuery.refetch()} />}{assetId && facesQuery.isPending && <LoadingState label={t("common.loading")} />}{assetId && faces.length > 0 && <><p>{t("intelligence.faceHint")}</p><div className={styles.faceStage} aria-label={t("intelligence.faceSelector")}><img src={assetContentUrl(assetId)} alt="" />{faces.map((face) => <FaceBox face={face} key={face.faceId} selected={selected === face.faceId} onSelect={() => setSelected(face.faceId)} label={t("intelligence.faceNumber").replace("{number}", String(face.ordinal)).replace("{state}", face.state)} />)}</div><div className={styles.faceList} aria-label={t("intelligence.faceAlternative")}>{faces.map((face) => <Button key={face.faceId} variant={selected === face.faceId ? "secondary" : "quiet"} onClick={() => setSelected(face.faceId)}>{t("intelligence.faceNumber").replace("{number}", String(face.ordinal)).replace("{state}", face.state)}</Button>)}</div><div className={styles.actions}><select className={styles.input} value={personId} onChange={(e) => setPersonId(e.target.value)}><option value="">{t("intelligence.move")}</option>{people.data?.items.map((p) => <option value={p.id} key={p.id}>{p.name} · …{p.id.slice(-6)}</option>)}</select><Button disabled={!selected || !personId} onClick={() => void apply("assign")}>{t("intelligence.move")}</Button><Button disabled={!selected} variant="secondary" onClick={() => void apply("exclude")}>{t("intelligence.exclude")}</Button></div></>}
  </section>;
}

function FaceBox({ face, label, onSelect, selected }: { face: AssetFace; label: string; onSelect: () => void; selected: boolean }) { return <button type="button" className={styles.faceBox} aria-label={label} aria-pressed={selected} onClick={onSelect} style={{ left: `${face.region.xPercent}%`, top: `${face.region.yPercent}%`, width: `${face.region.widthPercent}%`, height: `${face.region.heightPercent}%` }} />; }

function PersonPanel({ onBack, personId, session }: { onBack: () => void; personId: string; session: AuthenticatedSession }) {
  const { t } = useLocale(); const toast = useToast(); const person = usePersonQuery(personId); const assets = usePersonAssetsQuery(personId); const people = usePeopleQuery(""); const rename = useRenamePersonMutation(); const review = useApplyFaceReviewMutation(); const [name, setName] = useState(""); const [targetId, setTargetId] = useState("");
  const assetItems = useMemo(() => assets.data?.items ?? [], [assets.data]);
  const faceQueries = useQueries({ queries: assetItems.map(({ asset }) => ({ queryKey: ["ai", "asset-faces", asset.id], queryFn: () => listAssetFaces(asset.id) })) });
  if (person.isPending) return <LoadingState label={t("common.loading")} />; if (person.isError) return <ErrorState message={t("intelligence.loadFailed")} onRetry={() => void person.refetch()} />;
  async function save() { try { await rename.mutateAsync({ csrfToken: session.csrfToken, etag: person.data!.etag, name: name.trim(), personId }); setName(""); toast.show({ message: t("intelligence.saved"), tone: "success" }); } catch (error) { toast.show({ message: error instanceof ApiError && error.code === "precondition_failed" ? t("intelligence.revisionConflict") : t("intelligence.actionFailed"), tone: "danger" }); void person.refetch(); } }
  async function faceAction(index: number, mode: "remove" | "move") { const face = faceQueries[index]?.data?.items.find((item) => item.personId === personId); const target = people.data?.items.find((item) => item.id === targetId); if (!face) return; const request: FaceReviewRequest = mode === "remove" ? { action: "SplitFaceReview", faceId: face.faceId, expectedFaceRevision: face.revision, sourcePersonId: personId, expectedSourceRevision: person.data!.revision } : { action: "AssignFaceReview", faceId: face.faceId, expectedFaceRevision: face.revision, personId: targetId, expectedPersonRevision: target?.revision ?? 0 }; try { await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: request }); toast.show({ message: t("intelligence.saved"), tone: "success" }); void assets.refetch(); } catch { toast.show({ message: t("intelligence.revisionConflict"), tone: "danger" }); void person.refetch(); } }
  async function merge() { const target = people.data?.items.find((item) => item.id === targetId); if (!target) return; try { await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: createRequestKey(), review: { action: "MergePeopleReview", sourcePersonId: personId, targetPersonId: target.id, expectedSourceRevision: person.data!.revision, expectedTargetRevision: target.revision, conflictsAcknowledged: true } }); toast.show({ message: t("intelligence.saved"), tone: "success" }); onBack(); } catch { toast.show({ message: t("intelligence.revisionConflict"), tone: "danger" }); void person.refetch(); } }
  return <section><Button variant="quiet" onClick={onBack}>{t("intelligence.back")}</Button><h2>{person.data.name} <span className={styles.meta}>…{person.data.id.slice(-6)}</span></h2><p>{t("intelligence.personCounts").replace("{faces}", String(person.data.confirmedFaceCount)).replace("{assets}", String(person.data.assetCount))}</p><div className={styles.actions}><input className={styles.input} placeholder={person.data.name} aria-label={t("intelligence.name")} value={name} onChange={(e) => setName(e.target.value)} /><Button disabled={!name.trim()} onClick={() => void save()}>{t("intelligence.rename")}</Button><select className={styles.input} value={targetId} onChange={(e) => setTargetId(e.target.value)}><option value="">{t("intelligence.move")}</option>{people.data?.items.filter((item) => item.id !== personId).map((item) => <option value={item.id} key={item.id}>{item.name} · …{item.id.slice(-6)}</option>)}</select><Button disabled={!targetId} variant="secondary" onClick={() => void merge()}>{t("intelligence.mergePeople")}</Button></div><h3>{t("intelligence.personAssets")}</h3>{assets.isPending ? <LoadingState label={t("common.loading")} /> : <div className={styles.grid}>{assetItems.map(({ asset, faceIds }, index) => <article className={styles.card} key={asset.id}><Link className={styles.personLink} to={paths.media(asset.libraryId, asset.id)}><img className={styles.media} src={asset.thumbnail.url ?? assetContentUrl(asset.id)} alt={asset.name} /><strong>{asset.name}</strong><span className={styles.meta}>{faceIds.length} face · …{asset.id.slice(-6)}</span></Link><div className={styles.actions}><Button disabled={!targetId || !faceQueries[index]?.data} onClick={() => void faceAction(index, "move")}>{t("intelligence.move")}</Button><Button disabled={!faceQueries[index]?.data} variant="quiet" onClick={() => void faceAction(index, "remove")}>{t("intelligence.removeAssignment")}</Button></div></article>)}</div>}</section>;
}

function parseView(value: string | null): ReviewView { return value === "people" || value === "clusters" || value === "faces" ? value : "tags"; }
function parseSuggestionStatus(value: string | null): "pending" | "accepted" | "dismissed" { return value === "accepted" || value === "dismissed" ? value : "pending"; }
