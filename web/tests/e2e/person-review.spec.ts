// Production UI with bounded synthetic contracts; no developer media or real curation.
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";

const image = readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png", import.meta.url));
const person = { id: "person_source", name: "青岚", revision: 7, assetCount: 1, confirmedFaceCount: 2 };
const target = { id: "person_target", name: "白露", revision: 4, assetCount: 1, confirmedFaceCount: 1 };
const asset = { id: "asset_pair", libraryId: "lib_review", libraryName: "Synthetic", directoryId: "dir_review", name: "synthetic-pair.png", relativePath: "synthetic-pair.png", kind: "image", mimeType: "image/png", width: 1200, height: 800, durationMs: null, sizeBytes: 1024, modifiedAt: "2026-10-25T00:00:00Z", probeStatus: "ready", playbackStatus: "not_applicable", sourceAvailability: "available", thumbnail: { status: "ready", url: "/synthetic-person.png", errorCode: null } };
const entry = "/intelligence?view=people&libraryId=lib_review&person=person_source";
for (const locale of ["zh-CN", "en"] as const) for (const width of [390, 768, 1440, 1920]) {
  test(`${locale} explicit face and merge confirmation at ${width}`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 1000 });
    const state = await fixture(page, locale);
    await page.goto(entry);
    const photo = page.getByRole("article", { name: asset.name });
    const remove = photo.getByRole("button", { name: locale === "en" ? "Remove from person" : "从人物中移除", exact: true });
    await expect(remove).toBeDisabled();
    const choose = photo.getByRole("combobox", { name: locale === "en" ? "Face to organize" : "要整理的人脸", exact: true });
    await choose.selectOption("face_2");
    await expect(photo.getByRole("button", { name: locale === "en" ? "Select face 2" : "选择人脸 2", exact: true })).toHaveAttribute("aria-pressed", "true");
    await expect(photo.getByRole("button", { name: locale === "en" ? "Select face 2" : "选择人脸 2", exact: true })).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
    await remove.click();
    const removal = page.getByRole("dialog", { name: locale === "en" ? "Remove this face from the person?" : "从人物中移出这张人脸？", exact: true });
    await expect(removal.getByText(locale === "en" ? "Face 2" : "人脸 2", { exact: true })).toBeVisible();
    await page.keyboard.press("Escape"); await expect(remove).toBeFocused(); expect(state.writes).toEqual([]);
    await remove.click(); await removal.getByRole("button", { name: locale === "en" ? "Confirm assignment change" : "确认修改关联", exact: true }).click();
    await expect.poll(() => state.writes.length).toBe(1);
    expect(state.writes[0]).toMatchObject({ action: "split_face", faceId: "face_2", expectedFaceRevision: 4, sourcePersonId: person.id, expectedSourceRevision: 7 });
    const mergeButton = page.getByRole("button", { name: locale === "en" ? "Merge people" : "合并人物", exact: true });
    await mergeButton.click();
    const merge = page.getByRole("dialog", { name: locale === "en" ? "Merge people" : "合并人物", exact: true });
    await merge.getByRole("combobox").selectOption(target.id);
    const confirm = merge.getByRole("button", { name: locale === "en" ? "Confirm merge" : "确认合并", exact: true });
    await expect(confirm).toBeDisabled(); await merge.getByRole("checkbox").check();
    await expect(confirm).toBeEnabled();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    expect((await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze()).violations.map(v => v.id)).toEqual([]);
    await page.screenshot({ path: info.outputPath(`merge-${locale}-${width}.png`), fullPage: false });
    await page.keyboard.press("Escape"); await expect(mergeButton).toBeFocused();
    await page.keyboard.press("Control+Home");
    await page.screenshot({ path: info.outputPath(`person-${locale}-${width}.png`), fullPage: true });
    expect(state.unexpected).toEqual([]);
  });
}
test("a merge conflict is shown without retrying or adopting a newer revision", async ({ page }) => {
  const state = await fixture(page, "en"); await page.goto(entry);
  await page.getByRole("button", { name: "Merge people", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Merge people", exact: true }); await dialog.getByRole("combobox").selectOption(target.id);
  await dialog.getByRole("checkbox").check(); await dialog.getByRole("button", { name: "Confirm merge", exact: true }).click();
  await expect(page.getByText("State changed elsewhere. It was refreshed; review again.", { exact: true })).toBeVisible();
  expect(state.writes).toEqual([{ action: "merge_people", sourcePersonId: person.id, targetPersonId: target.id, expectedSourceRevision: 7, expectedTargetRevision: 4, conflictsAcknowledged: true }]);
  await expect(page).toHaveURL(/person=person_source/); expect(state.unexpected).toEqual([]);
});

test("single-photo review reuses the same transparent, keyboard-selectable face control", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 1000 });
  const state = await fixture(page, "en");
  await page.goto("/intelligence?view=faces&libraryId=lib_review&asset=asset_pair");
  const box = page.getByRole("button", { name: "Face 2 · Assigned", exact: true }).first();
  await expect(box).toBeEnabled(); await box.focus(); await page.keyboard.press("Enter");
  await expect(box).toHaveAttribute("aria-pressed", "true");
  await expect(box).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  expect(state.writes).toEqual([]); expect(state.unexpected).toEqual([]);
});

