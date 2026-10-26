// Production pages with bounded synthetic API fixtures; never reads developer media.
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";

const image = readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png", import.meta.url));
const library = { id: "lib_context", name: "合成媒体库", status: "ready", displayPath: "/synthetic", assetCount: 3, directoryCount: 1, contentRevision: 1, latestScanId: null, lastSuccessfulScanAt: null, automaticDiscoveryStatus: "active", automaticDiscoveryErrorCode: null, lastAutomaticDiscoveryAt: null };
const tag = { id: "tag_car", name: "car", assetCount: 2, createdAt: "2026-10-24T00:00:00Z", updatedAt: "2026-10-24T00:00:00Z", localization: { key: "car", zhCN: "汽车", en: "Car" } };
const assets = ["image", "video"].map((kind, index) => ({ id: `asset_${index}`, libraryId: library.id, libraryName: library.name, directoryId: "dir_context", name: `synthetic-${index}.${index ? "mp4" : "png"}`, relativePath: `synthetic-${index}.${index ? "mp4" : "png"}`, kind, mimeType: index ? "video/mp4" : "image/png", width: 1200, height: 800, durationMs: index ? 3000 : null, modifiedAt: "2026-10-24T00:00:00Z", sizeBytes: 1024, favorite: true, playbackStatus: index ? "playable" : "not_applicable", probeStatus: "ready", sourceAvailability: "available", storyboard: { cellHeight: null, cellWidth: null, columns: null, errorCode: null, frameCount: null, rows: null, status: "not_applicable", url: null }, thumbnail: { errorCode: null, status: "ready", url: "/synthetic-context.png" } }));
const routes = {
  search: "/libraries/lib_context/search?q=synthetic&mode=filename&scope=directory&directoryId=dir_context&recursive=1",
  people: "/intelligence?view=people&libraryId=lib_context",
  favorites: "/favorites?libraryId=lib_context",
  tags: "/tags/tag_car?libraryId=lib_context&kind=image",
};
const labels = { search: ["搜索工具", "Search tools"], people: ["人物工具", "People tools"], favorites: ["收藏筛选", "Favorite filters"], tags: ["标签筛选", "Tag filters"] };

