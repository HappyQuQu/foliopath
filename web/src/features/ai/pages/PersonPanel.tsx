import { useLayoutEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Button, Dialog, DialogCloseButton, EmptyState, InlineStatus, Input, LoadingState, Select, useToast } from "../../../components/ui";
import type { AssetFace, FaceReviewRequest, Person, PersonAssetPage } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { assetContentUrl } from "../../../lib/api/catalog";
import { isFaceReviewConflict } from "../reviewErrors";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useApplyFaceReviewMutation, useAssetFacesQuery, usePeopleQuery, usePersonAssetsQuery, usePersonQuery, usePersonMergeCheckQuery, useRenamePersonMutation } from "../queries";
import { ReviewErrorState, ReviewPagination } from "./ReviewFeedback";
import { FaceSelectionBox } from "./FaceSelectionBox";
import styles from "./PersonPanel.module.css";
import type { FaceReviewCallbacks } from "./FaceReviewUndo";

type FaceConfirmation = { request: FaceReviewRequest; assetName: string; ordinal: number; sourceName: string; targetName?: string };
type ReviewIntent = { key: string; request: FaceReviewRequest } & (
  { kind: "face"; face: FaceConfirmation } | { kind: "merge"; source: Person; target: Person }
);

export function PersonPanel({ onBack, personId, session, onReviewStart, onReviewCommitted }: { onBack: () => void; personId: string; session: AuthenticatedSession } & FaceReviewCallbacks) {
  const { t } = useLocale(); const toast = useToast(); const [params] = useSearchParams();
  const person = usePersonQuery(personId); const assets = usePersonAssetsQuery(personId, params.get("cursor") ?? "");
  const [name, setName] = useState(""); const [peopleSearch, setPeopleSearch] = useState("");
  const people = usePeopleQuery(peopleSearch.trim());
  const [target, setTarget] = useState<Person | null>(null); const [mergeSource, setMergeSource] = useState<Person | null>(null);
  const [acknowledged, setAcknowledged] = useState(false);
  const [intent, setIntent] = useState<ReviewIntent | null>(null); const [intentOpen, setIntentOpen] = useState(false);
  const [uncertain, setUncertain] = useState(false); const [saving, setSaving] = useState(false); const savingRef = useRef(false);
  const rename = useRenamePersonMutation(); const review = useApplyFaceReviewMutation();
  const backRef = useRef<HTMLButtonElement>(null); const headingRef = useRef<HTMLHeadingElement>(null);
  const items = assets.data?.items ?? []; const previousCount = useRef(items.length);
  useLayoutEffect(() => { if (items.length < previousCount.current && document.activeElement === document.body) headingRef.current?.focus(); previousCount.current = items.length; }, [items.length]);
  const source = person.data;
  const fresh = !!source && !person.isFetching && !person.isError && !assets.isFetching;
  const locked = saving || uncertain;
  const faceIntent = intent?.kind === "face" ? intent : null;
  const mergeIntent = intent?.kind === "merge" ? intent : null;
  const mergeSnapshot = mergeIntent?.source ?? mergeSource;
  const destination = mergeIntent?.target ?? target;
  const mergeCheck = usePersonMergeCheckQuery(mergeSource?.id ?? "", target?.id ?? "", !!mergeSource && !intent);
  const checkMatches = !!mergeCheck.data && mergeCheck.data.sourcePersonId === mergeSource?.id && mergeCheck.data.targetPersonId === target?.id && mergeCheck.data.sourceRevision === mergeSource?.revision && mergeCheck.data.targetRevision === target?.revision;
  const canSubmitNewMerge = !!target && !!mergeSource && acknowledged && checkMatches && mergeCheck.data?.canMerge && !mergeCheck.isFetching && !mergeCheck.isError && !mergeCheck.isPending;

  async function refresh() { await Promise.all([person.refetch(), assets.refetch(), people.refetch()]); }
  function clearIntent() { setIntent(null); setIntentOpen(false); setUncertain(false); setMergeSource(null); }
  function closeIntent() {
    if (savingRef.current) return;
    if (uncertain) setIntentOpen(false); else clearIntent();
  }
  function confirmFace(face: FaceConfirmation) {
    if (savingRef.current || intent) return;
    setIntent({ kind: "face", face: { ...face }, request: { ...face.request }, key: createRequestKey() }); setIntentOpen(true);
  }
  async function saveName() {
    if (!source || !name.trim() || savingRef.current || intent) return;
    savingRef.current = true; setSaving(true);
    try { await rename.mutateAsync({ csrfToken: session.csrfToken, etag: source.etag, personId, name: name.trim() }); setName(""); toast.show({ message: t("intelligence.saved"), tone: "success" }); }
    catch { toast.show({ message: t("peopleReview.actionUnconfirmed"), tone: "danger" }); }
    finally { await refresh(); savingRef.current = false; setSaving(false); }
  }
  async function submitReview(operation: ReviewIntent) {
    if (savingRef.current) return;
    savingRef.current = true; setSaving(true);
    try {
      onReviewStart?.();
      const result = await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: operation.key, review: operation.request });
      onReviewCommitted?.({ result, subject: operation.kind === "merge" ? t("intelligence.mergePeople") : `${operation.face.assetName} · ${t("photoFaces.number").replace("{number}", String(operation.face.ordinal))}`,
        source: operation.kind === "merge" ? operation.source.name : operation.face.sourceName,
        target: operation.kind === "merge" ? operation.target.name : operation.face.targetName });
      clearIntent(); toast.show({ message: t("intelligence.saved"), tone: "success" });
      if (operation.kind === "merge") onBack();
    } catch (error) {
      const conflict = isFaceReviewConflict(error);
      if (conflict) clearIntent(); else { setUncertain(true); setIntentOpen(true); }
      toast.show({ message: t(conflict ? "intelligence.revisionConflict" : "peopleReview.actionUnconfirmed"), tone: "danger" });
    } finally { await refresh(); savingRef.current = false; setSaving(false); }
  }
  function submitMerge() {
    if (savingRef.current) return;
    if (mergeIntent) { void submitReview(mergeIntent); return; }
    if (!mergeSource || !target || !canSubmitNewMerge || intent) return;
    const operation: ReviewIntent = { kind: "merge", source: { ...mergeSource }, target: { ...target }, key: createRequestKey(), request: { action: "merge_people", sourcePersonId: mergeSource.id, targetPersonId: target.id, expectedSourceRevision: mergeSource.revision, expectedTargetRevision: target.revision, conflictsAcknowledged: true } };
    setIntent(operation); setIntentOpen(true); void submitReview(operation);
  }
  return <section className={styles.panel} aria-labelledby="person-detail-title">
    <Button ref={backRef} variant="quiet" disabled={saving} onClick={onBack}>{t("intelligence.back")}</Button>
    <header className={styles.header}><div><h2 id="person-detail-title" ref={headingRef} tabIndex={-1}>{source?.name ?? (mergeIntent ? mergeIntent.source.name : faceIntent?.face.sourceName) ?? t("peopleRecovery.title")}</h2>{source && !person.isError && <p>{t("intelligence.personCounts").replace("{faces}", String(source.confirmedFaceCount)).replace("{assets}", String(source.assetCount))}</p>}</div>
      <Button disabled={locked || !fresh} variant="secondary" onClick={() => { if (!intent && source) { setMergeSource({ ...source }); setTarget(null); setPeopleSearch(""); setAcknowledged(false); } }}>{t("intelligence.mergePeople")}</Button>
    </header>
    {uncertain && intent && <section className={styles.namePanel} aria-label={t("peopleRecovery.title")}><InlineStatus tone="warning">{t("peopleRecovery.notice")}</InlineStatus><div className={styles.photoActions}><Button disabled={saving} onClick={() => setIntentOpen(true)}>{t("peopleRecovery.reopen")}</Button><Button variant="quiet" disabled={saving} onClick={clearIntent}>{t("peopleRecovery.dismiss")}</Button></div></section>}
    {person.isPending ? <LoadingState label={t("common.loading")} /> : person.isError ? <ReviewErrorState error={person.error} onRetry={() => void refresh()} /> : source && <>
      <section className={styles.namePanel} aria-label={t("peopleReview.nameSection")}>
        <form onSubmit={event => { event.preventDefault(); void saveName(); }}><label>{t("intelligence.name")}<Input placeholder={source.name} maxLength={100} value={name} disabled={locked} onChange={event => setName(event.target.value)} /></label><Button type="submit" variant="primary" disabled={locked || !fresh || !name.trim() || name.trim() === source.name}>{t("intelligence.rename")}</Button></form><p>{t("peopleReview.nameHint")}</p>
      </section>
      <header className={styles.photosHeader}><div><h3>{t("intelligence.personAssets")}</h3><p>{t("peopleReview.faceHint")}</p></div><span>{t("peopleReview.loaded").replace("{count}", String(items.length))}</span></header>
      {assets.isPending ? <LoadingState label={t("common.loading")} /> : assets.isError ? <ReviewErrorState error={assets.error} onRetry={() => void assets.refetch()} /> : !items.length ? <EmptyState title={t("peopleReview.noPhotos")} description={t("peopleReview.emptyHint")} /> : <div className={styles.photos}>{items.map(item => <PersonPhoto key={item.asset.id} item={item} person={source} disabled={locked || !fresh} onConfirm={confirmFace} />)}</div>}
      {!locked && <ReviewPagination nextCursor={assets.data?.nextCursor} />}
    </>}
    <Dialog fallbackFocusRef={headingRef} open={!!faceIntent && intentOpen} onOpenChange={open => { if (!open) closeIntent(); }} title={t(faceIntent?.request.action === "assign_face" ? "peopleReview.moveTitle" : "peopleReview.removeTitle")} description={t("peopleReview.faceSafety")} actions={<><Button disabled={saving} onClick={closeIntent}>{t("common.cancel")}</Button><Button variant="primary" disabled={saving || !faceIntent} onClick={() => { if (faceIntent) void submitReview(faceIntent); }}>{t("peopleReview.confirmFace")}</Button></>}>
      {faceIntent && <div className={styles.confirmation}><strong>{faceIntent.face.assetName}</strong><p>{t("peopleReview.selectedFace").replace("{number}", String(faceIntent.face.ordinal))}</p><p>{faceIntent.face.sourceName}{faceIntent.face.targetName ? ` → ${faceIntent.face.targetName}` : ""}</p><p>{t(faceIntent.face.targetName ? "peopleReview.moveHint" : "peopleReview.removeHint")}</p>{uncertain && <InlineStatus tone="warning">{t("peopleRecovery.uncertain")}</InlineStatus>}</div>}
    </Dialog>
    <Dialog fallbackFocusRef={headingRef} open={!!mergeSource && (!mergeIntent || intentOpen)} onOpenChange={open => { if (!open) closeIntent(); }} title={t("intelligence.mergePeople")} description={t("peopleReview.mergeSafety")} actions={<><DialogCloseButton onClick={closeIntent} /><Button variant="primary" disabled={saving || (!mergeIntent && !canSubmitNewMerge)} onClick={submitMerge}>{t("peopleReview.confirmMerge")}</Button></>}>
      <div className={styles.confirmation}>{!mergeIntent && <><label>{t("intelligence.searchPeople")}<Input type="search" value={peopleSearch} maxLength={100} disabled={saving} onChange={event => { setPeopleSearch(event.target.value); setTarget(null); setAcknowledged(false); }} /></label>
        <label>{t("peopleReview.keepPerson")}<Select value={target?.id ?? ""} disabled={saving || people.isFetching || people.isError} onChange={event => { const next = people.data?.items.find(item => item.id === event.target.value); setTarget(next ? { ...next } : null); setAcknowledged(false); }}><option value="">{t("peopleReview.choosePerson")}</option>{people.data?.items.filter(item => item.id !== personId).map(item => <option key={item.id} value={item.id}>{item.name} · {t("peopleDirectory.photoCount").replace("{count}", String(item.assetCount))}</option>)}</Select></label>
        {people.isError && <ReviewErrorState error={people.error} onRetry={() => void people.refetch()} />}</>}
        {destination && mergeSnapshot && <><p>{mergeSnapshot.name} → <strong>{destination.name}</strong></p><p>{t("peopleReview.mergeCounts").replace("{source}", String(mergeSnapshot.assetCount)).replace("{target}", String(destination.assetCount))}</p>{!mergeIntent && <label className={styles.acknowledgement}><input type="checkbox" checked={acknowledged} disabled={saving || !checkMatches || !mergeCheck.data?.canMerge || mergeCheck.isFetching || mergeCheck.isError} onChange={event => setAcknowledged(event.target.checked)} />{t("peopleReview.acknowledgeMerge")}</label>}</>}
        {target && !mergeIntent && <section aria-label={t("mergeCheck.title")}>
          {mergeCheck.isPending || mergeCheck.isFetching ? <LoadingState label={t("mergeCheck.loading")} /> : mergeCheck.isError ? <><InlineStatus tone="warning">{t("mergeCheck.unavailable")}</InlineStatus><ReviewErrorState error={mergeCheck.error} onRetry={() => void mergeCheck.refetch()} /></> : mergeCheck.data && <>
            {!checkMatches ? <InlineStatus tone="warning">{t("mergeCheck.changed")}</InlineStatus> : mergeCheck.data.canMerge ? <InlineStatus>{t("mergeCheck.allowed")}</InlineStatus> : <>
              <InlineStatus tone="warning">{t("mergeCheck.blocked")}</InlineStatus>
              {mergeCheck.data.cannotLinkCount > 0 && <p>{t("mergeCheck.links").replace("{count}", String(mergeCheck.data.cannotLinkCount))}</p>}
              {mergeCheck.data.duplicateAssignmentCount > 0 && <p>{t("mergeCheck.duplicates").replace("{count}", String(mergeCheck.data.duplicateAssignmentCount))}</p>}
              {mergeCheck.data.sourceLimitExceeded && <p>{t("mergeCheck.limit")}</p>}
              {mergeCheck.data.cannotLinkCount > 0 && <p>{mergeCheck.data.examples.length ? t("mergeCheck.examples").replace("{count}", String(mergeCheck.data.examples.length)) : t("mergeCheck.noExamples")}</p>}
              {mergeCheck.data.examples.length > 0 && <ul>{mergeCheck.data.examples.map((example,index) => <li key={`${example.sourceAssetId}:${example.targetAssetId}`}><p>{t("mergeCheck.example").replace("{number}", String(index + 1))}</p><div className={styles.photoActions}><Link target="_blank" rel="noopener" to={paths.media(example.sourceLibraryId,example.sourceAssetId)}>{t("mergeCheck.sourcePhoto")}</Link><Link target="_blank" rel="noopener" to={paths.media(example.targetLibraryId,example.targetAssetId)}>{t("mergeCheck.targetPhoto")}</Link></div></li>)}</ul>}
            </>}
            <Button variant="quiet" disabled={saving} onClick={() => { setAcknowledged(false); void mergeCheck.refetch(); }}>{t("mergeCheck.refresh")}</Button>
          </>}
        </section>}
        {uncertain && mergeIntent && <InlineStatus tone="warning">{t("peopleRecovery.uncertain")}</InlineStatus>}
      </div>
    </Dialog>
  </section>;
}

