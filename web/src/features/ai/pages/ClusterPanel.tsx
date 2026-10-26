import { useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Button, Dialog, DialogCloseButton, EmptyState, InlineStatus, Input, LoadingState, RadioOption, Select } from "../../../components/ui";
import type { FaceClusterDetailPage, FaceReviewRequest, Person } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { assetContentUrl } from "../../../lib/api/catalog";
import { isFaceReviewConflict } from "../reviewErrors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useApplyFaceReviewMutation, useAssetFacesQueries, useCreatePersonMutation, useFaceClusterQuery, usePeopleQuery, usePersonQuery } from "../queries";
import { FaceSelectionBox } from "./FaceSelectionBox";
import { usePersonCreationIntent } from "./usePersonCreationIntent";
import { ReviewErrorState, ReviewPagination } from "./ReviewFeedback";
import styles from "./ClusterPanel.module.css";
import type { FaceReviewCallbacks } from "./FaceReviewUndo";

type Member = FaceClusterDetailPage["items"][number];
type Confirmation = { request: FaceReviewRequest; key: string; label: string; targetName?: string };
export function ClusterPanel({ clusterId, libraryId, onBack, session, onReviewStart, onReviewCommitted }: { clusterId: string; libraryId: string; onBack: () => void; session: AuthenticatedSession } & FaceReviewCallbacks) {
  const { t } = useLocale(); const [params] = useSearchParams();
  const query = useFaceClusterQuery(libraryId, clusterId, params.get("cursor") ?? "");
  const [selected, setSelected] = useState("");
  const [search, setSearch] = useState("");
  const people = usePeopleQuery(search.trim());
  const [target, setTarget] = useState<Person | null>(null);
  const targetQuery = usePersonQuery(target?.id ?? "");
  const [confirmation, setConfirmation] = useState<Confirmation | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false); const busy = useRef(false);
  const [result, setResult] = useState<{ key: "created" | "saved" | "conflict" | "unconfirmed"; name?: string } | null>(null);
  const createIntent = usePersonCreationIntent();
  const create = useCreatePersonMutation(); const review = useApplyFaceReviewMutation();
  const heading = useRef<HTMLHeadingElement>(null);
  const items = query.data?.items ?? [];
  const photos = new Map<string, Member[]>();
  for (const member of items) photos.set(member.assetId, [...(photos.get(member.assetId) ?? []), member]);
  const photoIds = [...photos.keys()];
  const faceQueries = useAssetFacesQueries(photoIds);
  function faceState(member: Member) { return faceQueries[photoIds.indexOf(member.assetId)]?.data?.items.find(face => face.faceId === member.faceId); }
  function eligible(member: Member) {
    const faceQuery = faceQueries[photoIds.indexOf(member.assetId)]; const state = faceState(member);
    return !!state && ["anonymous_core", "anonymous_edge", "unclustered"].includes(state.state) && state.revision === member.revision && !faceQuery?.isFetching && !faceQuery?.isError;
  }
  const chosen = items.find(member => member.faceId === selected && eligible(member));
  const fresh = !query.isFetching && !query.isError && !query.isPending;
  const choices = target && !people.data?.items.some(person => person.id === target.id) ? [target, ...(people.data?.items ?? [])] : people.data?.items ?? [];
  function photoLabel(photo: number, mark: number) { return t("groupReview.markerLabel").replace("{photo}", String(photo)).replace("{mark}", String(mark)); }
  function openConfirmation(action: "assign" | "exclude") {
    const destination = targetQuery.data;
    if (!chosen || !fresh || busy.current || (action === "assign" && (!target || !destination || destination.id !== target.id || targetQuery.isFetching || targetQuery.isError))) return;
    const entries = [...photos.entries()]; const index = entries.findIndex(([id]) => id === chosen.assetId);
    const mark = entries[index]![1].findIndex(member => member.faceId === chosen.faceId) + 1;
    const request: FaceReviewRequest = action === "exclude" ? { action: "exclude_face", faceId: chosen.faceId, expectedFaceRevision: chosen.revision }
      : { action: "assign_face", faceId: chosen.faceId, expectedFaceRevision: chosen.revision, personId: destination!.id, expectedPersonRevision: destination!.revision };
    setResult(null);
    setConfirmation({ request, key: createRequestKey(), label: photoLabel(index + 1, mark), ...(action === "assign" ? { targetName: destination!.name } : {}) });
  }
  async function submit() {
    if (!confirmation || busy.current) return;
    busy.current = true; setSaving(true);
    try {
      onReviewStart?.();
      const result = await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: confirmation.key, review: confirmation.request });
      onReviewCommitted?.({ result, subject: confirmation.label, target: confirmation.targetName });
      setResult({ key: "saved" }); setSelected(""); setConfirmation(null);
    } catch (error) {
      const conflict = isFaceReviewConflict(error);
      setResult({ key: conflict ? "conflict" : "unconfirmed" });
      if (conflict) setConfirmation(null);
      // An uncertain response keeps the exact intent/key for an explicit retry.
    } finally {
      await Promise.all([query.refetch(), people.refetch(), ...faceQueries.map(query => query.refetch()), ...(target ? [targetQuery.refetch()] : [])]);
      busy.current = false; setSaving(false);
    }
  }
  async function createEmpty() {
    const draft = name.trim(); if (!draft || busy.current) return;
    const intent = createIntent.forName(draft);
    busy.current = true; setSaving(true);
    try {
      const person = await create.mutateAsync({ csrfToken: session.csrfToken, ...intent });
      setTarget(person); setSearch(""); setName(""); setCreateOpen(false); createIntent.complete();
      setResult({ key: "created", name: person.name });
    } catch { setResult({ key: "unconfirmed" }); }
    finally { await people.refetch(); busy.current = false; setSaving(false); }
  }

  return <section className={styles.panel} aria-labelledby="group-review-title">
    <Button variant="quiet" disabled={saving} onClick={onBack}>{t("intelligence.back")}</Button>
    <header><h2 id="group-review-title" ref={heading} tabIndex={-1}>{t("groupReview.title")}</h2><p>{t("groupReview.description")}</p></header>
    {result && <InlineStatus tone={result.key === "saved" || result.key === "created" ? "success" : "warning"}>{t(`groupReview.${result.key}`).replace("{name}", result.name ?? "")}</InlineStatus>}
    {query.isPending ? <LoadingState label={t("common.loading")} /> : query.isError ? <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} /> : <>
      <div className={styles.summary}><strong>{t(`intelligence.${query.data.cluster.kind}`)}</strong><span>{t("groupReview.faceCount").replace("{count}", String(query.data.cluster.memberCount))}</span><span>{t("groupReview.pageCount").replace("{faces}", String(items.length)).replace("{photos}", String(photos.size))}</span></div>
      <div className={styles.toolbar}><label>{t("intelligence.searchPeople")}<Input type="search" maxLength={100} disabled={saving} value={search} onChange={event => { setSearch(event.target.value); setTarget(null); }} /></label>
        <label>{t("peopleReview.keepPerson")}<Select disabled={saving || people.isFetching || people.isError} value={target?.id ?? ""} onChange={event => { const next = choices.find(person => person.id === event.target.value); setTarget(next ? { ...next } : null); }}><option value="">{t("peopleReview.choosePerson")}</option>{choices.map(person => <option key={person.id} value={person.id}>{person.name} · {t("peopleDirectory.photoCount").replace("{count}", String(person.assetCount))}</option>)}</Select></label>
        <Button disabled={saving} variant="quiet" onClick={() => { setResult(null); setCreateOpen(true); }}>{t("intelligence.createPerson")}</Button>
        <Button disabled={saving || !fresh || !chosen || !target || !targetQuery.data || targetQuery.isFetching || targetQuery.isError} variant="primary" onClick={() => openConfirmation("assign")}>{t("groupReview.assign")}</Button>
        <Button disabled={saving || !fresh || !chosen} variant="secondary" onClick={() => openConfirmation("exclude")}>{t("groupReview.exclude")}</Button>
      </div>
      {people.isError && <ReviewErrorState error={people.error} onRetry={() => void people.refetch()} />}
      {target && targetQuery.isError && <ReviewErrorState error={targetQuery.error} onRetry={() => void targetQuery.refetch()} />}
      {!items.length ? <EmptyState title={t("groupReview.empty")} description={t("groupReview.returnHint")} /> : <div className={styles.photos}>{[...photos.entries()].map(([assetId, members], photoIndex) => <article className={styles.photo} key={assetId} aria-label={t("groupReview.photo").replace("{number}", String(photoIndex + 1))}>
        <div className={styles.stage}><img loading="lazy" src={assetContentUrl(assetId)} alt={t("groupReview.photo").replace("{number}", String(photoIndex + 1))} />{members.map((member, mark) => <FaceSelectionBox key={member.faceId} face={member} marker={String(mark + 1)} label={photoLabel(photoIndex + 1, mark + 1)} selected={selected === member.faceId} disabled={saving || !fresh || !eligible(member)} onSelect={() => setSelected(member.faceId)} />)}</div>
        <fieldset><legend>{t("groupReview.photo").replace("{number}", String(photoIndex + 1))}</legend>{members.map((member, mark) => <div key={member.faceId}><RadioOption name="group-face" checked={selected === member.faceId} disabled={saving || !fresh || !eligible(member)} onChange={() => setSelected(member.faceId)}>{photoLabel(photoIndex + 1, mark + 1)}</RadioOption><span>{t(faceState(member)?.state === "assigned" ? "groupReview.assigned" : faceState(member)?.state === "excluded" ? "groupReview.excluded" : eligible(member) ? "groupReview.pending" : "groupReview.stateUnavailable")}</span></div>)}
          {faceQueries[photoIndex]?.isPending && <LoadingState label={t("common.loading")} />}
          {faceQueries[photoIndex]?.isError && <ReviewErrorState error={faceQueries[photoIndex]!.error} onRetry={() => void faceQueries[photoIndex]!.refetch()} />}
          <Link to={paths.media(libraryId, assetId)}>{t("groupReview.openPhoto")}</Link></fieldset>
      </article>)}</div>}
      {!saving && <ReviewPagination nextCursor={query.data.nextCursor} />}
    </>}
    <Dialog fallbackFocusRef={heading} open={!!confirmation} title={t(confirmation?.request.action === "exclude_face" ? "groupReview.excludeTitle" : "groupReview.assignTitle")} description={t("groupReview.confirmHint")} onOpenChange={open => { if (!open && !busy.current) setConfirmation(null); }} actions={<><Button disabled={saving} onClick={() => setConfirmation(null)}>{t("common.cancel")}</Button><Button disabled={saving || !confirmation} variant="primary" onClick={() => void submit()}>{t("groupReview.confirm")}</Button></>}>
      <p>{confirmation?.label}</p>{confirmation?.targetName && <p>{t("groupReview.destination").replace("{name}", confirmation.targetName)}</p>}
      {result?.key === "unconfirmed" && <InlineStatus tone="warning">{t("groupReview.unconfirmed")}</InlineStatus>}
    </Dialog>
    <Dialog fallbackFocusRef={heading} open={createOpen} title={t("intelligence.createPerson")} description={t("groupReview.createHint")} onOpenChange={open => { if (!open && !busy.current) setCreateOpen(false); }} actions={<><DialogCloseButton onClick={() => { if (!busy.current) setCreateOpen(false); }} /><Button variant="primary" disabled={saving || !name.trim()} onClick={() => void createEmpty()}>{t("intelligence.createPerson")}</Button></>}>
      <label className={styles.createName}>{t("intelligence.name")}<Input maxLength={100} value={name} disabled={saving} onChange={event => setName(event.target.value)} /></label>
      {result?.key === "unconfirmed" && <InlineStatus tone="warning">{t("groupReview.unconfirmed")}</InlineStatus>}
    </Dialog>
  </section>;
}
