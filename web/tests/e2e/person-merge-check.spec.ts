// Production dialog against labelled, bounded synthetic read contracts.
import AxeBuilder from "@axe-core/playwright";
import {expect,test,type Page} from "@playwright/test";
import {personRecoveryFixture,recoveryEntry} from "./fixtures/personRecovery";
async function open(page:Page,en=true){await page.getByRole("button",{name:en?"Merge people":"合并人物",exact:true}).click();const d=page.getByRole("dialog");await d.getByRole("combobox").selectOption("person_target");return d;}
for(const locale of ["zh-CN","en"] as const)for(const width of [390,768,1440,1920])test(`${locale} pre-merge reasons at ${width}`,async({page},info)=>{
 await page.setViewportSize({width,height:1000});const state=await personRecoveryFixture(page,locale,{mergeCheck:"blocked"});const en=locale==="en";await page.goto(recoveryEntry);const trigger=page.getByRole("button",{name:en?"Merge people":"合并人物",exact:true});const d=await open(page,en);
 await expect(d.getByText(en?"Face pairs marked as different people: 5.":"有 5 对人脸被标记为不同人物。",{exact:true})).toBeVisible();await expect(d.getByText(en?"Available photo examples: 1. This is not the complete relationship list.":"以下为 1 组可用照片示例，并非全部人脸关系。",{exact:true})).toBeVisible();await expect(d.getByRole("checkbox")).toBeDisabled();await expect(d.getByRole("button",{name:en?"Confirm merge":"确认合并",exact:true})).toBeDisabled();await expect(d.getByRole("link",{name:en?"Review target photo (new tab)":"核对目标照片（新页）",exact:true})).toHaveAttribute("href","/libraries/lib_other/media/asset_other");expect((await new AxeBuilder({page}).withTags(["wcag2a","wcag2aa"]).analyze()).violations.map(v=>v.id)).toEqual([]);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);await page.screenshot({path:info.outputPath(`merge-check-${locale}-${width}.png`),fullPage:true});
 await page.keyboard.press("Escape");await expect(trigger).toBeFocused();expect(state.writes).toEqual([]);expect(state.checks).toHaveLength(1);expect(new URL(state.checks[0]!).searchParams.get("targetPersonId")).toBe("person_target");expect(state.unexpected).toEqual([]);
});
test("unavailable reads pause merge and retry only the read",async({page})=>{
 const state=await personRecoveryFixture(page,"en",{mergeCheck:"unavailable"});await page.goto(recoveryEntry);const d=await open(page);await expect(d.getByText("Merge checks are unavailable. Confirmation is paused.",{exact:true})).toBeVisible();await expect(d.getByRole("checkbox")).toBeDisabled();state.allowMerge();await d.getByRole("button",{name:"Try again",exact:true}).click();await expect(d.getByRole("checkbox")).toBeEnabled();await expect(d.getByRole("button",{name:"Confirm merge",exact:true})).toBeDisabled();expect(state.writes).toEqual([]);expect(state.checks).toHaveLength(2);
});
test("changed person versions never silently replace the confirmed snapshot",async({page})=>{
 const state=await personRecoveryFixture(page,"en",{mergeCheck:"changed"});await page.goto(recoveryEntry);const d=await open(page);await expect(d.getByText(/Person versions changed\./)).toBeVisible();await expect(d.getByRole("checkbox")).toBeDisabled();await expect(d.getByRole("button",{name:"Confirm merge",exact:true})).toBeDisabled();await d.getByRole("button",{name:"Check again",exact:true}).click();await expect(d.getByRole("checkbox")).toBeDisabled();expect(state.writes).toEqual([]);
});
test("offline constraints remain blocking when no photo can be shown",async({page})=>{
 const state=await personRecoveryFixture(page,"en",{mergeCheck:"offline"});await page.goto(recoveryEntry);const d=await open(page);await expect(d.getByText(/Offline or detached relationships still block merging\./)).toBeVisible();await expect(d.getByRole("link")).toHaveCount(0);await expect(d.getByRole("button",{name:"Confirm merge",exact:true})).toBeDisabled();expect(state.writes).toEqual([]);
});
test("duplicate assignments have a distinct explanation",async({page})=>{
 const state=await personRecoveryFixture(page,"en",{mergeCheck:"duplicates"});await page.goto(recoveryEntry);const d=await open(page);await expect(d.getByText("Duplicate face assignments: 2.",{exact:true})).toBeVisible();await expect(d.getByRole("checkbox")).toBeDisabled();expect(state.writes).toEqual([]);
});

test("atomic batch limits are shown without offering force merge",async({page})=>{
 const state=await personRecoveryFixture(page,"en",{mergeCheck:"limit"});await page.goto(recoveryEntry);const d=await open(page);await expect(d.getByText("The source exceeds the current atomic merge limit. Review assignments individually.",{exact:true})).toBeVisible();await expect(d.getByRole("checkbox")).toBeDisabled();expect(state.writes).toEqual([]);
});
