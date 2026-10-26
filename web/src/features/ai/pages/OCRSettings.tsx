import { ArrowRight, TextT } from "@phosphor-icons/react";
import { useId, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Button, Dialog, ErrorState, InlineStatus, LoadingState, Switch } from "../../../components/ui";
import { ApiError } from "../../../lib/api/errors";
import { useLocale, type MessageKey } from "../../../lib/i18n/LocaleProvider";
import { createRequestKey } from "../../../lib/requestKey";
import { paths } from "../../../routes/paths";
import { useLibraryOCR, useOCRAction, type OCRCommand } from "../ocrQueries";
import { AIProcessingRow } from "./AIProcessingRow";
import styles from "./AISettingsOverview.module.css";

type Props={libraryId:string;csrfToken:string;offline:boolean;mode?:"settings"|"tasks";onSettings:()=>void;onTasks:()=>void};
export function OCRSettings(props:Props){return <LibraryOCRSettings key={`${props.libraryId}-${props.mode??"settings"}`} {...props}/>;}

function LibraryOCRSettings({libraryId,csrfToken,offline,mode="settings",onSettings,onTasks}:Props){
 const {t,locale}=useLocale();
 const heading=useId();const headingRef=useRef<HTMLHeadingElement>(null);
 const query=useLibraryOCR(libraryId);const mutation=useOCRAction();
 const [confirm,setConfirm]=useState<"retry_failed"|"clear"|null>(null);
 const [failure,setFailure]=useState<MessageKey|null>(null);
 const value=query.isError?undefined:query.data;
 const job=value?.latestJob;
 const active=Boolean(job&&["queued","running","cancelling"].includes(job.state));
 const locked=!value||query.isPending||query.isError||mutation.isPending||value.clearing;
 const canRun=!locked&&!offline&&value.enabled&&value.state!=="model_mismatch"&&!active;
 const number=new Intl.NumberFormat(locale);
 async function run(command:OCRCommand){
  if(!value)return;
  setFailure(null);
  try{await mutation.mutateAsync({...command,libraryId,csrfToken,etag:value.etag});setConfirm(null);}
  catch(error){setFailure(error instanceof ApiError&&error.code==="precondition_failed"?"ocr.conflict":"ocr.actionFailed");setConfirm(null);}
 }
 const status=<>
  {query.isPending?<LoadingState density={mode==="tasks"?"compact":"default"} label={t("ai.tasks.loadingStatus")}/>:query.isError?<ErrorState density={mode==="tasks"?"compact":"default"} message={t("ai.tasks.statusFailed")} onRetry={()=>void query.refetch()}/>:!value?<InlineStatus tone="warning">{t("ocr.unavailable")}</InlineStatus>:<>
   <strong>{t(`ocr.state.${value.state}`)}</strong>
   <p>{t("ocr.coverage").replace("{ready}",number.format(value.coverage.ready)).replace("{eligible}",number.format(value.coverage.eligible)).replace("{failed}",number.format(value.coverage.failed))}</p>
   <div className={styles.coverage}><progress aria-label={t("ocr.progress")} max={Math.max(1,value.coverage.eligible)} value={value.coverage.ready}/></div>
   <p className={styles.hint}>{t("ocr.pending").replace("{pending}",number.format(value.coverage.pending)).replace("{stale}",number.format(value.coverage.stale)).replace("{empty}",number.format(value.coverage.empty))}</p>
   {job&&mode==="tasks"&&<>
    {(job.epoch!==value.epoch||job.fingerprint!==value.expectedFingerprint)&&<p className={styles.hint}>{t("ocr.historicalTask")}</p>}
    <p className={styles.hint}>{t("ocr.recorded").replace("{recorded}",number.format(job.recorded)).replace("{failed}",number.format(job.failed)).replace("{attempts}",number.format(job.attempts))}</p>
    {job.errorCode?<InlineStatus tone="warning">{t(`ocr.error.${job.errorCode}`)}</InlineStatus>:!active&&<p>{t(job.state==="succeeded"?"ocr.job.succeeded":job.state==="failed"?"ocr.job.failed":"ocr.job.cancelled")}</p>}
   </>}
   {offline&&<InlineStatus tone="warning">{t("ocr.offline")}</InlineStatus>}
  </>}
  {failure&&<InlineStatus tone="danger">{t(failure)}</InlineStatus>}
 </>;
 const maintenance=value&&!query.isError?<>
  {value.coverage.failed>0&&<p className={styles.hint}>{t("ocr.sourceFailures")}</p>}
  <p className={styles.hint}>{t("ocr.clearHint")}</p>
  <div className={styles.actions}>
   <Button disabled={!canRun||value.coverage.failed===0} onClick={()=>setConfirm("retry_failed")}>{t("ocr.retry")}</Button>
   <Button disabled={locked} onClick={()=>setConfirm("clear")}>{t("ocr.clear")}</Button>
   <Link className={styles.textLink} to={paths.librarySearch(libraryId)+"?mode=ocr"}>{t("ocr.openSearch")}<ArrowRight aria-hidden="true" size={16}/></Link>
  </div>
 </>:undefined;
 const confirmation=<Dialog open={confirm!==null} onOpenChange={open=>{if(!open)setConfirm(null);}} fallbackFocusRef={headingRef} title={t(confirm==="clear"?"ocr.clearTitle":"ocr.retryTitle")} actions={<>
  <Button disabled={mutation.isPending} variant="quiet" onClick={()=>setConfirm(null)}>{t("common.cancel")}</Button>
  <Button disabled={confirm==="clear"?locked:!canRun} loading={mutation.isPending} variant="primary" onClick={()=>void run(confirm==="clear"?{action:"clear"}:{action:"run",mode:"retry_failed",idempotencyKey:createRequestKey()})}>{t(confirm==="clear"?"ocr.clear":"ocr.retry")}</Button>
 </>}><p>{t(confirm==="clear"?"ocr.clearDescription":"ocr.retryDescription")}</p></Dialog>;
 if(mode==="tasks")return <>
  <AIProcessingRow headingRef={headingRef} title={t("ocr.title")} description={t("ocr.taskDescription")} icon={<TextT aria-hidden="true" size={26}/>} maintenance={maintenance}
   action={job&&active?<Button disabled={locked||job.state==="cancelling"} loading={mutation.isPending} onClick={()=>void run({action:"cancel",jobId:job.id})}>{t("ocr.cancel")}</Button>:value&&!value.enabled?<Button onClick={onSettings}>{t("ai.tasks.enable")}</Button>:<Button disabled={!canRun||value?.coverage.pending===0} loading={mutation.isPending} onClick={()=>void run({action:"run",mode:"missing",idempotencyKey:createRequestKey()})}>{t("ocr.fill")}</Button>}>
   {status}
  </AIProcessingRow>{confirmation}
 </>;
 return <section className={styles.feature} aria-labelledby={heading}>
  <header className={styles.featureHeader}>
   <TextT className={styles.featureIcon} size={24} aria-hidden="true"/>
   <div><h2 id={heading} ref={headingRef} tabIndex={-1}>{t("ocr.title")}</h2><p>{t("ocr.description")}</p></div>
   {value&&<Switch aria-label={t("ocr.enable")} checked={value.enabled} disabled={locked||offline} onChange={event=>void run({action:"settings",enabled:event.currentTarget.checked})}/>}
  </header>
  {status}
  {value&&<><p className={styles.hint}>{t("ocr.enableHint")}</p><div className={styles.featureActions}>
   <Link className={styles.textLink} to={paths.librarySearch(libraryId)+"?mode=ocr"}>{t("ocr.openSearch")}<ArrowRight aria-hidden="true" size={16}/></Link>
   <Button variant="quiet" onClick={onTasks}>{t("ocr.manage")}</Button>
  </div></>}
  {confirmation}
 </section>;
}
