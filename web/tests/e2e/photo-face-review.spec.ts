// Production single-photo UI with explicitly synthetic bounded API contracts.
import AxeBuilder from "@axe-core/playwright";
import {expect,test} from "@playwright/test";
import {fixture} from "./fixtures/photoReview";
const entry="/intelligence?view=faces&libraryId=lib_wrong&asset=asset_photo";
for(const locale of ["zh-CN","en"] as const)for(const width of [390,768,1440,1920])test(`${locale} single-photo review at ${width}`,async({page},info)=>{
 await page.setViewportSize({width,height:1000});const state=await fixture(page,locale);await page.goto(entry);await expect(page).toHaveURL(/libraryId=lib_photos/);
 const english=locale==="en";const radio=page.getByRole("radio",{name:english?"Face 2 · Assigned":"人脸 2 · 已归类",exact:true});await expect(radio).toBeEnabled();await expect(radio).not.toBeChecked();await radio.check();
 const box=page.getByRole("button",{name:english?"Face 2 · Assigned":"人脸 2 · 已归类",exact:true});await expect(box).toHaveAttribute("aria-pressed","true");await expect(box).toHaveCSS("background-color","rgba(0, 0, 0, 0)");
 await page.getByRole("combobox",{name:english?"Target person":"目标人物",exact:true}).selectOption("person_target");const trigger=page.getByRole("button",{name:english?"Review person assignment":"核对归入人物",exact:true});await trigger.click();const dialog=page.getByRole("dialog");
 await expect(dialog.getByText(english?"Source person: 青岚":"来源人物： 青岚",{exact:true})).toBeVisible();await expect(dialog.getByText(english?"Target person: 白露":"目标人物： 白露",{exact:true})).toBeVisible();
 expect((await new AxeBuilder({page}).withTags(["wcag2a","wcag2aa"]).analyze()).violations.map(v=>v.id)).toEqual([]);await page.keyboard.press("Escape");await expect(trigger).toBeFocused();expect(state.writes).toEqual([]);
 await trigger.click();await dialog.getByRole("button",{name:english?"Confirm save":"确认保存",exact:true}).click();await expect(page.getByText(english?"The selected face was saved. Other faces were not changed in a batch.":"所选人脸已保存，其他人脸未批量修改。",{exact:true})).toBeVisible();
 expect(state.writes).toEqual([{action:"assign_face",faceId:"face_second",expectedFaceRevision:5,personId:"person_target",expectedPersonRevision:4,sourcePersonId:"person_source",expectedSourceRevision:7}]);expect(state.faceState("face_first")).toBe("anonymous_core");
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);await page.screenshot({path:info.outputPath(`single-${locale}-${width}.png`),fullPage:true});expect(state.unexpected).toEqual([]);
});
test("an anonymous exclusion cannot erase a concurrent assignment",async({page})=>{
 const state=await fixture(page,"en");await page.goto(entry);await page.getByRole("radio",{name:"Face 1 · To confirm",exact:true}).check();await page.getByRole("button",{name:"Exclude this face",exact:true}).click();state.assignElsewhere("face_first");await page.getByRole("dialog").getByRole("button",{name:"Confirm save",exact:true}).click();await expect(page.getByText("Curation changed. Nothing was resubmitted. Refreshed; select and check again.",{exact:true})).toBeVisible();expect(state.writes).toEqual([{action:"exclude_face",faceId:"face_first",expectedFaceRevision:3}]);expect(state.faceState("face_first")).toBe("assigned");
});
test("assigned exclusion freezes the source revision and does not adopt a newer one",async({page})=>{
 const state=await fixture(page,"en");await page.goto(entry);await page.getByRole("radio",{name:"Face 2 · Assigned",exact:true}).check();await page.getByRole("button",{name:"Exclude this face",exact:true}).click();state.bumpSource();await page.getByRole("dialog").getByRole("button",{name:"Confirm save",exact:true}).click();await expect(page.getByText(/Curation changed\./)).toBeVisible();expect(state.writes[0]).toEqual({action:"exclude_face",faceId:"face_second",expectedFaceRevision:5,sourcePersonId:"person_source",expectedSourceRevision:7});expect(state.faceState("face_second")).toBe("assigned");expect(state.writes).toHaveLength(1);
});
test("lost success responses retry the identical payload and operation key",async({page})=>{
 const state=await fixture(page,"en",{uncertain:true});await page.goto(entry);await page.getByRole("radio",{name:"Face 2 · Assigned",exact:true}).check();await page.getByRole("button",{name:"Exclude this face",exact:true}).click();const dialog=page.getByRole("dialog");await dialog.getByRole("button",{name:"Confirm save",exact:true}).dblclick();await expect(dialog.getByText(/Saving has not been confirmed/)).toBeVisible();expect(state.writes).toHaveLength(1);await dialog.getByRole("button",{name:"Confirm save",exact:true}).click();await expect(dialog).not.toBeVisible();expect(state.keys[1]).toBe(state.keys[0]);expect(state.writes[1]).toEqual(state.writes[0]);expect(state.faceState("face_second")).toBe("excluded");
});
test("failed photo loads pause curation",async({page})=>{
 const state=await fixture(page,"en",{badImage:true});await page.goto(entry);await expect(page.getByText(/The photo could not load/)).toBeVisible();await expect(page.getByRole("radio",{name:"Face 2 · Assigned",exact:true})).toBeDisabled();await expect(page.getByRole("button",{name:"Exclude this face",exact:true})).toBeDisabled();expect(state.writes).toEqual([]);
});