for (const locale of ["zh-CN", "en"] as const) for (const width of [390, 768, 1440, 1920]) {
  test(`${locale} contextual workspaces at ${width}px`, async ({ page }, info) => {
    test.skip(info.project.name === "context-visual-chromium" && process.platform !== "linux", "Pixel baselines use the pinned Linux browser environment.");
    await page.setViewportSize({ width, height: 1000 });
    const requests = await fixtures(page, locale);
    for (const key of Object.keys(routes) as (keyof typeof routes)[]) {
      await page.goto(routes[key]);
      const label = labels[key][locale === "en" ? 1 : 0]!;
      const aside = page.getByRole("complementary", { name: label, exact: true });
      const opener = page.getByRole("button", { name: locale === "en" ? `Open ${label.toLowerCase()}` : `打开${label}`, exact: true });
      if (width <= 600) {
        await expect(aside).not.toBeVisible();
        await opener.click();
      }
      await expect(aside).toBeVisible();
      await expect(aside.getByRole("navigation", { name: locale === "en" ? "Quick access" : "快速访问" })).toBeVisible();
      await expect(page.getByRole("navigation", { name: locale === "en" ? "Directory navigation" : "目录导航" })).toHaveCount(0);
      if (key === "people") {
        await expect(aside.getByText(locale === "en" ? "All libraries" : "全部媒体库", { exact: true })).toBeVisible();
        await expect(aside.getByRole("navigation", { name: locale === "en" ? "Established people" : "已建立人物列表" }).getByRole("button")).toHaveCount(2);
      } else if (key === "tags") {
        await expect(aside.getByRole("link", { name: /Car|汽车/ })).toBeVisible();
      } else if (key === "search") {
        await expect(aside.getByRole("radio", { name: locale === "en" ? "Current directory" : "当前目录" })).toBeChecked();
      }
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      const violations = (await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze()).violations;
      expect(violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
      if (info.project.name === "context-visual-chromium") await expect(aside).toHaveScreenshot(`${key}-${locale}-${width}.png`, { animations: "disabled" });
      await page.screenshot({ path: info.outputPath(`${key}-${locale}-${width}.png`), fullPage: true });
      if (width <= 600) {
        await page.keyboard.press("Escape");
        await expect(opener).toBeFocused();
        await expect(aside).not.toBeVisible();
      }
    }
    expect(requests.unexpected).toEqual([]);
  });
}

test("sidebar filters use real URL owners, counts, and independent person cursors", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const requests = await fixtures(page, "zh-CN");
  await page.goto(routes.favorites);
  const favorites = page.getByRole("complementary", { name: "收藏筛选", exact: true });
  const video = favorites.getByRole("button", { name: /视频/ });
  await expect(video).toHaveText(/1/);
  await video.click();
  await expect(page.getByRole("button", { name: "预览 synthetic-1.mp4", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "预览 synthetic-0.png", exact: true })).toHaveCount(0);
  await favorites.getByRole("combobox", { name: "排序" }).selectOption("name");
  await expect(page).toHaveURL(/kind=video&sort=name/);
  await expect.poll(() => requests.urls.some(u => u.pathname === "/api/v1/favorites" && u.searchParams.get("kind") === "video" && u.searchParams.get("sort") === "name")).toBe(true);
  await page.goto(`${routes.people}&person=person_1&peopleCursor=people-next&cursor=assets-next`);
  const people = page.getByRole("complementary", { name: "人物工具", exact: true });
  await expect(people.getByRole("button", { name: /青岚/ })).toBeVisible();
  await expect.poll(() => requests.urls.some(u => u.pathname === "/api/v1/people/person_1/assets" && u.searchParams.get("cursor") === "assets-next")).toBe(true);
  expect(requests.urls.some(u => u.pathname === "/api/v1/people" && u.searchParams.get("cursor") === "people-next")).toBe(true);
  await people.getByRole("searchbox").fill("白露");
  await expect(page).not.toHaveURL(/peopleCursor/);
  await expect(page).toHaveURL(/cursor=assets-next/);
  await expect(people.getByRole("button", { name: /青岚/ })).toHaveCount(0);
  await people.getByRole("button", { name: "待整理", exact: true }).click();
  await expect(people.getByRole("button", { name: /媒体库：合成媒体库/ })).toBeVisible();
  await people.getByRole("button", { name: "可能相似", exact: true }).click();
  await expect(page).toHaveURL(/kind=edge/);
  await page.goto(routes.search);
  const search = page.getByRole("complementary", { name: "搜索工具", exact: true });
  await search.getByRole("radio", { name: "全部媒体库" }).click();
  await expect(page).toHaveURL(/directoryId=dir_context/);
  await expect.poll(() => requests.urls.some(u => u.pathname === "/api/v1/assets" && !u.searchParams.has("directoryId"))).toBe(true);
  await search.getByRole("radio", { name: "当前目录" }).click();
  await expect(search.getByRole("radio", { name: "当前目录" })).toBeChecked();
  await search.getByRole("radio", { name: "画面内容", exact: true }).click();
  await expect(page.getByRole("searchbox", { name: "描述画面内容" })).toHaveValue("synthetic");
  await expect(page).toHaveURL(/directoryId=dir_context/);
  await expect(search.getByRole("combobox", { name: "排序" })).toHaveCount(0);
  await page.goto(routes.tags);
  const tags = page.getByRole("complementary", { name: "标签筛选", exact: true });
  await expect(page.getByRole("heading", { level: 1, name: "汽车", exact: true })).toBeVisible();
  await tags.getByRole("combobox", { name: "AI 标签显示语言" }).selectOption("en");
  await expect(page.getByRole("heading", { level: 1, name: "Car", exact: true })).toBeVisible();
  await expect(page).toHaveURL(/tags\/tag_car/);
  await expect(page.getByRole("button", { name: "预览 synthetic-0.png", exact: true })).toBeVisible();
  await tags.getByRole("link", { name: "清除标签筛选" }).click();
  await expect(page).toHaveURL(/\/tags\?libraryId=lib_context/);
  expect(requests.unexpected).toEqual([]);
});

