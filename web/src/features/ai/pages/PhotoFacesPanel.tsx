import { useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Button, Dialog, DialogCloseButton, EmptyState, InlineStatus, Input, LoadingState, RadioOption, Select } from "../../../components/ui";
import type { AssetFace, FaceReviewRequest } from "../../../lib/api/ai";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { assetContentUrl } from "../../../lib/api/catalog";
import { useLocale } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useAssetQuery } from "../../media";
import { useApplyFaceReviewMutation, useAssetFacesQuery, usePeopleQuery, usePersonQuery } from "../queries";
import { isFaceReviewConflict } from "../reviewErrors";
import { FaceSelectionBox } from "./FaceSelectionBox";
import { ReviewErrorState } from "./ReviewFeedback";
import styles from "./PhotoFacesPanel.module.css";
import type { FaceReviewCallbacks } from "./FaceReviewUndo";

type Confirmation = { request: FaceReviewRequest; key: string; ordinal: number; name: string; action: "assign" | "exclude" | "remove" | "separate"; otherOrdinal?: number; source?: string; target?: string };
export function PhotoFacesPanel({ assetId, session, onReviewStart, onReviewCommitted }: { assetId: string; onAsset: (id: string) => void; session: AuthenticatedSession } & FaceReviewCallbacks) {
  const { t } = useLocale(); const heading = useRef<HTMLHeadingElement>(null);
  const asset = useAssetQuery(assetId); const faces = useAssetFacesQuery(assetId);
  const [selected, setSelected] = useState(""); const [otherId, setOtherId] = useState(""); const [search, setSearch] = useState(""); const [targetId, setTargetId] = useState("");
  const people = usePeopleQuery(search.trim()); const target = usePersonQuery(targetId);
  const chosen = faces.data?.items.find(face => face.faceId === selected && face.assetId === assetId);
  const source = usePersonQuery(chosen?.personId ?? "");
  const peers = faces.data?.items.filter(face => face.assetId === assetId && face.faceId !== chosen?.faceId && face.state === "assigned" && face.personId && face.personId !== chosen?.personId) ?? [];
  const peer = peers.find(face => face.faceId === otherId);
  const peerPerson = usePersonQuery(peer?.personId ?? ""); const review = useApplyFaceReviewMutation();
  const imageToken = `${assetId}:${asset.data?.modifiedAt ?? ""}:${asset.data?.sizeBytes ?? ""}`;
  const [loadedToken,setLoadedToken]=useState(""); const [failedToken,setFailedToken]=useState("");
  const imageReady=loadedToken===imageToken;const imageFailed=failedToken===imageToken;
  const [confirmation, setConfirmation] = useState<Confirmation | null>(null); const [result, setResult] = useState<"saved" | "separated" | "conflict" | "unconfirmed" | null>(null);
  const [saving, setSaving] = useState(false); const busy = useRef(false);
  const fresh = asset.data?.sourceAvailability === "available" && asset.data.kind === "image" && !asset.isError && !asset.isFetching && !faces.isError && !faces.isFetching && !!faces.data && imageReady;
  const sourceReady = !chosen?.personId || (source.data?.id === chosen.personId && !source.isFetching && !source.isError);
  const targetReady = targetId && target.data?.id === targetId && !target.isFetching && !target.isError && !people.isFetching && !people.isError;
  const locked = saving || confirmation !== null;
  const canEdit = fresh && chosen && sourceReady && !saving;
  const pairKnown = !!chosen && !!peer && Array.isArray(chosen.cannotLinkFaceIds) && Array.isArray(peer.cannotLinkFaceIds) && chosen.cannotLinkFaceIds.includes(peer.faceId) === peer.cannotLinkFaceIds.includes(chosen.faceId);
  const alreadySeparate = pairKnown && chosen!.cannotLinkFaceIds!.includes(peer!.faceId);
  const canSeparate = canEdit && !!chosen?.personId && !!peer && peerPerson.data?.id === peer.personId && !peerPerson.isFetching && !peerPerson.isError && pairKnown && !alreadySeparate;
  function selectFace(id: string) { setSelected(id); setOtherId(""); setResult(null); }
  function openSeparate() {
    if (!canSeparate || locked || busy.current || !chosen || !peer || !source.data || !peerPerson.data) return;
    const request: FaceReviewRequest = { action: "cannot_link", leftFaceId: chosen.faceId, rightFaceId: peer.faceId, expectedLeftRevision: chosen.revision, expectedRightRevision: peer.revision, sourcePersonId: source.data.id, targetPersonId: peerPerson.data.id, expectedSourceRevision: source.data.revision, expectedTargetRevision: peerPerson.data.revision };
    setResult(null); setConfirmation({ request, key: createRequestKey(), ordinal: chosen.ordinal, otherOrdinal: peer.ordinal, name: asset.data!.name, action: "separate", source: source.data.name, target: peerPerson.data.name });
  }
  function label(face: AssetFace) { return t("photoFaces.face").replace("{number}",String(face.ordinal)).replace("{state}",t(`photoFaces.state.${face.state}`)); }
  function open(action: Exclude<Confirmation["action"], "separate">) {
    if (!canEdit || locked || !chosen || busy.current || (action === "assign" && (!targetReady || targetId === chosen.personId)) || (action === "remove" && !chosen.personId) || (action === "exclude" && chosen.state === "excluded")) return;
    const guard = chosen.personId && source.data ? { sourcePersonId: source.data.id, expectedSourceRevision: source.data.revision } : {};
    const request: FaceReviewRequest = action === "assign" ? { action: "assign_face", faceId: chosen.faceId, expectedFaceRevision: chosen.revision, personId: target.data!.id, expectedPersonRevision: target.data!.revision, ...guard }
      : action === "remove" ? { action: "split_face", faceId: chosen.faceId, expectedFaceRevision: chosen.revision, sourcePersonId: source.data!.id, expectedSourceRevision: source.data!.revision }
      : { action: "exclude_face", faceId: chosen.faceId, expectedFaceRevision: chosen.revision, ...guard };
    setResult(null); setConfirmation({ request, key: createRequestKey(), ordinal: chosen.ordinal, name: asset.data!.name, action, ...(chosen.personId && source.data ? {source:source.data.name}:{}), ...(action === "assign" ? {target:target.data!.name}:{}) });
  }
  async function submit() {
    if (!confirmation || busy.current) return; busy.current = true; setSaving(true); setResult(null);
    try { onReviewStart?.(); const result = await review.mutateAsync({ csrfToken: session.csrfToken, idempotencyKey: confirmation.key, review: confirmation.request }); onReviewCommitted?.({ result, subject: `${confirmation.name} · ${t("photoFaces.number").replace("{number}", String(confirmation.ordinal))}${confirmation.otherOrdinal ? ` / ${t("photoFaces.number").replace("{number}", String(confirmation.otherOrdinal))}` : ""}`, source: confirmation.source, target: confirmation.target }); setConfirmation(null); setSelected(""); setResult(confirmation.action === "separate" ? "separated" : "saved"); }
    catch(error) { const conflict = isFaceReviewConflict(error); setResult(conflict ? "conflict" : "unconfirmed"); if(conflict){setConfirmation(null);setSelected("");} }
    // The mutation owner invalidates person queries, including inactive sources.
    // A captured observer.refetch may now point at the empty/changed selection.
    finally { await Promise.all([faces.refetch(),people.refetch(),asset.refetch()]);busy.current=false;setSaving(false); }
  }
  if (!assetId) return <section className={styles.panel}><h2>{t("photoFaces.title")}</h2><p>{t("photoFaces.choosePhoto")}</p></section>;
  return <section className={styles.panel} aria-labelledby="photo-faces-title">
    <header><h2 id="photo-faces-title" ref={heading} tabIndex={-1}>{asset.data?.name ?? t("photoFaces.title")}</h2><p>{t("photoFaces.description")}</p>{asset.data && <Link to={paths.media(asset.data.libraryId,assetId)}>{t("photoFaces.back")}</Link>}</header>
    {result && <InlineStatus tone={(result === "saved" || result === "separated") ? "success" : "warning"}>{t(result === "unconfirmed" ? "photoFaces.unconfirmedSummary" : `photoFaces.${result}`)}</InlineStatus>}
    {asset.isPending ? <LoadingState label={t("common.loading")} /> : asset.isError ? <ReviewErrorState error={asset.error} onRetry={()=>void asset.refetch()} /> : asset.data.sourceAvailability !== "available" ? <InlineStatus tone="warning">{t(asset.data.sourceAvailability === "offline" ? "intelligence.offline" : "photoFaces.unavailable")}</InlineStatus> : asset.data.kind !== "image" ? <InlineStatus tone="warning">{t("photoFaces.unsupported")}</InlineStatus> : <>
      {faces.isPending ? <LoadingState label={t("common.loading")} /> : faces.isError ? <ReviewErrorState error={faces.error} onRetry={()=>void faces.refetch()} /> : !faces.data.items.length ? <EmptyState title={t("intelligence.none")} description={t("intelligence.emptyFaces")} /> : <div className={styles.workspace}>
        <div className={styles.stage}><img key={imageToken} src={assetContentUrl(assetId)} alt={asset.data.name} onLoad={()=>{setLoadedToken(imageToken);setFailedToken("");}} onError={()=>{setLoadedToken("");setFailedToken(imageToken);}} />{faces.data.items.map(face=><FaceSelectionBox key={face.faceId} face={face} marker={String(face.ordinal)} label={label(face)} selected={face.faceId===selected} disabled={locked || !imageReady || !fresh} onSelect={()=>selectFace(face.faceId)} />)}</div>
        <div className={styles.tools}>{imageFailed && <InlineStatus tone="warning">{t("photoFaces.imageFailed")}</InlineStatus>}
          <fieldset><legend>{t("photoFaces.select")}</legend>{faces.data.items.map(face=><RadioOption key={face.faceId} name="single-photo-face" value={face.faceId} checked={face.faceId===selected} disabled={locked || !imageReady || !fresh} onChange={()=>selectFace(face.faceId)}>{label(face)}</RadioOption>)}</fieldset>
          {chosen?.personId && <div>{source.isPending ? <LoadingState label={t("common.loading")} /> : source.isError ? <ReviewErrorState error={source.error} onRetry={()=>void source.refetch()} /> : source.data && <p>{t("photoFaces.current")} <Link to={`/intelligence?view=people&person=${encodeURIComponent(source.data.id)}`}>{source.data.name}</Link></p>}</div>}
          <label>{t("intelligence.searchPeople")}<Input type="search" maxLength={100} disabled={locked} value={search} onChange={event=>{setSearch(event.target.value);setTargetId("");}} /></label>
          <label>{t("peopleReview.keepPerson")}<Select value={targetId} disabled={locked || people.isFetching || people.isError} onChange={event=>setTargetId(event.target.value)}><option value="">{t("peopleReview.choosePerson")}</option>{people.data?.items.filter(person=>person.id!==chosen?.personId).map(person=><option key={person.id} value={person.id}>{person.name} · {t("peopleDirectory.photoCount").replace("{count}", String(person.assetCount))}</option>)}</Select></label>
          {people.isError && <ReviewErrorState error={people.error} onRetry={()=>void people.refetch()} />}{targetId && target.isError && <ReviewErrorState error={target.error} onRetry={()=>void target.refetch()} />}
          <div className={styles.actions}><Button disabled={!canEdit || !targetReady || targetId===chosen?.personId} onClick={()=>open("assign")}>{t("photoFaces.assign")}</Button>{chosen?.personId && <Button variant="secondary" disabled={!canEdit} onClick={()=>open("remove")}>{t("intelligence.removeAssignment")}</Button>}<Button variant="quiet" disabled={!canEdit || chosen?.state === "excluded"} onClick={()=>open("exclude")}>{t("intelligence.exclude")}</Button></div>
          {chosen?.personId && <details className={styles.separation}><summary>{t("faceSeparate.title")}</summary><p>{t("faceSeparate.hint")}</p>
            {!peers.length ? <p>{t("faceSeparate.choosePeopleFirst")}</p> : <>
              <label>{t("faceSeparate.otherFace")}<Select value={otherId} disabled={locked || !fresh} onChange={event => setOtherId(event.target.value)}><option value="">{t("faceSeparate.chooseFace")}</option>{peers.map(face => <option value={face.faceId} key={face.faceId}>{t("photoFaces.number").replace("{number}", String(face.ordinal))}</option>)}</Select></label>
              {peer && (peerPerson.isPending ? <LoadingState label={t("common.loading")} /> : peerPerson.isError ? <ReviewErrorState error={peerPerson.error} onRetry={()=>void peerPerson.refetch()} /> : peerPerson.data && <p>{t("faceSeparate.otherPerson")} {peerPerson.data.name}</p>)}
              {peer && !pairKnown && <InlineStatus tone="warning">{t("faceSeparate.unknown")}</InlineStatus>}
              {alreadySeparate && <InlineStatus>{t("faceSeparate.exists")}</InlineStatus>}
              <Button variant="secondary" disabled={!canSeparate} onClick={openSeparate}>{t("faceSeparate.mark")}</Button>
            </>}
          </details>}

        </div>
      </div>}
    </>}
    <Dialog open={!!confirmation} onOpenChange={open=>{if(!open&&!busy.current)setConfirmation(null);}} fallbackFocusRef={heading} title={t(confirmation?.action === "separate" ? "faceSeparate.confirm" : confirmation?.action === "assign" ? "photoFaces.confirmAssign" : confirmation?.action === "remove" ? "photoFaces.confirmRemove" : "photoFaces.confirmExclude")} description={t(confirmation?.action === "separate" ? "faceSeparate.confirmHint" : "photoFaces.safety")} actions={<><DialogCloseButton onClick={()=>{if(!busy.current)setConfirmation(null);}} /><Button disabled={saving} onClick={()=>void submit()}>{t("groupReview.confirm")}</Button></>}>
      {confirmation && <div className={styles.confirmation}><strong>{confirmation.name}</strong><p>{t("photoFaces.number").replace("{number}",String(confirmation.ordinal))}</p>{confirmation.otherOrdinal && <p>{t("photoFaces.number").replace("{number}",String(confirmation.otherOrdinal))}</p>}{confirmation.source && <p>{t("photoFaces.source")} {confirmation.source}</p>}{confirmation.target && <p>{t("photoFaces.destination")} {confirmation.target}</p>}{result === "unconfirmed" && <InlineStatus tone="warning">{t("photoFaces.unconfirmed")}</InlineStatus>}</div>}
    </Dialog>
  </section>;
}
