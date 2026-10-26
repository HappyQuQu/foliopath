import type {Page} from "@playwright/test";
import {readFileSync} from "node:fs";
const image=readFileSync(new URL("../../../../tests/fixtures/media/viewer-blue-violet.png",import.meta.url));
export async function fixture(page:Page,locale:string,options:{uncertain?:boolean;badImage?:boolean;uncertainUndo?:boolean;separate?:boolean;knownPair?:boolean;unknownPair?:boolean}={}){
 const writes:Record<string,unknown>[]=[],keys:string[]=[],unexpected:string[]=[];const receipts=new Map<string,{id:string;before:string;after:string;undone:boolean}>();const undoWrites:{reviewId:string;key:string;validator:string}[]=[];const undoReceipts=new Set<string>();
 const people=[{id:"person_source",name:"青岚",revision:7,confirmedFaceCount:1,assetCount:1,previewAssetId:"asset_photo"},{id:"person_target",name:"白露",revision:4,confirmedFaceCount:0,assetCount:0,previewAssetId:null}].map(person=>({...person,createdAt:"2026-10-25T00:00:00Z",updatedAt:"2026-10-25T00:00:00Z"}));
 const faces=[{faceId:"face_first",assetId:"asset_photo",ordinal:1,revision:3,state:"anonymous_core",personId:null as string|null,region:{xPercent:15,yPercent:15,widthPercent:20,heightPercent:30}},{faceId:"face_second",assetId:"asset_photo",ordinal:2,revision:5,state:"assigned",personId:"person_source" as string|null,region:{xPercent:60,yPercent:15,widthPercent:20,heightPercent:30}}];
 if(options.separate){faces[0]!.personId="person_target";faces[0]!.state="assigned";people[1]!.confirmedFaceCount=1;people[1]!.assetCount=1;people[1]!.previewAssetId="asset_photo";}
 const pairFaces=faces as (typeof faces[number]&{cannotLinkFaceIds?:string[]})[];if(options.separate&&!options.unknownPair)for(const face of pairFaces)face.cannotLinkFaceIds=options.knownPair?[faces.find(f=>f.faceId!==face.faceId)!.faceId]:[];
 const asset={id:"asset_photo",libraryId:"lib_photos",libraryName:"Synthetic",directoryId:"dir_photos",name:"synthetic.png",relativePath:"synthetic.png",kind:"image",mimeType:"image/png",sizeBytes:1024,width:1200,height:800,durationMs:null,modifiedAt:"2026-10-25T00:00:00Z",probeStatus:"ready",playbackStatus:"not_applicable",sourceAvailability:"available",thumbnail:{status:"ready",url:"/api/v1/assets/asset_photo/content",errorCode:null}};
 await page.addInitScript(locale=>localStorage.setItem("foliopath.preferences.v1",JSON.stringify({locale,theme:locale==="en"?"dark":"light"})),locale);
 await page.route("**/api/v1/**",async route=>{
  const req=route.request(),url=new URL(req.url()),path=url.pathname;let body:unknown,headers:Record<string,string>={};
  if(path.endsWith("/content"))return options.badImage?route.fulfill({status:404}):route.fulfill({contentType:"image/png",body:image});
  if(req.method()==="POST"&&path.endsWith("/undo")){
   const reviewId=path.split("/").at(-2)!;const key=req.headers()["idempotency-key"]!;const validator=req.headers()["if-match"]!;undoWrites.push({reviewId,key,validator});
   if(!undoReceipts.has(key)){
    const original=[...receipts.values()].find(r=>r.id===reviewId);
    if(!original||original.undone||validator!==`"face-review-${reviewId}-r2"`||JSON.stringify({faces,people})!==original.after)return route.fulfill({status:409,json:{error:{code:"face_constraint_conflict",message:"Synthetic later curation"}}});
    const before=JSON.parse(original.before);faces.splice(0,faces.length,...before.faces);people.splice(0,people.length,...before.people);original.undone=true;undoReceipts.add(key);
    if(options.uncertainUndo&&undoWrites.length===1){await new Promise(r=>setTimeout(r,150));return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic lost undo response after commit"}}});}
   }
   return route.fulfill({json:{reviewId:"undo_synthetic",action:"undo",affectedPersonIds:[],revision:1,undoable:false}});
  }
  if(req.method()==="POST"&&path==="/api/v1/ai/face-reviews"){
   const value=req.postDataJSON();writes.push(value);const key=req.headers()["idempotency-key"]!;keys.push(key);
   if(!receipts.has(key)){
    if(value.action==="cannot_link"){
     const left=pairFaces.find(f=>f.faceId===value.leftFaceId),right=pairFaces.find(f=>f.faceId===value.rightFaceId);const source=people.find(p=>p.id===left?.personId),target=people.find(p=>p.id===right?.personId);
     if(!left||!right||!source||!target||source.id===target.id||value.sourcePersonId!==source.id||value.targetPersonId!==target.id||value.expectedSourceRevision!==source.revision||value.expectedTargetRevision!==target.revision||value.expectedLeftRevision!==left.revision||value.expectedRightRevision!==right.revision||left.cannotLinkFaceIds?.includes(right.faceId))return route.fulfill({status:409,json:{error:{code:"face_constraint_conflict",message:"Synthetic pair changed"}}});
     const before=JSON.stringify({faces,people});left.cannotLinkFaceIds=[right.faceId];right.cannotLinkFaceIds=[left.faceId];receipts.set(key,{id:`review_synthetic_${writes.length}`,before,after:JSON.stringify({faces,people}),undone:false});
     if(options.uncertain&&writes.length===1)return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic lost pair response after commit"}}});
    }else{
    const face=faces.find(f=>f.faceId===value.faceId)!;const owner=people.find(p=>p.id===face.personId);const target=people.find(p=>p.id===value.personId);
    if(owner&&(value.sourcePersonId!==owner.id||value.expectedSourceRevision!==owner.revision)||!owner&&value.sourcePersonId||target&&target.revision!==value.expectedPersonRevision)return route.fulfill({status:409,json:{error:{code:"face_constraint_conflict",message:"Synthetic concurrent curation"}}});
    const before=JSON.stringify({faces,people});
    if(owner)owner.revision++;face.personId=value.action==="assign_face"?value.personId:null;face.state=value.action==="assign_face"?"assigned":value.action==="exclude_face"?"excluded":"anonymous_core";if(target)target.revision++;for(const person of people){const own=faces.filter(f=>f.personId===person.id);person.confirmedFaceCount=own.length;person.assetCount=own.length?1:0;person.previewAssetId=own.length?"asset_photo":null;}
    receipts.set(key,{id:`review_synthetic_${writes.length}`,before,after:JSON.stringify({faces,people}),undone:false});
    if(options.uncertain&&writes.length===1){await new Promise(resolve=>setTimeout(resolve,150));return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic lost response after save"}}});}
    }
   }
   return route.fulfill({json:{reviewId:receipts.get(key)!.id,action:value.action,affectedPersonIds:[],revision:2,undoable:true}});
  }
  if(path==="/api/v1/auth/session")body={administrator:{id:"admin_photo",username:"fixture",displayName:"Synthetic"},csrfToken:"synthetic-photo-token-not-a-credential",expiresAt:"2099-01-01T00:00:00Z"};
  else if(path==="/api/v1/libraries")body={items:[{id:"lib_photos",name:"Synthetic",status:"ready",assetCount:1,directoryCount:1},{id:"lib_wrong",name:"Offline other library",status:"offline",assetCount:1,directoryCount:1}],nextCursor:null};
  else if(path==="/api/v1/assets/asset_photo")body=asset;
  else if(path==="/api/v1/assets/asset_photo/faces")body={items:faces};
  else if(path==="/api/v1/people")body={items:people,nextCursor:null};
  else if(path.endsWith("/merge-check")){const source=people.find(p=>p.id===path.split("/").at(-2))!,target=people.find(p=>p.id===url.searchParams.get("targetPersonId"))!;const linked=pairFaces.some(f=>f.personId===source.id&&f.cannotLinkFaceIds?.some(id=>pairFaces.find(other=>other.faceId===id)?.personId===target.id));body={sourcePersonId:source.id,targetPersonId:target.id,sourceRevision:source.revision,targetRevision:target.revision,canMerge:!linked,cannotLinkCount:linked?1:0,duplicateAssignmentCount:0,sourceLimitExceeded:false,examples:linked?[{sourceLibraryId:"lib_photos",sourceAssetId:"asset_photo",targetLibraryId:"lib_photos",targetAssetId:"asset_photo"}]:[]};}
  else if(path.startsWith("/api/v1/people/person_")){body=people.find(p=>path.endsWith(p.id));headers={ETag:`"person-r${people.find(p=>path.endsWith(p.id))!.revision}"`};}
  else if(path==="/api/v1/catalog/state"){body={contentRevision:1};headers={ETag:'"catalog-r1"'};}
  else if(path.endsWith("/media-processing"))body={active:false,thumbnails:{total:0},videoPreviews:{total:0},videoPreviewsPendingEligibility:0};
  else if(path==="/api/v1/diagnostics/media-failures")body={items:[],nextCursor:null,revision:null};
  else if(path==="/api/v1/releases")body={currentVersion:"fixture",latestVersion:null,releases:[],checkedAt:"2026-10-25T00:00:00Z",updateAvailable:false};
  else{unexpected.push(`${req.method()} ${path}`);return route.fulfill({status:404,json:{error:{code:"not_found",message:"Unseeded fixture"}}});}
  await route.fulfill({json:body,headers});
 });
 return{writes,keys,unexpected,undoWrites,bumpSource(){people[0]!.revision++;},assignElsewhere(id:string){const face=faces.find(f=>f.faceId===id)!;face.personId="person_source";face.state="assigned";people[0]!.revision++;},faceState(id:string){return faces.find(f=>f.faceId===id)!.state;}};
}
