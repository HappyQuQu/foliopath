// Production people directory, synthetic bounded contracts. Never developer media.
import AxeBuilder from "@axe-core/playwright";
import {expect,test,type Page} from "@playwright/test";
import {readFileSync} from "node:fs";
const image=readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png",import.meta.url));
const people=[{id:"person_directory_a",name:"青岚",confirmedFaceCount:2,assetCount:1,previewAssetId:"ast_1"},{id:"person_directory_b",name:"青岚",confirmedFaceCount:1,assetCount:1,previewAssetId:"ast_2"},{id:"person_directory_empty",name:"LongName".repeat(12),confirmedFaceCount:0,assetCount:0,previewAssetId:null},{id:"person_directory_tail",name:"白露",confirmedFaceCount:0,assetCount:0,previewAssetId:null}].map(p=>({...p,revision:1,createdAt:"2026-10-25T00:00:00Z",updatedAt:"2026-10-25T00:00:00Z"}));
const entry="/intelligence?view=people&libraryId=lib_directory";
for(const locale of ["zh-CN","en"] as const)for(const width of [390,768,1440,1920])test(`${locale} named people directory at ${width}`,async({page},info)=>{
 await page.setViewportSize({width,height:1000});const state=await fixture(page,locale);await page.goto(entry);const main=page.getByRole("main");
 await expect(main.getByText(locale==="en"?"People on this page: 3":"本页 3 位人物",{exact:true})).toBeVisible();
 await expect(main.locator("button").filter({hasText:"青岚"})).toHaveCount(2);
 await expect(main.locator('img[src*="/ast_1/thumbnail"]')).toBeVisible();await expect(main.getByText(locale==="en"?"No photo preview available":"暂无可用照片预览",{exact:true})).toBeVisible();
 const card=main.locator("button").filter({hasText:"青岚"}).first();const cardBox=(await card.boundingBox())!;const photoBox=(await card.locator("img").boundingBox())!;expect(photoBox.y-cardBox.y).toBeLessThanOrEqual(2);
 expect(state.reads.filter(u=>u.pathname.endsWith("/people")).length).toBe(1);expect(state.reads.some(u=>u.pathname.endsWith("/assets"))).toBe(false);
 expect((await new AxeBuilder({page}).withTags(["wcag2a","wcag2aa"]).analyze()).violations.map(v=>v.id)).toEqual([]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);await page.screenshot({path:info.outputPath(`people-${locale}-${width}.png`),fullPage:true});
 await main.getByRole("button",{name:locale==="en"?"Next page":"下一页",exact:true}).click();await expect(page).toHaveURL(/peopleCursor=directory-next/);await expect(main.getByText("白露",{exact:true})).toBeVisible();
 await main.getByLabel(locale==="en"?"Search people by name":"搜索人物名称",{exact:true}).fill("青岚");await expect(page).not.toHaveURL(/peopleCursor/);await expect(main.getByText(locale==="en"?"People on this page: 2":"本页 2 位人物",{exact:true})).toBeVisible();
 await main.locator("button").filter({hasText:"青岚"}).first().click();await expect(page).toHaveURL(/person=person_directory_a/);await expect(page).toHaveURL(/q=/);
 await page.getByRole("button",{name:locale==="en"?"Back to list":"返回列表",exact:true}).click();await expect(main.getByLabel(locale==="en"?"Search people by name":"搜索人物名称",{exact:true})).toHaveValue("青岚");expect(state.writes).toEqual([]);expect(state.unexpected).toEqual([]);
});
test("thumbnail failures keep names and counts usable",async({page})=>{
 await fixture(page,"en",{badPhoto:true});await page.goto(entry);const main=page.getByRole("main");await expect(main.getByText("No photo preview available",{exact:true})).toHaveCount(3);await expect(main.getByText("2 confirmed faces · 1 assets",{exact:true})).toBeVisible();
});
test("filtered empty results clear the search without changing people",async({page})=>{
 const state=await fixture(page,"en");await page.goto(entry+"&q=missing");const main=page.getByRole("main");await expect(main.getByText("No matching people",{exact:true})).toBeVisible();expect(await page.getByText("No named people yet",{exact:true}).count()).toBe(0);await main.getByRole("button",{name:"Clear search",exact:true}).click();await expect(main.getByText("People on this page: 3",{exact:true})).toBeVisible();expect(state.writes).toEqual([]);
});
test("uncertain empty-person creation reuses its key, while cancel and double click stay safe",async({page})=>{
 const state=await fixture(page,"en",{uncertain:true});await page.goto(entry);const main=page.getByRole("main");const trigger=main.getByRole("button",{name:"Create person",exact:true});await trigger.click();let dialog=page.getByRole("dialog",{name:"Create person",exact:true});await dialog.getByRole("textbox").fill("Synthetic new");await page.keyboard.press("Escape");await expect(trigger).toBeFocused();expect(state.writes).toEqual([]);
 await trigger.click();dialog=page.getByRole("dialog",{name:"Create person",exact:true});await dialog.getByRole("button",{name:"Create person",exact:true}).dblclick();await expect(dialog.getByText(/Creation has not been confirmed/)).toBeVisible();expect(state.writes).toHaveLength(1);
 await dialog.getByRole("button",{name:"Create person",exact:true}).click();await expect(page).toHaveURL(/person=person_directory_created/);expect(state.writes).toHaveLength(2);expect(state.keys[1]).toBe(state.keys[0]);expect(state.writes[0]).toEqual({name:"Synthetic new"});expect(state.unexpected).toEqual([]);
});
test("invalid people cursors return to the first page with the search intact",async({page})=>{
 const state=await fixture(page,"en",{stale:true});await page.goto(entry+"&q=青岚&peopleCursor=expired");const main=page.getByRole("main");await main.getByRole("button",{name:"Try again",exact:true}).click();await expect(page).not.toHaveURL(/peopleCursor/);await expect(main.getByLabel("Search people by name",{exact:true})).toHaveValue("青岚");await expect(main.getByText("People on this page: 2",{exact:true})).toBeVisible();expect(state.writes).toEqual([]);
});
async function fixture(page:Page,locale:string,options:{badPhoto?:boolean;uncertain?:boolean;stale?:boolean}={}){
 const reads:URL[]=[],writes:unknown[]=[],keys:string[]=[],unexpected:string[]=[];const items=[...people];
 await page.addInitScript(locale=>localStorage.setItem("foliopath.preferences.v1",JSON.stringify({locale,theme:locale==="en"?"dark":"light"})),locale);
 await page.route("**/api/v1/**",async route=>{
  const req=route.request(),url=new URL(req.url()),path=url.pathname;reads.push(url);let body:unknown,headers:Record<string,string>={};
  if(path.includes("/thumbnail"))return options.badPhoto?route.fulfill({status:404}):route.fulfill({body:image,contentType:"image/png"});
  if(path.endsWith("/content"))return route.fulfill({body:image,contentType:"image/png"});
  if(req.method()==="POST"&&path==="/api/v1/people"){
   writes.push(req.postDataJSON());keys.push(req.headers()["idempotency-key"]!);if(writes.length===1&&options.uncertain){await new Promise(resolve=>setTimeout(resolve,150));return route.fulfill({status:503,json:{error:{code:"internal_error",message:"Synthetic unconfirmed response"}}});}
   const p={...people[2]!,id:"person_directory_created",name:req.postDataJSON().name};items.push(p);return route.fulfill({status:201,json:p,headers:{ETag:'"person-r1"'}});
  }
  if(path==="/api/v1/auth/session")body={administrator:{id:"admin_directory",username:"fixture",displayName:"Synthetic"},csrfToken:"synthetic-directory-token-not-a-credential",expiresAt:"2099-01-01T00:00:00Z"};
  else if(path==="/api/v1/libraries")body={items:[{id:"lib_directory",name:"Synthetic",status:"ready",assetCount:2,directoryCount:1}],nextCursor:null};
  else if(path==="/api/v1/catalog/state"){body={contentRevision:1};headers={ETag:'"catalog-r1"'};}
  else if(path==="/api/v1/people"){
   if(options.stale&&url.searchParams.get("cursor"))return route.fulfill({status:400,json:{error:{code:"invalid_cursor",message:"Synthetic expired cursor"}}});
   const q=(url.searchParams.get("query")??"").trim();const matches=q?items.filter(p=>p.name.includes(q)):url.searchParams.get("cursor")?[items[3]!]:items.slice(0,3);body={items:matches,nextCursor:q||url.searchParams.get("cursor")?null:"directory-next"};
  }
  else if(/^\/api\/v1\/people\/person_/.test(path)){
   const p=items.find(p=>path.includes(p.id))!;headers={ETag:'"person-r1"'};
   body=path.endsWith("/assets")?{items:p.previewAssetId?[{asset:{id:p.previewAssetId,libraryId:"lib_directory",libraryName:"Synthetic",directoryId:"dir_synthetic",name:"synthetic.png",relativePath:"synthetic.png",kind:"image",mimeType:"image/png",width:1200,height:800,durationMs:null,sizeBytes:1024,modifiedAt:"2026-10-25T00:00:00Z",probeStatus:"ready",playbackStatus:"not_applicable",sourceAvailability:"available",thumbnail:{status:"ready",url:`/api/v1/assets/${p.previewAssetId}/thumbnail?variant=grid`,errorCode:null}},faceIds:Array.from({length:p.confirmedFaceCount},(_,i)=>`face_directory_${p.id}_${i}`)}]:[],nextCursor:null}:p;
  }
  else if(path.endsWith("/faces")){const p=items.find(p=>p.previewAssetId&&path.includes(p.previewAssetId))!;body={items:Array.from({length:p.confirmedFaceCount},(_,i)=>({faceId:`face_directory_${p.id}_${i}`,assetId:p.previewAssetId,personId:p.id,ordinal:i+1,revision:1,state:"assigned",region:{xPercent:15+i*40,yPercent:15,widthPercent:20,heightPercent:30}}))};}
  else if(path.endsWith("/curation")){body={assetId:path.split("/")[4],favorite:false,revision:1,tags:[],favoritedAt:null};headers={ETag:'"curation-r1"'};}
  else if(path.endsWith("/media-processing"))body={active:false,thumbnails:{total:0},videoPreviews:{total:0},videoPreviewsPendingEligibility:0};
  else if(path==="/api/v1/diagnostics/media-failures")body={items:[],nextCursor:null,revision:null};
  else if(path==="/api/v1/releases")body={currentVersion:"fixture",latestVersion:null,releases:[],checkedAt:"2026-10-25T00:00:00Z",updateAvailable:false};
  else{unexpected.push(`${req.method()} ${path}`);return route.fulfill({status:404,json:{error:{code:"not_found",message:"Unseeded fixture"}}});}
  await route.fulfill({json:body,headers});
 });return{reads,writes,keys,unexpected};
}
