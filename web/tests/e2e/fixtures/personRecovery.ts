// Synthetic bounded contracts for production PersonPanel, never developer media.
import type {Page} from "@playwright/test";
import {readFileSync} from "node:fs";
const image=readFileSync(new URL("../../../../tests/fixtures/media/viewer-blue-violet.png",import.meta.url));
export const recoveryEntry="/intelligence?view=people&person=person_source&libraryId=lib_recovery";
export async function personRecoveryFixture(page:Page,locale:"zh-CN"|"en",options:{mode?:"merge"|"remove"|"move";beforeCommit?:boolean;conflict?:boolean;mergeCheck?:"blocked"|"unavailable"|"changed"|"duplicates"|"limit"|"offline"}={}){
 let checkMode=options.mergeCheck;const checks:string[]=[];
 const source={id:"person_source",name:"青岚",revision:7,confirmedFaceCount:1,assetCount:1,previewAssetId:"asset_recovery" as string|null,createdAt:"2026-10-25T00:00:00Z",updatedAt:"2026-10-25T00:00:00Z"};
 const target={...source,id:"person_target",name:"白露",revision:4,confirmedFaceCount:0,assetCount:0,previewAssetId:null as string|null};
 const face={faceId:"face_recovery",assetId:"asset_recovery",ordinal:1,personId:source.id as string|null,state:"assigned",revision:3,region:{xPercent:15,yPercent:15,widthPercent:20,heightPercent:30}};
 const asset={id:"asset_recovery",libraryId:"lib_recovery",libraryName:"Synthetic",directoryId:"dir_recovery",name:"synthetic.png",relativePath:"synthetic.png",kind:"image",mimeType:"image/png",sizeBytes:1024,width:1200,height:800,durationMs:null,modifiedAt:"2026-10-25T00:00:00Z",probeStatus:"ready",playbackStatus:"not_applicable",sourceAvailability:"available",thumbnail:{status:"ready",url:"/api/v1/assets/asset_recovery/content",errorCode:null}};
 let merged=false,commits=0;const writes:{body:Record<string,unknown>;key:string}[]=[],unexpected:string[]=[],undos:{key:string;validator:string;reviewId:string}[]=[];
 const receipts=new Map<string,{reviewId:string;revision:number;action:string}>();const undoKeys=new Set<string>();
 await page.addInitScript(locale=>localStorage.setItem("foliopath.preferences.v1",JSON.stringify({locale,theme:locale==="en"?"dark":"light"})),locale);
 await page.route("**/api/v1/**",async route=>{
  const req=route.request(),path=new URL(req.url()).pathname;let body:unknown,headers:Record<string,string>={};
  if(path.endsWith("/content")||path==="/api/v1/assets/asset_recovery/thumbnail")return route.fulfill({contentType:"image/png",body:image});
  if(req.method()==="POST"&&path.endsWith("/undo")){
   const reviewId=path.split("/").at(-2)!,key=req.headers()["idempotency-key"]!,validator=req.headers()["if-match"]!;undos.push({key,validator,reviewId});
   if(!undoKeys.has(key)){
    if(![...receipts.values()].some(r=>r.reviewId===reviewId)||validator!==`"face-review-${reviewId}-r8"`)return route.fulfill({status:409,json:{error:{code:"face_constraint_conflict",message:"Synthetic stale receipt"}}});
    merged=false;source.revision++;source.confirmedFaceCount=1;source.assetCount=1;source.previewAssetId=asset.id;target.revision++;target.confirmedFaceCount=0;target.assetCount=0;target.previewAssetId=null;face.personId=source.id;face.state="assigned";undoKeys.add(key);
   }
   return route.fulfill({json:{reviewId:"undo_recovery",action:"undo",affectedPersonIds:[source.id,target.id],revision:1,undoable:false}});
  }
  if(req.method()==="POST"&&path==="/api/v1/ai/face-reviews"){
   const value=req.postDataJSON(),key=req.headers()["idempotency-key"]!;writes.push({body:value,key});const receipt=receipts.get(key);
   if(receipt)return route.fulfill({headers:{"Idempotency-Replayed":"true"},json:{...receipt,affectedPersonIds:[source.id,target.id],undoable:true}});
   if(options.beforeCommit&&writes.length===1)return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic response failure before commit"}}});
   if(options.conflict||value.expectedSourceRevision!==source.revision||value.sourcePersonId!==source.id||value.action!=="split_face"&&(value.action==="merge_people"?value.expectedTargetRevision:value.expectedPersonRevision)!==target.revision)return route.fulfill({status:409,json:{error:{code:"face_constraint_conflict",message:"Synthetic changed owner"}}});
   commits++;source.revision++;source.confirmedFaceCount=0;source.assetCount=0;source.previewAssetId=null;
   if(value.action!=="split_face"){target.revision++;target.confirmedFaceCount=1;target.assetCount=1;target.previewAssetId=asset.id;face.personId=target.id;face.state="assigned";}else{face.personId=null;face.state="unclustered";}
   if(value.action==="merge_people")merged=true;
   const saved={reviewId:"review_recovery",revision:8,action:value.action};receipts.set(key,saved);
   if(!options.beforeCommit&&writes.length===1){await new Promise(r=>setTimeout(r,150));return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic lost response after commit"}}});}
   return route.fulfill({json:{...saved,affectedPersonIds:[source.id,target.id],undoable:true}});
  }
  if(path==="/api/v1/auth/session")body={administrator:{id:"admin_recovery",username:"fixture",displayName:"Synthetic"},csrfToken:"synthetic-recovery-not-a-credential",expiresAt:"2099-01-01T00:00:00Z"};
  else if(path==="/api/v1/libraries")body={items:[{id:"lib_recovery",name:"Synthetic",status:"ready",assetCount:1,directoryCount:1}],nextCursor:null};
  else if(path==="/api/v1/people")body={items:merged?[target]:[source,target],nextCursor:null};
  else if(path.endsWith("/merge-check")){
   checks.push(req.url());if(checkMode==="unavailable")return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic read unavailable"}}});
   body={sourcePersonId:source.id,targetPersonId:target.id,sourceRevision:checkMode==="changed"?source.revision+1:source.revision,targetRevision:target.revision,canMerge:!checkMode||checkMode==="changed",cannotLinkCount:checkMode==="blocked"||checkMode==="offline"?5:0,duplicateAssignmentCount:checkMode==="duplicates"?2:0,sourceLimitExceeded:checkMode==="limit",examples:checkMode==="blocked"?[{sourceLibraryId:asset.libraryId,sourceAssetId:asset.id,targetLibraryId:"lib_other",targetAssetId:"asset_other"}]:[]};
  }
  else if(path==="/api/v1/people/person_source"||path==="/api/v1/people/person_source/assets"){
   if(merged)return route.fulfill({status:404,json:{error:{code:"person_not_found",message:"Synthetic merged source"}}});
   if(path.endsWith("/assets"))body={items:face.personId===source.id?[{asset,faceIds:[face.faceId]}]:[],nextCursor:null};else{body=source;headers={ETag:`"person-r${source.revision}"`};}
  }
  else if(path==="/api/v1/people/person_target"){body=target;headers={ETag:`"person-r${target.revision}"`};}
  else if(path==="/api/v1/assets/asset_recovery/faces")body={items:[face]};
  else if(path==="/api/v1/catalog/state"){body={contentRevision:1};headers={ETag:'"catalog-r1"'};}
  else if(path.endsWith("/media-processing"))body={active:false,thumbnails:{total:0},videoPreviews:{total:0},videoPreviewsPendingEligibility:0};
  else if(path==="/api/v1/diagnostics/media-failures")body={items:[],nextCursor:null,revision:null};
  else if(path==="/api/v1/releases")body={currentVersion:"fixture",latestVersion:null,releases:[],checkedAt:"2026-10-25T00:00:00Z",updateAvailable:false};
  else{unexpected.push(`${req.method()} ${path}`);return route.fulfill({status:404,json:{error:{code:"resource_not_found",message:"Unseeded fixture"}}});}
  await route.fulfill({json:body,headers});
 });
 return{writes,undos,unexpected,checks,allowMerge(){checkMode=undefined;},commits:()=>commits,owner:()=>face.personId,isMerged:()=>merged,bumpSource(){source.revision++;}};
}
