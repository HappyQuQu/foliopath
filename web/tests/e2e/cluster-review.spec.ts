// Production review UI, synthetic bounded API fixtures; no real media or curation.
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
const image = readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png", import.meta.url));
const entry = "/intelligence?view=clusters&libraryId=lib_group&cluster=cluster_review";
const members = [1, 2, 3].map(n => ({ faceId: `face_${n}`, assetId: n === 3 ? "asset_2" : "asset_1", revision: n + 2, kind: "core", region: { xPercent: n === 2 ? 60 : 15, yPercent: 15, widthPercent: 20, heightPercent: 30 } }));
for (const locale of ["zh-CN", "en"] as const) for (const width of [390, 768, 1440, 1920]) test(`${locale} one-face review at ${width}`, async ({ page }, info) => {
  await page.setViewportSize({ width, height: 1000 }); const state = await fixture(page, locale); await page.goto(entry);
  await expect(page.getByRole("article")).toHaveCount(2);
  const mark = locale === "en" ? "Photo 1 · Marker 2" : "照片 1 · 标记 2";
  const assign = page.getByRole("button", { name: locale === "en" ? "Confirm person assignment" : "确认归入人物", exact: true });
  await expect(assign).toBeDisabled(); await page.getByRole("button", { name: mark, exact: true }).click();
  await expect(page.getByRole("radio", { name: mark, exact: true })).toBeChecked();
  await page.getByRole("combobox", { name: locale === "en" ? "Target person" : "目标人物", exact: true }).selectOption("person_target");
  await assign.click(); const dialog = page.getByRole("dialog", { name: locale === "en" ? "Confirm this face belongs to the selected person?" : "确认这张人脸属于所选人物？", exact: true });
  await expect(dialog.getByText(mark, { exact: true })).toBeVisible();
  expect((await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze()).violations.map(v => v.id)).toEqual([]);
  await page.keyboard.press("Escape"); await expect(assign).toBeFocused(); expect(state.writes).toEqual([]);
  await assign.click(); await dialog.getByRole("button", { name: locale === "en" ? "Confirm save" : "确认保存", exact: true }).click();
  await expect.poll(() => state.writes.length).toBe(1);
  expect(state.writes[0]).toEqual({ action: "assign_face", faceId: "face_2", expectedFaceRevision: 4, personId: "person_target", expectedPersonRevision: 4 });
  await expect(page.getByRole("radio", { name: mark, exact: true })).toBeDisabled();
  await expect(page.getByText(locale === "en" ? "Assigned" : "已归类", { exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  expect((await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze()).violations.map(v => v.id)).toEqual([]);
  await page.keyboard.press("Control+Home"); await page.screenshot({ path: info.outputPath(`group-${locale}-${width}.png`), fullPage: true });
  expect(state.unexpected).toEqual([]);
});
test("creates only an empty person, and an uncertain save is explicitly retried with the same key", async ({ page }) => {
  const state = await fixture(page, "en", true); await page.goto(entry);
  await page.getByRole("button", { name: "Create person", exact: true }).click(); const creation = page.getByRole("dialog", { name: "Create person", exact: true });
  await creation.getByRole("textbox").fill("New synthetic person"); await creation.getByRole("button", { name: "Create person", exact: true }).click();
  await expect(page.getByText(/Created “New synthetic person” without assigning faces/)).toBeVisible();
  expect(state.creates).toEqual([{ name: "New synthetic person" }]); expect(state.writes).toEqual([]);
  await page.getByRole("radio", { name: "Photo 2 · Marker 1", exact: true }).check(); await page.getByRole("button", { name: "Exclude selected face", exact: true }).click(); const dialog = page.getByRole("dialog", { name: "Exclude this face?", exact: true });
  await dialog.getByRole("button", { name: "Confirm save", exact: true }).click(); await expect(dialog.getByText(/The save was not confirmed/)).toBeVisible(); expect(state.writes).toHaveLength(1);
  await dialog.getByRole("button", { name: "Confirm save", exact: true }).click(); await expect(dialog).not.toBeVisible();
  expect(state.keys[0]).toBe(state.keys[1]); expect(state.writes[0]).toEqual(state.writes[1]);
  await expect(page.getByText("Excluded", { exact: true })).toBeVisible(); expect(state.unexpected).toEqual([]);
});
test("conflict stops with no automatic resubmit, even after state refresh", async ({ page }) => {
  const state = await fixture(page, "en", false, true); await page.goto(entry);
  await page.getByRole("radio", { name: "Photo 1 · Marker 1", exact: true }).check(); await page.getByRole("button", { name: "Exclude selected face", exact: true }).click(); await page.getByRole("button", { name: "Confirm save", exact: true }).click();
  await expect(page.getByText(/State changed. Nothing was resubmitted/)).toBeVisible(); expect(state.writes).toHaveLength(1); expect(state.unexpected).toEqual([]);
});
test("group type and list cursor survive detail pagination and back navigation", async ({ page }) => {
  const state = await fixture(page, "en");
  await page.goto("/intelligence?view=clusters&libraryId=lib_group&kind=edge&review=all&cursor=legacy-groups");
  await expect(page).toHaveURL(/groupsCursor=legacy-groups/);
  await page.getByRole("button", { name: /Possibly similar.*3 faces/ }).click();
  await expect(page).toHaveURL(/kind=edge/); await expect(page).toHaveURL(/groupsCursor=legacy-groups/); await expect(page).toHaveURL(/cluster=cluster_review/); await expect(page).toHaveURL(/review=all/);
  await page.getByRole("button", { name: "Next page", exact: true }).click(); await expect(page).toHaveURL(/cursor=members-next/); await expect(page.getByRole("article")).toHaveCount(1);
  await page.getByRole("button", { name: "Back to list", exact: true }).click();
  await expect(page).toHaveURL(/groupsCursor=legacy-groups/); await expect(page).toHaveURL(/kind=edge/); expect(new URL(page.url()).searchParams.has("cursor")).toBe(false);
  expect(state.reads.some(url => url.pathname.endsWith("/face-clusters") && url.searchParams.get("cursor") === "legacy-groups")).toBe(true);
  await page.getByRole("button", { name: "Person to confirm", exact: true }).click(); await expect(page).not.toHaveURL(/groupsCursor|kind=/);
  expect(state.unexpected).toEqual([]);
});

for (const locale of ["zh-CN", "en"] as const) for (const width of [390,768,1440,1920]) test(`${locale} pending group directory at ${width}`, async ({page},info)=>{
 await page.setViewportSize({width,height:1000});const state=await fixture(page,locale);
 await page.goto("/intelligence?view=clusters&libraryId=lib_group&groupsCursor=old");
 await expect(page.getByText(locale==="en"?"Current type: 3 / 7 groups pending · 9 faces to review":"当前类型：3 / 7 组待处理 · 9 张人脸待确认")).toBeVisible();
 expect(state.reads.some(url=>url.searchParams.get("review")==="pending")).toBe(true);
 const filter=page.getByRole("group",{name:locale==="en"?"Group review status":"分组处理状态",exact:true});
 await filter.getByRole("button",{name:locale==="en"?"All groups":"全部分组",exact:true}).click();
 await expect(page).toHaveURL(/review=all/);await expect(page).not.toHaveURL(/groupsCursor/);
 await expect.poll(()=>state.reads.some(url=>url.searchParams.get("review")==="all"&&!url.searchParams.has("cursor"))).toBe(true);
 await expect(filter.getByRole("button",{name:locale==="en"?"All groups":"全部分组",exact:true})).toHaveAttribute("aria-pressed","true");
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);
 expect((await new AxeBuilder({page}).withTags(["wcag2a","wcag2aa"]).analyze()).violations.map(v=>v.id)).toEqual([]);
 await page.screenshot({path:info.outputPath(`directory-${locale}-${width}.png`),fullPage:true});expect(state.writes).toEqual([]);expect(state.unexpected).toEqual([]);
});

test("empty pending group directory preserves access to reviewed history",async({page})=>{
 const state=await fixture(page,"en",false,false,true);await page.goto("/intelligence?view=clusters&libraryId=lib_group");
 await expect(page.getByText("No pending groups of this type",{exact:true})).toBeVisible();
 await expect(page.getByText("Current type: 0 / 7 groups pending · 0 faces to review",{exact:true})).toBeVisible();
 await page.getByRole("group",{name:"Group review status",exact:true}).getByRole("button",{name:"All groups",exact:true}).click();
 await expect(page.getByText("Pending 0 / 3 faces",{exact:true})).toBeVisible();expect(state.writes).toEqual([]);
});

async function fixture(page: Page, locale: string, uncertain = false, conflict = false, emptyDirectory = false) {
  const writes: Record<string, unknown>[] = [], creates: Record<string, unknown>[] = [], keys: string[] = [], unexpected: string[] = [], reads: URL[] = [];
  let groupKind = "core";
  const states = new Map<string, string>(); const people = [{ id: "person_target", name: "青岚", revision: 4, confirmedFaceCount: 0, assetCount: 0 }];
  await page.addInitScript(locale => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale, theme: locale === "en" ? "dark" : "light" })), locale);
  await page.route("**/health/ready", route => route.fulfill({ json: { status: "ready" } }));
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname; let body: unknown, headers: Record<string, string> = {};
    if (request.method() === "GET") reads.push(url);
    if (path.endsWith("/content") || path.endsWith("/thumbnail")) return route.fulfill({ contentType: "image/png", body: image });
    if (request.method() === "POST" && path === "/api/v1/people") { const value = request.postDataJSON(); creates.push(value); const person = { id: "person_created", name: value.name, revision: 1, confirmedFaceCount: 0, assetCount: 0 }; people.push(person); return route.fulfill({ status: 201, json: person, headers: { ETag: '"person-r1"' } }); }
    if (request.method() === "POST" && path === "/api/v1/ai/face-reviews") {
      const value = request.postDataJSON(); writes.push(value); keys.push(request.headers()["idempotency-key"] ?? "");
      if (conflict) return route.fulfill({ status: 412, json: { error: { code: "precondition_failed", message: "Synthetic conflict" } } });
      if (uncertain && writes.length === 1) return route.fulfill({ status: 503, json: { error: { code: "internal_error", message: "Unconfirmed synthetic response" } } });
      states.set(value.faceId, value.action === "exclude_face" ? "excluded" : "assigned");
      return route.fulfill({ json: { reviewId: "review_synthetic", action: value.action, affectedPersonIds: ["person_target"], revision: 5, undoable: true, replayed: false } });
    }
    if (path === "/api/v1/auth/session") body = { administrator: { id: "admin_group", username: "fixture", displayName: "Synthetic" }, csrfToken: "synthetic-group-token-not-a-credential", expiresAt: "2099-01-01T00:00:00Z" };
    else if (path === "/api/v1/catalog/state") { body = { contentRevision: 1 }; headers = { ETag: '"catalog-r1"' }; }
    else if (path === "/api/v1/libraries") body = { items: [{ id: "lib_group", name: "Synthetic", status: "ready", assetCount: 2, directoryCount: 1 }], nextCursor: null };
    else if (path === "/api/v1/people") body = { items: people, nextCursor: null };
    else if (/^\/api\/v1\/people\/person_/.test(path)) { body = people.find(p => path.endsWith(p.id)); headers = { ETag: '"person-r4"' }; }
    else if (path === "/api/v1/libraries/lib_group/ai/face-clusters") { groupKind = url.searchParams.get("kind") ?? "core"; body = { items: emptyDirectory && url.searchParams.get("review") === "pending" ? [] : [{ id: "cluster_review", libraryId: "lib_group", kind: groupKind, memberCount: 3, pendingMemberCount: emptyDirectory ? 0 : 3, revision: 1, previewAssetIds: ["asset_1", "asset_2"] }], nextCursor: emptyDirectory && url.searchParams.get("review") === "pending" ? null : "groups-next", groupAssignmentAllowed: false, counts: { totalGroups: 7, pendingGroups: emptyDirectory ? 0 : 3, totalFaces: 20, pendingFaces: emptyDirectory ? 0 : 9 }, coverage: { complete: true, eligible: 2, completed: 2, failed: 0, stale: 0, degraded: 0, revision: 1 } }; }
    else if (path === "/api/v1/libraries/lib_group/ai/face-clusters/cluster_review") body = { cluster: { id: "cluster_review", libraryId: "lib_group", kind: groupKind, memberCount: 3, pendingMemberCount: emptyDirectory ? 0 : 3, revision: 1, previewAssetIds: ["asset_1", "asset_2"] }, items: (url.searchParams.get("cursor") ? members.slice(2) : members).map(m => ({ ...m, kind: groupKind })), nextCursor: url.searchParams.get("cursor") ? null : "members-next" };
    else if (/^\/api\/v1\/assets\/asset_[12]\/faces$/.test(path)) body = { items: members.filter(m => path.includes(`/${m.assetId}/`)).map((m, index) => ({ ...m, ordinal: index + 1, state: states.get(m.faceId) ?? `anonymous_${groupKind}`, personId: states.get(m.faceId) === "assigned" ? "person_target" : null })) };
    else if (path.endsWith("/media-processing")) body = { active: false, thumbnails: { total: 0 }, videoPreviews: { total: 0 }, videoPreviewsPendingEligibility: 0 };
    else if (path === "/api/v1/diagnostics/media-failures") body = { items: [], nextCursor: null, revision: null };
    else if (path === "/api/v1/releases") body = { currentVersion: "fixture", latestVersion: null, releases: [], checkedAt: "2026-10-25T00:00:00Z", updateAvailable: false };
    else { unexpected.push(`${request.method()} ${path}`); return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "Unseeded fixture" } } }); }
    await route.fulfill({ json: body, headers });
  });
  return { writes, creates, keys, unexpected, reads };
}
