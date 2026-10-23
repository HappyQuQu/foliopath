// Production bundle UI regression with synthetic API fixtures, not model-quality evidence.
import { readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";

const image = readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png", import.meta.url));
for (const locale of ["zh-CN", "en"] as const) for (const width of [320, 390, 768, 1440, 1920]) {
  test(`${locale} search mode guidance at ${width}px`, async ({ page }, testInfo) => {
    const english = locale === "en";
    const semanticLabel = english ? "Describe visual content" : "描述画面内容";
    const literalLabel = english ? "Search filenames or paths" : "搜索文件名或路径";
    const semanticSource = english ? "Visual semantic search" : "画面语义检索";
    const literalSource = english ? "Filename or path search" : "文件名或路径检索";
    const errors: string[] = [];
    const requests: URL[] = [];
    const unsupported: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    await page.setViewportSize({ width, height: 960 });
    await page.addInitScript(({ locale }) => window.localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale, theme: locale === "en" ? "dark" : "light" })), { locale });
    await page.route("**/health/ready", route => route.fulfill({ json: { status: "ready" } }));
    await page.route("**/synthetic-search-image.png", route => route.fulfill({ contentType: "image/png", body: image }));
    await page.route("**/api/v1/**", async route => {
      const url = new URL(route.request().url());
      if (route.request().method() !== "GET") {
        unsupported.push(`unexpected write ${url.pathname}`);
        await route.fulfill({ status: 412, json: { error: { code: "precondition_failed", message: "Read-only synthetic fixture" } } });
        return;
      }
      let body: unknown;
      let headers: Record<string, string> = {};
      switch (url.pathname) {
        case "/api/v1/auth/session": body = { administrator: { displayName: "Synthetic administrator", id: "adm_fixture", username: "fixture" }, csrfToken: "synthetic-csrf-token-not-a-credential", expiresAt: "2099-01-01T00:00:00Z" }; break;
        case "/api/v1/catalog/state": body = { contentRevision: 1 }; headers = { ETag: '"catalog-r1"' }; break;
        case "/api/v1/libraries": body = { items: [], nextCursor: null }; break;
        case "/api/v1/diagnostics/media-failures": body = { items: [], nextCursor: null, revision: null }; break;
        case "/api/v1/releases": body = { currentVersion: "synthetic-local", latestVersion: null, releases: [], checkedAt: "2026-10-23T00:00:00Z", updateAvailable: false }; break;
        case "/api/v1/libraries/lib_fixture": body = { id: "lib_fixture", name: "Synthetic library", status: "ready", displayPath: "/synthetic", assetCount: 3, directoryCount: 1, contentRevision: 1, latestScanId: null, lastSuccessfulScanAt: null, automaticDiscoveryStatus: "active", automaticDiscoveryErrorCode: null, lastAutomaticDiscoveryAt: null }; headers = { ETag: '"library-r1"' }; break;
        case "/api/v1/semantic/assets":
        case "/api/v1/libraries/lib_fixture/assets": {
          requests.push(url);
          const semantic = url.pathname.includes("/semantic/");
          body = { items: [{ id: "ast_fixture", libraryId: "lib_fixture", libraryName: "Synthetic library", directoryId: "dir_fixture", name: "synthetic-car.png", relativePath: "synthetic/synthetic-car.png", kind: "image", mimeType: "image/png", width: 1200, height: 800, durationMs: null, modifiedAt: "2026-10-23T00:00:00Z", sizeBytes: 1024, favorite: false, playbackStatus: "not_applicable", probeStatus: "ready", sourceAvailability: "available", storyboard: { cellHeight: null, cellWidth: null, columns: null, errorCode: null, frameCount: null, rows: null, status: "not_applicable", url: null }, thumbnail: { errorCode: null, status: "ready", url: "/synthetic-search-image.png" } }], nextCursor: null,
            ...(semantic ? { coverage: { complete: false, completed: 2, eligible: 3, failed: 1, stale: 0, excludedLibraries: [] } } : {}) };
          break;
        }
        default:
          unsupported.push(url.pathname);
          await route.fulfill({ status: 404, json: { error: { code: "not_found", message: "Unseeded synthetic fixture" } } });
          return;
      }
      await route.fulfill({ json: body, headers });
    });
    await page.goto("/libraries/lib_fixture/search?mode=semantic&scope=directory&directoryId=dir_fixture&recursive=1");
    const input = page.getByRole("searchbox", { name: semanticLabel });
    await expect(input).toBeVisible();
    // No horizontal overflow alone did not catch a one-character mobile input.
    expect((await input.boundingBox())!.width).toBeGreaterThanOrEqual(width === 320 ? 110 : 160);
    const header = page.getByRole("banner");
    const heading = page.getByRole("heading", { level: 1 });
    expect((await heading.boundingBox())!.y).toBeGreaterThanOrEqual((await header.boundingBox())!.height);
    if (width <= 480) {
      const submit = page.getByRole("button", { name: english ? "Search" : "搜索", exact: true });
      await expect(submit).toBeVisible();
      await input.press("Tab");
      await expect(submit).toBeFocused();
      await submit.press("Tab");
      const people = page.getByRole("link", { name: english ? "People" : "人物", exact: true });
      await expect(people).toBeFocused();
      expect((await people.boundingBox())!.y).toBeGreaterThan((await input.boundingBox())!.y);
    }
    await expect(page.getByText(english ? "Describe a scene in Chinese or English. Search uses analyzed images or video frames in the selected scope, not filenames." : "用中文或英文描述想找的画面。只检索所选范围内已分析的图片或视频关键帧，不按文件名匹配。")).toBeVisible();
    expect(requests).toHaveLength(0);
    await input.fill("车");
    await input.press("Enter");
    await expect(page.getByText(semanticSource, { exact: true })).toBeVisible();
    await expect(page.getByText(english ? "Ranked by visual similarity, not exact keyword matches. Image text recognition (OCR) is not connected." : "按画面相似度排序，不代表关键词精确命中。图片文字识别（OCR）尚未接入。")).toBeVisible();
    expect(requests).toHaveLength(1);
    expect(requests[0]!.pathname).toBe("/api/v1/semantic/assets");
    expect(requests[0]!.searchParams.get("q")).toBe("车");
    expect(requests[0]!.searchParams.get("recursive")).toBe("true");
    expect(requests[0]!.searchParams.get("directoryId")).toBe("dir_fixture");
    await expectNoOverflow(page);
    // A fixed two-line footer used to squash all three visible identity rows.
    expect(await page.getByText(semanticSource, { exact: true }).evaluate(element => {
      const identity = element.parentElement!;
      const parent = identity.getBoundingClientRect();
      const rows = Array.from(identity.children).map(child => ({ rect: child.getBoundingClientRect(), scrollHeight: child.scrollHeight }));
      return rows.every((row, index) => row.rect.height >= row.scrollHeight - 1 && row.rect.top >= parent.top && row.rect.bottom <= parent.bottom + 1 && (index === 0 || row.rect.top >= rows[index - 1]!.rect.bottom));
    })).toBe(true);
    await page.screenshot({ path: testInfo.outputPath(`semantic-${locale}-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: english ? "Filename" : "文件名", exact: true }).click();
    await expect(page.getByRole("searchbox", { name: literalLabel })).toHaveValue("车");
    await expect(page.getByText(literalSource, { exact: true })).toBeVisible();
    await expect(page.getByText(semanticSource, { exact: true })).toHaveCount(0);
    expect(requests).toHaveLength(2);
    expect(requests[1]!.pathname).toBe("/api/v1/libraries/lib_fixture/assets");
    expect(requests[1]!.searchParams.get("directoryId")).toBe("dir_fixture");
    expect(requests[1]!.searchParams.get("q")).toBe("车");
    await expect(page).toHaveURL(/recursive=1/);
    await expectNoOverflow(page);
    expect(errors).toEqual([]);
    expect(unsupported).toEqual([]);
  });
}

async function expectNoOverflow(page: import("@playwright/test").Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
}