function PersonPhoto({ item, person, disabled, onConfirm }: { item: PersonAssetPage["items"][number]; person: Person; disabled: boolean; onConfirm: (confirmation: FaceConfirmation) => void }) {
  const { t } = useLocale();
  const query = useAssetFacesQuery(item.asset.id);
  const [selected, setSelected] = useState("");
  const [search, setSearch] = useState("");
  const [target, setTarget] = useState<Person | null>(null);
  const [moving, setMoving] = useState(false);
  const people = usePeopleQuery(moving ? search.trim() : "");
  // Both contracts must agree; never pick another person's face or an old anchor.
  const owned = query.data?.items.filter(face => face.personId === person.id && item.faceIds.includes(face.faceId)) ?? [];
  const chosen = owned.length === 1 ? owned[0] : owned.find(face => face.faceId === selected);
  const locked = disabled || query.isPending || query.isFetching || query.isError || !chosen;
  function confirm(mode: "remove" | "move", face: AssetFace) {
    const request: FaceReviewRequest = mode === "remove"
      ? { action: "split_face", faceId: face.faceId, expectedFaceRevision: face.revision, sourcePersonId: person.id, expectedSourceRevision: person.revision }
      : { action: "assign_face", faceId: face.faceId, expectedFaceRevision: face.revision, sourcePersonId: person.id, expectedSourceRevision: person.revision, personId: target!.id, expectedPersonRevision: target!.revision };
    onConfirm({ request, assetName: item.asset.name, ordinal: face.ordinal, sourceName: person.name, ...(mode === "move" ? { targetName: target!.name } : {}) });
  }
  return <article className={styles.photo} aria-label={item.asset.name}>
    <div className={styles.stage}><img src={item.asset.thumbnail.url ?? assetContentUrl(item.asset.id)} alt={item.asset.name} loading="lazy" />{owned.map(face => <FaceSelectionBox key={face.faceId} face={face} disabled={disabled} label={t("peopleReview.selectFace").replace("{number}", String(face.ordinal))} selected={chosen?.faceId === face.faceId} onSelect={() => setSelected(face.faceId)} />)}</div>
    <div className={styles.photoBody}><Link className={styles.mediaLink} to={paths.media(item.asset.libraryId, item.asset.id)}>{item.asset.name}</Link>
      {query.isPending ? <LoadingState label={t("common.loading")} /> : query.isError ? <ReviewErrorState error={query.error} onRetry={() => void query.refetch()} /> : !owned.length ? <p>{t("peopleReview.noCurrentFaces")}</p> : <>
        <label>{t("peopleReview.faceToEdit")}<Select value={chosen?.faceId ?? ""} disabled={disabled || query.isFetching} onChange={event => setSelected(event.target.value)}><option value="">{t("peopleReview.chooseFace")}</option>{owned.map(face => <option value={face.faceId} key={face.faceId}>{t("peopleReview.selectedFace").replace("{number}", String(face.ordinal))}</option>)}</Select></label>
        <div className={styles.photoActions}><Button disabled={locked} variant="quiet" onClick={() => { if (chosen) confirm("remove", chosen); }}>{t("intelligence.removeAssignment")}</Button><Button disabled={locked} variant="secondary" aria-expanded={moving} onClick={() => setMoving(value => !value)}>{t("intelligence.move")}</Button></div>
        {moving && <div className={styles.moveFields}><label>{t("intelligence.searchPeople")}<Input type="search" maxLength={100} value={search} disabled={disabled} onChange={event => { setSearch(event.target.value); setTarget(null); }} /></label><label>{t("peopleReview.keepPerson")}<Select value={target?.id ?? ""} disabled={disabled || people.isFetching || people.isError} onChange={event => { const next = people.data?.items.find(person => person.id === event.target.value); setTarget(next ? { ...next } : null); }}><option value="">{t("peopleReview.choosePerson")}</option>{people.data?.items.filter(value => value.id !== person.id).map(value => <option key={value.id} value={value.id}>{value.name} · {t("peopleDirectory.photoCount").replace("{count}", String(value.assetCount))}</option>)}</Select></label>{people.isError && <ReviewErrorState error={people.error} onRetry={() => void people.refetch()} />}<Button disabled={locked || !target} onClick={() => { if (chosen && target) confirm("move", chosen); }}>{t("peopleReview.reviewMove")}</Button></div>}
      </>}
    </div>
  </article>;
}