async function fixture(page: Page, locale: "zh-CN" | "en") {
  const writes: Record<string, unknown>[] = []; const unexpected: string[] = [];
  await page.addInitScript(locale => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale, theme: locale === "en" ? "dark" : "light" })), locale);
  await page.route("**/synthetic-person.png", route => route.fulfill({ contentType: "image/png", body: image }));
  await page.route("**/health/ready", route => route.fulfill({ json: { status: "ready" } }));
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/v1/assets/asset_pair/content") return route.fulfill({ contentType: "image/png", body: image });
    if (route.request().method() === "POST" && path === "/api/v1/ai/face-reviews") {
      const request = route.request().postDataJSON(); writes.push(request);
      if (request.action === "merge_people") return route.fulfill({ status: 409, json: { error: { code: "face_constraint_conflict", message: "Synthetic conflict" } } });
      return route.fulfill({ json: { reviewId: "review_fixture", action: request.action, affectedPersonIds: [person.id], revision: 8, undoable: true, replayed: false } });
    }
    let body: unknown; let headers: Record<string, string> = {};
    if (path === "/api/v1/auth/session") body = { administrator: { id: "admin_review", username: "fixture", displayName: "Synthetic" }, csrfToken: "synthetic-review-token-not-a-credential", expiresAt: "2099-01-01T00:00:00Z" };
    else if (path === "/api/v1/catalog/state") { body = { contentRevision: 1 }; headers = { ETag: '"catalog-r1"' }; }
    else if (path === "/api/v1/libraries") body = { items: [{ id: "lib_review", name: "Synthetic", status: "ready", assetCount: 1, directoryCount: 1 }], nextCursor: null };
    else if (path === "/api/v1/people") body = { items: [person, target], nextCursor: null };
    else if (path.endsWith("/merge-check")) body = {sourcePersonId:person.id,targetPersonId:target.id,sourceRevision:person.revision,targetRevision:target.revision,canMerge:true,cannotLinkCount:0,duplicateAssignmentCount:0,sourceLimitExceeded:false,examples:[]};
    else if (path === "/api/v1/people/person_source") { body = person; headers = { ETag: '"person-r7"' }; }
    else if (path === "/api/v1/people/person_source/assets") body = { items: [{ asset, faceIds: ["face_1", "face_2"] }], nextCursor: null };
    else if (path === "/api/v1/assets/asset_pair") body = asset;
    else if (path === "/api/v1/assets/asset_pair/faces") body = { assetId: asset.id, items: [1, 2].map(ordinal => ({ faceId: `face_${ordinal}`, assetId: asset.id, ordinal, personId: person.id, revision: ordinal + 2, state: "assigned", region: { xPercent: ordinal === 1 ? 15 : 60, yPercent: 15, widthPercent: 20, heightPercent: 30 } })) };
    else if (path.endsWith("/media-processing")) body = { active: false, thumbnails: { total: 0 }, videoPreviews: { total: 0 }, videoPreviewsPendingEligibility: 0 };
    else if (path === "/api/v1/diagnostics/media-failures") body = { items: [], nextCursor: null, revision: null };
    else if (path === "/api/v1/releases") body = { currentVersion: "fixture", latestVersion: null, releases: [], checkedAt: "2026-10-25T00:00:00Z", updateAvailable: false };
    else { unexpected.push(`${route.request().method()} ${path}`); return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "Unseeded fixture" } } }); }
    await route.fulfill({ json: body, headers });
  });
  return { writes, unexpected };
}