async function fixtures(page: Page, locale: "zh-CN" | "en") {
  const urls: URL[] = []; const unexpected: string[] = [];
  await page.addInitScript(({ locale }) => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale, theme: locale === "en" ? "dark" : "light" })), { locale });
  await page.route("**/synthetic-context.png", route => route.fulfill({ contentType: "image/png", body: image }));
  await page.route("**/health/ready", route => route.fulfill({ json: { status: "ready" } }));
  await page.route("**/api/v1/**", async route => {
    const url = new URL(route.request().url()); urls.push(url);
    let body: unknown; let headers: Record<string, string> = {};
    const path = url.pathname;
    if (route.request().method() !== "GET") { unexpected.push(`${route.request().method()} ${path}`); return route.fulfill({ status: 412, json: { error: { code: "precondition_failed", message: "Read-only fixture" } } }); }
    const people = [{ id: "person_1", name: "青岚", assetCount: 2, confirmedFaceCount: 3, revision: 1 }, { id: "person_2", name: "白露", assetCount: 1, confirmedFaceCount: 1, revision: 1 }];
    const curation = (asset: typeof assets[number]) => ({ assetId: asset.id, favorite: true, favoritedAt: "2026-10-24T00:00:00Z", revision: 1, tags: [{ ...tag, source: "ai" }] });
    if (path === "/api/v1/auth/session") body = { administrator: { displayName: "合成管理员", id: "adm_context", username: "fixture" }, csrfToken: "synthetic-context-csrf-token-not-a-credential", expiresAt: "2099-01-01T00:00:00Z" };
    else if (path === "/api/v1/catalog/state") { body = { contentRevision: 1 }; headers = { ETag: '"catalog-r1"' }; }
    else if (path === "/api/v1/libraries") body = { items: [library], nextCursor: null };
    else if (path.endsWith("/media-processing")) body = { active: false, thumbnails: { total: 0, processed: 0, queued: 0, running: 0, succeeded: 0, failed: 0 }, videoPreviews: { total: 0, processed: 0, queued: 0, running: 0, succeeded: 0, failed: 0 }, videoPreviewsPendingEligibility: 0 };
    else if (path === "/api/v1/libraries/lib_context") { body = library; headers = { ETag: '"library-r1"' }; }
    else if (path === "/api/v1/diagnostics/media-failures") body = { items: [], nextCursor: null, revision: null };
    else if (path === "/api/v1/releases") body = { currentVersion: "synthetic-local", latestVersion: null, releases: [], checkedAt: "2026-10-24T00:00:00Z", updateAvailable: false };
    else if (path === "/api/v1/people") body = { items: people.filter(p => p.name.includes(url.searchParams.get("query") ?? "")), nextCursor: null };
    else if (/^\/api\/v1\/people\/[^/]+\/assets$/.test(path)) body = { items: [], nextCursor: null };
    else if (/^\/api\/v1\/people\/[^/]+$/.test(path)) { body = people[0]; headers = { ETag: '"person-r1"' }; }
    else if (path.endsWith("/ai/face-clusters")) body = { items: [], nextCursor: null, groupAssignmentAllowed: false, coverage: { complete: true, eligible: 2, completed: 2, failed: 0, stale: 0, degraded: 0, revision: 1 } };
    else if (path === "/api/v1/tags") body = { items: [tag], nextCursor: null };
    else if (path === "/api/v1/favorites" || path === "/api/v1/tags/tag_car/assets") {
      const kinds = (url.searchParams.get("kind") ?? "").split(",");
      body = { items: assets.filter(a => !url.searchParams.has("kind") || kinds.includes(a.kind)).map(asset => ({ asset, curation: curation(asset) })), counts: { all: 2, images: 1, videos: 1 }, nextCursor: null };
    } else if (path === "/api/v1/assets" || path === "/api/v1/libraries/lib_context/assets" || path === "/api/v1/semantic/assets") body = { items: [assets[0]], nextCursor: null, ...(path.includes("semantic") ? { coverage: { complete: true, eligible: 2, completed: 2, failed: 0, stale: 0, excludedLibraries: [] } } : {}) };
    else { unexpected.push(path); return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "Unseeded synthetic fixture" } } }); }
    await route.fulfill({ json: body, headers });
  });
  return { urls, unexpected };
}

test("mobile favorite filtering keeps tools open until the user closes the drawer", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 1000 });
  await fixtures(page, "zh-CN");
  await page.goto(routes.favorites);
  const opener = page.getByRole("button", { name: "打开收藏筛选", exact: true });
  await opener.click();
  const sidebar = page.getByRole("complementary", { name: "收藏筛选", exact: true });
  await sidebar.getByRole("button", { name: /视频/ }).click();
  await expect(page).toHaveURL(/kind=video/);
  await expect(opener).toHaveAttribute("aria-expanded", "true");
  await sidebar.getByRole("combobox", { name: "排序" }).selectOption("name");
  await expect(page).toHaveURL(/sort=name/);
  await page.keyboard.press("Escape");
  await expect(sidebar).not.toBeVisible();
  await expect(opener).toBeFocused();
});
