// CR-2026-035: production bundle + finite synthetic contract fixtures, not model quality.
import AxeBuilder from "@axe-core/playwright";
import { readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
const image = readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png", import.meta.url));
const library = {id: "lib_fused", name: "Synthetic search", status: "ready", displayPath: "/synthetic", assetCount: 3, directoryCount: 1, contentRevision: 1, latestScanId: null, lastSuccessfulScanAt: null, automaticDiscoveryStatus: "active", automaticDiscoveryErrorCode: null, lastAutomaticDiscoveryAt: null};
const hits = [
  {id: "asset_both", name: "car-blue.png", path: "synthetic/car-blue.png", sources: ["filename", "semantic", "ocr"], score: .32},
  {id: "asset_path", name: "scene.png", path: "car-trip/scene.png", sources: ["path"], score: null},
  {id: "asset_visual", name: "photo.png", path: "synthetic/photo.png", sources: ["semantic"], score: .27},
].map(hit => ({asset: {id: hit.id, name: hit.name, relativePath: hit.path, libraryId: library.id, libraryName: library.name, directoryId: "dir_fused", kind: "image", mimeType: "image/png", width: 1200, height: 800, durationMs: null, modifiedAt: "2026-10-25T00:00:00Z", sizeBytes: 1024, favorite: false, playbackStatus: "not_applicable", probeStatus: "ready", sourceAvailability: "available", storyboard: {cellHeight: null, cellWidth: null, columns: null, errorCode: null, frameCount: null, rows: null, status: "not_applicable", url: null}, thumbnail: {errorCode: null, status: "ready", url: "/synthetic-fused.png"}}, sources: hit.sources, semanticScore: hit.score, ocr: hit.id === "asset_both" ? [{line: 3,text: "car 京都 <script>alert(1)</script>"}] : null}));
for (const locale of ["zh-CN", "en"] as const) for (const width of [320, 390, 768, 1440, 1920]) {
  test(`${locale} fused evidence at ${width}px`, async ({page}, info) => {
    const en = locale === "en";
    const requests: URL[] = [], unexpected: string[] = [], errors: string[] = [];
    let degraded = false;
    page.on("pageerror", e => errors.push(e.message));
    await page.setViewportSize({width, height: 1000});
    await page.addInitScript(({locale}) => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({locale, theme: locale === "en" ? "dark" : "light"})), {locale});
    await page.route("**/synthetic-fused.png", r => r.fulfill({contentType: "image/png", body: image}));
    await page.route("**/health/ready", r => r.fulfill({json: {status: "ready"}}));
    await page.route("**/api/v1/**", async route => {
      const url = new URL(route.request().url()), path = url.pathname;
      let body: unknown, headers: Record<string, string> = {};
      if (route.request().method() !== "GET") unexpected.push(`write ${path}`);
      else if (path === "/api/v1/auth/session") body = {administrator: {id: "adm_fused", displayName: "Synthetic administrator", username: "fixture"}, csrfToken: "synthetic-csrf-token-not-a-credential", expiresAt: "2099-01-01T00:00:00Z"};
      else if (path === "/api/v1/catalog/state") {body = {contentRevision: 1}; headers = {ETag: '"catalog-r1"'};}
      else if (path === "/api/v1/libraries") body = {items: [library], nextCursor: null};
      else if (path === "/api/v1/libraries/lib_fused") {body = library; headers = {ETag: '"library-r1"'};}
      else if (path === "/api/v1/search/assets") {
        requests.push(url);
        body = {ocrState:"available",ocrCoverage:{eligible:3,ready:2,failed:1},items: degraded ? [{...hits[1], sources: ["path"], semanticScore: null}] : hits, nextCursor: null, semanticState: degraded ? "unavailable" : "ready", coverage: degraded ? null : {complete: false, completed: 2, eligible: 3, failed: 1, stale: 0, excludedLibraries: []}};
      } else if (path.endsWith("/content")) {await route.fulfill({contentType: "image/png", body: image});return;}
      else if (path.endsWith("/media-processing")) body = {active: false, thumbnails: {total: 0, processed: 0, queued: 0, running: 0, succeeded: 0, failed: 0}, videoPreviews: {total: 0, processed: 0, queued: 0, running: 0, succeeded: 0, failed: 0}, videoPreviewsPendingEligibility: 0};
      else if (path === "/api/v1/tags") body = {items: [], nextCursor: null};
      else if (/\/assets\/[^/]+\/curation$/.test(path)) body = {assetId: path.split("/")[4], revision: 1, favorite: false, tags: []};
      else if (path === "/api/v1/diagnostics/media-failures") body = {items: [], nextCursor: null, revision: null};
      else if (path === "/api/v1/releases") body = {currentVersion: "synthetic", latestVersion: null, releases: [], checkedAt: "2026-10-25T00:00:00Z", updateAvailable: false};
      else unexpected.push(path);
      if (body === undefined) await route.fulfill({status: 412, json: {error: {code: "precondition_failed", message: "Unseeded read-only fixture"}}});
      else await route.fulfill({json: body, headers});
    });
    await page.goto("/libraries/lib_fused/search?q=car&scope=directory&directoryId=dir_fused&recursive=1");
    await expect(page.getByRole("button", {name: "预览 car-blue.png"}).or(page.getByRole("button", {name: "Preview car-blue.png"}))).toBeVisible();
    expect(requests[0]!.searchParams.get("mode")).toBe("all");
    expect(requests[0]!.searchParams.get("directoryId")).toBe("dir_fused");
    expect(requests[0]!.searchParams.get("recursive")).toBe("true");
    const main = page.getByRole("main");
    const first = page.getByRole("button", {name: en ? "Preview car-blue.png" : "预览 car-blue.png"});
    await expect(page.getByRole("heading", { level: 1, name: en ? "Results for “car”" : "“car”的搜索结果" })).toBeVisible();
    const firstTop = (await first.boundingBox())!.y;
    expect(firstTop, "results must start in the upper part of the viewport").toBeLessThanOrEqual(width >= 1440 ? 340 : width >= 768 ? 360 : 520);

    await expect(main.locator('span[data-tone="accent"]')).toHaveCount(2);
    await expect(main.locator('span[data-tone="neutral"]')).toHaveCount(3);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    if (width === 1920) {
      const box = await main.boundingBox();
      expect(box!.width).toBeGreaterThan(1500);
    }
    await page.screenshot({path: info.outputPath(`fused-${locale}-${width}.png`), fullPage: true});
    const trigger = page.getByRole("button", {name: en ? "Preview car-blue.png" : "预览 car-blue.png"});
    await trigger.click();
    const preview = page.getByRole("complementary", {name: en ? "Preview: car-blue.png" : "预览: car-blue.png"});
    await expect(preview.getByRole("heading", {name: en ? "Why it matched" : "为什么匹配"})).toBeVisible();
    const evidence = preview.getByRole("region", {name: en ? "Why it matched" : "为什么匹配"});
    await expect(evidence.getByText(en ? "Filename" : "文件名", {exact: true})).toBeVisible();
    await expect(evidence.getByText(en ? "Visual content" : "画面内容", {exact: true})).toBeVisible();
    await expect(evidence.getByText(en ? "Image text" : "图片文字", {exact: true})).toBeVisible();
    await expect(evidence.getByText("car 京都 <script>alert(1)</script>")).toBeVisible();
    expect(await evidence.locator("script").count()).toBe(0);
    await expect(evidence.getByText(en ? "Library path" : "库内路径", {exact: true})).toHaveCount(0);
    if (width <= 1024) {
      await expect(preview).toBeFocused();
      await expect(preview.getByRole("button", {name: en ? "Close preview" : "关闭预览"})).toBeInViewport();
    }
    await evidence.scrollIntoViewIfNeeded();
    await expect(evidence.getByText(en ? "Visual content" : "画面内容", {exact: true})).toBeInViewport();
    await page.screenshot({path: info.outputPath(`fused-preview-${locale}-${width}.png`), fullPage: false});
    await preview.getByRole("button", {name: en ? "Close preview" : "关闭预览"}).click();
    await expect(trigger).toBeFocused();
    if (width <= 600) await page.getByRole("button", {name: en ? "Open search tools" : "打开搜索工具"}).click();
    const tools = page.getByRole("complementary", {name: en ? "Search tools" : "搜索工具", exact: true});
    await expect(tools.getByRole("radio", {name: en ? "All sources" : "全部来源"})).toBeChecked();
    await expect(tools.getByRole("radio", {name: en ? "Image text" : "图片文字"})).toBeEnabled();
    if (width <= 600) await page.keyboard.press("Escape");
    degraded = true;
    await page.getByRole("button", {name: en ? "Refresh search results" : "刷新搜索结果"}).click();
    await expect(page.getByText(en ? "Visual search is unavailable. You can still search filenames and image text." : "画面搜索暂不可用，仍可按文件名和图片文字查找。")).toBeVisible();
    await expect(main.locator('span[data-tone="accent"]')).toHaveCount(0);
    await expect(main.locator('span[data-tone="neutral"]')).toHaveCount(1);
    const violations = (await new AxeBuilder({page}).withTags(["wcag2a", "wcag2aa"]).analyze()).violations;
    expect(violations.map(v => ({id: v.id, targets: v.nodes.map(n => n.target)}))).toEqual([]);
    expect(errors).toEqual([]);expect(unexpected).toEqual([]);
  });
}
