import { expect, test } from "@playwright/test";

test("real isolated group review creates empty people and changes only explicitly confirmed faces", async ({ page }, info) => {
  test.skip(process.env.FOLIOPATH_GROUP_REVIEW_REAL !== "1", "Run the isolated cluster_review_smoke.mjs probe.");
  await page.addInitScript(() => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale: "en", theme: "light" })));
  await page.goto("/"); await page.getByLabel("Username *").fill("person-review-test"); await page.getByLabel("Password *").fill(process.env.FOLIOPATH_PERSON_REVIEW_PASSWORD!); await page.getByRole("button", { name: "Log in", exact: true }).click(); await expect(page).not.toHaveURL(/login/);
  const url = `/intelligence?view=clusters&libraryId=${process.env.FOLIOPATH_PERSON_REVIEW_LIBRARY}&cluster=${process.env.FOLIOPATH_CLUSTER_REVIEW_ID}`;
  await page.goto(url); await expect(page.getByRole("article")).toHaveCount(2);
  await expect(page.getByRole("button", { name: "Confirm person assignment", exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Create person", exact: true }).click(); const creation = page.getByRole("dialog", { name: "Create person", exact: true }); await creation.getByRole("textbox").fill("Reviewed synthetic");
  const created = page.waitForResponse(r => r.url().endsWith("/people") && r.request().method() === "POST"); await creation.getByRole("button", { name: "Create person", exact: true }).click(); const createdResponse = await created;
  expect(createdResponse.status()).toBe(201); expect(await createdResponse.json()).toMatchObject({ confirmedFaceCount: 0, assetCount: 0 }); expect(createdResponse.request().postDataJSON()).toEqual({ name: "Reviewed synthetic" });
  await page.getByRole("radio", { name: "Photo 1 · Marker 1", exact: true }).check(); await page.getByRole("button", { name: "Confirm person assignment", exact: true }).click(); const confirm = page.getByRole("dialog", { name: "Confirm this face belongs to the selected person?", exact: true });
  await expect(confirm.getByText("Assign to: Reviewed synthetic", { exact: true })).toBeVisible(); const assigned = page.waitForResponse(r => r.url().endsWith("/ai/face-reviews") && r.request().method() === "POST"); await confirm.getByRole("button", { name: "Confirm save", exact: true }).click(); const assignResponse = await assigned;
  expect(assignResponse.status()).toBe(200); expect(assignResponse.request().postDataJSON()).toMatchObject({ action: "assign_face", faceId: process.env.FOLIOPATH_CLUSTER_REVIEW_FIRST_FACE });
  await expect(page.getByRole("radio", { name: "Photo 1 · Marker 1", exact: true })).toBeDisabled(); await expect(page.getByText("Assigned", { exact: true })).toBeVisible();
  await page.getByRole("radio", { name: "Photo 2 · Marker 1", exact: true }).check(); await page.getByRole("button", { name: "Exclude selected face", exact: true }).click(); const excluded = page.waitForResponse(r => r.url().endsWith("/ai/face-reviews") && r.request().method() === "POST"); await page.getByRole("dialog", { name: "Exclude this face?", exact: true }).getByRole("button", { name: "Confirm save", exact: true }).click(); const excludeResponse = await excluded;
  expect(excludeResponse.status()).toBe(200); expect(excludeResponse.request().postDataJSON()).toMatchObject({ action: "exclude_face", faceId: process.env.FOLIOPATH_CLUSTER_REVIEW_SECOND_FACE });
  await expect(page.getByText("Excluded", { exact: true })).toBeVisible(); await page.reload(); await expect(page.getByText("Assigned", { exact: true })).toBeVisible(); await expect(page.getByText("Excluded", { exact: true })).toBeVisible();
  await page.getByRole("button", {name:"Back to list",exact:true}).click();
  await expect(page.getByText("Current type: 1 / 2 groups pending · 2 faces to review",{exact:true})).toBeVisible();
  await expect(page.getByText("Pending 2 / 2 faces",{exact:true})).toHaveCount(1);
  await page.getByRole("group",{name:"Group review status",exact:true}).getByRole("button",{name:"All groups",exact:true}).click();
  await expect(page.getByText("Pending 0 / 2 faces",{exact:true})).toBeVisible();
  await page.setViewportSize({ width: 1920, height: 1000 }); await page.screenshot({ path: info.outputPath("cluster-real-desktop.png"), fullPage: true }); await page.setViewportSize({ width: 390, height: 1000 }); expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true); await page.screenshot({ path: info.outputPath("cluster-real-mobile.png"), fullPage: true });
  if(process.env.FOLIOPATH_PEOPLE_DIRECTORY_REAL === "1") {
    await page.goto(`/intelligence?view=people&libraryId=${process.env.FOLIOPATH_PERSON_REVIEW_LIBRARY}`);const main=page.getByRole("main");
    await expect(main.locator(`img[src="/api/v1/assets/${process.env.FOLIOPATH_DIRECTORY_PREVIEW_ASSET}/thumbnail?variant=grid"]`)).toBeVisible();
    await main.getByRole("button",{name:"Create person",exact:true}).click();const dialog=page.getByRole("dialog",{name:"Create person",exact:true});await dialog.getByRole("textbox").fill("Empty directory fixture");
    const response=page.waitForResponse(r=>r.url().endsWith("/people")&&r.request().method()==="POST");await dialog.getByRole("button",{name:"Create person",exact:true}).click();expect(await (await response).json()).toMatchObject({assetCount:0,confirmedFaceCount:0,previewAssetId:null});
    await page.getByRole("button",{name:"Back to list",exact:true}).click();await expect(main.getByText("People on this page: 2",{exact:true})).toBeVisible();await expect(main.getByText("No photo preview available",{exact:true})).toBeVisible();
    await page.setViewportSize({width:1920,height:1000});await page.screenshot({path:info.outputPath("people-real-1920.png"),fullPage:true});
  }

});
