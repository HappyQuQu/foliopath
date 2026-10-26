import AxeBuilder from "@axe-core/playwright";
import { readFileSync } from "node:fs";

import { expect, test, type Page, type Route } from "@playwright/test";

const videoBytes = readFileSync(
  new URL("../../../tests/fixtures/media/range-video.mp4", import.meta.url),
);
const imageBytes = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
  "base64",
);
const libraryId = "lib_matrix";

test("desktop keyboard, focus, and degraded-state matrix", async ({
  page,
}, testInfo) => {
  const chromiumBased = [
    "chromium",
    "chrome-stable",
    "chrome-forced-colors",
  ].includes(testInfo.project.name);
  test.skip(
    ![
      "chromium",
      "chrome-stable",
      "chrome-forced-colors",
      "firefox",
      "webkit",
    ].includes(testInfo.project.name),
  );
  const rangeRequests: string[] = [];
  await mockViewerBackend(page, rangeRequests);
  let codecRequests = 0;
  await page.route("**/api/v1/assets/video_codec/content", async (route) => {
    codecRequests += 1;
    await route.fulfill({
      contentType: "video/x-matroska",
      body: Buffer.from("not-a-playable-video"),
    });
  });

  await page.goto(
    `/libraries/${libraryId}/media/image_ready?from=%2Fsearch%3Fq%3Dmatrix`,
  );
  await expect(page.getByRole("img", { name: "ready.png" })).toBeVisible();
  await expect(page.getByRole("main", { name: "ready.png" })).toBeFocused();
  await expectNoPageOverflow(page);

  // Directly injected React Router history state is not retained by Firefox
  // across reload. Keep that internal-state navigation assertion in Chromium;
  // every engine still verifies the public keyboard and recovery behavior.
  if (chromiumBased) {
    await page.evaluate(() => {
      window.history.replaceState(
        {
          ...window.history.state,
          usr: {
            returnTo: "/search?q=matrix",
            sequence: [
              { id: "image_ready", libraryId: "lib_matrix" },
              { id: "image_next", libraryId: "lib_matrix" },
            ],
          },
        },
        "",
      );
    });
    await page.reload();
    await expect(page.getByRole("main", { name: "ready.png" })).toBeFocused();
  }
  await expect(
    page.getByRole("complementary", { name: "Basic information" }),
  ).toHaveCount(0);
  await page.keyboard.press("i");
  await expect(
    page.getByRole("complementary", { name: "Basic information" }),
  ).toBeVisible();
  await page.keyboard.press("I");
  await expect(
    page.getByRole("complementary", { name: "Basic information" }),
  ).toHaveCount(0);
  if (chromiumBased) {
    await page.keyboard.press("ArrowRight");
    await expect(page).toHaveURL(
      new RegExp(`/libraries/${libraryId}/media/image_next`),
    );
    await expect(page.getByRole("img", { name: "next.png" })).toBeVisible();
  }
  await page.keyboard.press("Escape");
  await expect(page).toHaveURL("/search?q=matrix&scope=all");

  // Unbranded Playwright Chromium may omit proprietary codecs. Check the actual
  // runner capability instead of assuming every Chromium build decodes H.264/AAC.
  const supportsFixture = await page.evaluate(() => {
    const media = document.createElement("video");
    return Boolean(media.canPlayType('video/mp4; codecs="avc1.42E01E"') && media.canPlayType('audio/mp4; codecs="mp4a.40.2"'));
  });
  if (chromiumBased && supportsFixture) {
    await page.goto(`/libraries/${libraryId}/media/video_range`);
    const video = page
      .getByRole("main", { name: "range-video.mp4" })
      .locator("video");
    await expect(video).toHaveAttribute("controls", "");
    await expect(video).toHaveAttribute(
      "poster",
      "/api/v1/assets/video_range/thumbnail",
    );
    await expect
      .poll(() =>
        video.evaluate((element) => (element as HTMLVideoElement).readyState),
      )
      .toBeGreaterThan(0);
    await expect.poll(() => rangeRequests.length).toBeGreaterThan(0);
    expect(rangeRequests.some((value) => value.startsWith("bytes="))).toBe(
      true,
    );
  } else if (chromiumBased) {
    testInfo.annotations.push({ type: "codec-unavailable", description: "This runner lacks H.264/AAC; playback/206 decoding is not validated here. Product fallback is asserted." });
    await page.goto(`/libraries/${libraryId}/media/video_range`);
    await expect(page.getByRole("heading", { name: "This video could not be played." })).toBeVisible();
  }

  await page.goto(`/libraries/${libraryId}/media/video_codec`);
  await expect.poll(() => codecRequests).toBeGreaterThan(0);
  await expect(
    page.getByRole("heading", {
      name: "This video could not be played.",
    }),
  ).toBeVisible();
  await expect(page.locator('video[aria-label="video-codec.mkv"]')).toHaveCount(
    0,
  );
  await expect(page.getByRole("button", { name: "Close" })).toBeVisible();

  await page.goto(`/libraries/${libraryId}/media/offline`);
  await expect(
    page.getByRole("heading", { name: "Library is offline" }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Check again" })).toBeVisible();
  await expectNoSeriousAxeViolations(page);

  await page.goto(`/libraries/${libraryId}/media/deleted`);
  await expect(
    page.getByRole("heading", { name: "Media removed from the index" }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Check again" })).toHaveCount(
    0,
  );
  await expectNoPageOverflow(page);
});

test("mobile touch controls keep the viewer and recovery action reachable", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "mobile-chromium");
  await mockViewerBackend(page, []);

  await page.goto(`/libraries/${libraryId}/media/image_ready`);
  await expect(page.getByRole("img", { name: "ready.png" })).toBeVisible();
  await expect(
    page.getByRole("complementary", { name: "Basic information" }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Show basic information" }).tap();
  await expect(
    page.getByRole("complementary", { name: "Basic information" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Show basic information" }).tap();
  await expect(
    page.getByRole("complementary", { name: "Basic information" }),
  ).toHaveCount(0);
  await expectNoPageOverflow(page);

  await page.goto(`/libraries/${libraryId}/media/offline`);
  await expect(
    page.getByRole("heading", { name: "Library is offline" }),
  ).toBeVisible();
  const retry = page.getByRole("button", { name: "Check again" });
  await expect(retry).toBeVisible();
  await retry.tap();
  await expect(retry).toBeVisible();
  await expect(page.getByRole("button", { name: "Close" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Previous item" }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Next item" })).toBeVisible();
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);
});

test("200 percent equivalent desktop reflow keeps browse and viewer controls reachable", async ({
  page,
}, testInfo) => {
  test.skip(
    ![
      "chromium",
      "chrome-stable",
      "chrome-forced-colors",
      "firefox",
      "webkit",
    ].includes(testInfo.project.name),
  );
  // At 200% browser zoom a 1280×800 desktop exposes an effective 640×400 CSS
  // viewport. This deterministic proxy guards product reflow in every desktop
  // engine; the release Gate still requires a physical-browser zoom review.
  await page.setViewportSize({ width: 640, height: 400 });
  await mockStoryboardBrowseBackend(page, () => {});

  await page.goto(`/libraries/${libraryId}/browse`);
  const card = page.getByRole("article", {
    name: "storyboard-video.mp4 · Video",
  });
  await expect(card).toBeVisible();
  await expectNoPageOverflow(page);

  const preview = card.getByRole("button", {
    name: "Preview storyboard-video.mp4",
  });
  await preview.focus();
  await expect(preview).toBeFocused();

  await page.goto(`/libraries/${libraryId}/media/image_ready`);
  await expect(page.getByRole("main", { name: "ready.png" })).toBeFocused();
  await expect(page.getByRole("button", { name: "Close" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Zoom in" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Zoom out" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Show basic information" }),
  ).toBeVisible();
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);
});

test("storyboard hover is lazy, bounded, and input-mode aware", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name === "visual-chromium",
    "The reduced-motion visual project owns a separate screenshot baseline.",
  );
  let storyboardRequests = 0;
  await mockStoryboardBrowseBackend(page, () => {
    storyboardRequests += 1;
  });

  await page.goto(`/libraries/${libraryId}/browse`);
  const card = page.getByRole("article", {
    name: "storyboard-video.mp4 · Video",
  });
  await expect(card).toBeVisible();
  expect(storyboardRequests).toBe(0);

  if (testInfo.project.name === "mobile-chromium") {
    await card.tap();
    await page.waitForTimeout(400);
    expect(storyboardRequests).toBe(0);
    await expect(card).not.toHaveAttribute("data-storyboard-playing");
    return;
  }

  await card.hover();
  await expect.poll(() => storyboardRequests).toBeGreaterThan(0);
  await expect(card).toHaveAttribute("data-storyboard-playing", "true");
  expect(storyboardRequests).toBeLessThanOrEqual(2);
  const requestsAfterDecode = storyboardRequests;
  const sprite = card.locator('[src*="variant=storyboard"]');
  await expect(sprite).toBeVisible();
  await expect
    .poll(() => sprite.evaluate((element) => element.getAttribute("style")))
    .toContain("--storyboard-x:");

  await page.getByRole("banner").hover();
  await expect(card).not.toHaveAttribute("data-storyboard-playing");
  await expect(sprite).toHaveCount(0);
  expect(storyboardRequests).toBe(requestsAfterDecode);

  storyboardRequests = 0;
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();
  const reducedCard = page.getByRole("article", {
    name: "storyboard-video.mp4 · Video",
  });
  await reducedCard.hover();
  await page.waitForTimeout(400);
  expect(storyboardRequests).toBe(0);
  await expect(reducedCard).not.toHaveAttribute("data-storyboard-playing");
});

for (const locale of ["zh-CN", "en"]) {
  for (const theme of ["light", "dark"]) {
    for (const width of [390, 768, 1280, 1920]) {
      test(`seamless masonry ${locale}/${theme} has no gaps at ${width}px and remembers the layout`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 });
        await mockStoryboardBrowseBackend(page, () => undefined);
        // Synthetic catalog fixtures; production components and generated adapters stay in use.
        const assets = Array.from({ length: 120 }, (_, index) => ({
          ...assetFixture("image_ready"),
          id: `photo_${index}`,
          name: `photo-${index}.png`,
          relativePath: `synthetic-set/photo-${index}.png`,
          height: index % 3 === 0 ? 1200 : 800,
          width: index % 3 === 0 ? 800 : 1200,
          thumbnail: { errorCode: null, status: "ready", url: `/api/v1/assets/photo_${index}/thumbnail` },
        }));
        await page.route(new RegExp(`/api/v1/libraries/${libraryId}/assets`), (route) =>
          route.fulfill({ json: { counts: { all: 120, images: 120, videos: 0 }, items: assets, nextCursor: null } }),
        );
        await page.route(/\/api\/v1\/assets\/photo_\d+\/thumbnail/, (route) =>
          route.fulfill({ contentType: "image/png", body: readFileSync(new URL("../../../tests/fixtures/media/viewer-blue-violet.png", import.meta.url)) }),
        );
        await page.addInitScript(({ dark, chinese }) => {
          if (localStorage.getItem("foliopath.preferences.v1")) return;
          localStorage.setItem("foliopath.preferences.v1", JSON.stringify({
            locale: chinese ? "zh-CN" : "en", theme: dark ? "dark" : "light",
          }));
        }, { dark: theme === "dark", chinese: locale === "zh-CN" });
        await page.goto(`/libraries/${libraryId}/browse?recursive=1`);
        const name = locale === "zh-CN" ? "无缝瀑布流" : "Seamless masonry";
        await page.getByRole("button", { name, exact: true }).click();
        const collection = page.locator('ul[data-layout="seamless"]');
        await expect(collection.locator("article").first()).toBeVisible();
        await expect(collection.locator("strong")).toHaveCount(0);
        expect(await collection.locator("article").count()).toBeLessThan(100);
        const assertGeometry = async () => {
          await expect.poll(() => collection.evaluate((list) => {
            const cards = Array.from(list.querySelectorAll("li")).map((item) => {
              const article = item.querySelector("article")!;
              const image = article.querySelector("img")!;
              return { card: article.getBoundingClientRect(), image: image.getBoundingClientRect(), radius: getComputedStyle(article).borderRadius };
            });
            const errors: string[] = [];
            for (const { card, image, radius } of cards) {
              if (radius !== "0px" || Math.abs(card.height - image.height) > 0.08) errors.push("card chrome");
            }
            const lanes = new Map<number, DOMRect[]>();
            for (const { card } of cards) {
              const key = Math.round(card.x * 10);
              lanes.set(key, [...(lanes.get(key) ?? []), card]);
            }
            const columns = [...lanes.values()].sort((a, b) => a[0]!.x - b[0]!.x);
            for (let column = 0; column < columns.length; column += 1) {
              const lane = columns[column]!.sort((a, b) => a.y - b.y);
              for (let row = 1; row < lane.length; row += 1) {
                if (Math.abs(lane[row]!.top - lane[row - 1]!.bottom) > 0.08) errors.push("vertical gap");
              }
              if (column > 0 && Math.abs(lane[0]!.left - columns[column - 1]![0]!.right) > 0.08) errors.push("horizontal gap");
            }
            return errors;
          })).toEqual([]);
        };
        await assertGeometry();
        const source = collection.locator("a").first();
        await expect(source).toHaveText("");
        // Hover/focus reveals only an icon, never a directory-name overlay.
        await source.focus();
        await expect(source).toBeFocused();
        await expect(source).toHaveCSS("opacity", "1");
        const slider = page.getByRole("slider", { name: locale === "zh-CN" ? "缩略图大小" : "Thumbnail size" });
        await slider.focus();
        await slider.press("End");
        await expect(slider).toHaveValue("480");
        await assertGeometry();
        const largeWidth = await collection.locator("li").first().evaluate((item) => item.getBoundingClientRect().width);
        await slider.press("Home");
        await expect(slider).toHaveValue("120");
        await assertGeometry();
        const smallWidth = await collection.locator("li").first().evaluate((item) => item.getBoundingClientRect().width);
        expect(largeWidth).toBeGreaterThan(smallWidth);
        await slider.press("ArrowRight");
        await expect(slider).toHaveValue("130");
        await assertGeometry();
        await collection.locator('[data-media-id="photo_0"]').focus();
        await expect(collection.locator('[data-media-id="photo_0"]')).toBeFocused();
        await expectNoPageOverflow(page);
        await page.mouse.wheel(0, 600);
        await assertGeometry();
        await page.reload();
        await expect(page.getByRole("button", { name, exact: true })).toHaveAttribute("aria-pressed", "true");
        await expect(collection.locator("article").first()).toBeVisible();
        await assertGeometry();
        await expect(slider).toHaveValue("130");
        await page.screenshot({ path: test.info().outputPath(`seamless-${width}.png`) });
        await page.getByRole("button", { name: locale === "zh-CN" ? "自适应网格" : "Adaptive grid", exact: true }).click();
        await expect(page.locator('ul[data-layout="grid"] strong').first()).toBeVisible();
      });
    }
  }
}

async function mockViewerBackend(page: Page, rangeRequests: string[]) {
  await page.route("**/health/ready", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ status: "ready" }),
    });
  });
  await page.route("**/api/v1/auth/session", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        administrator: {
          displayName: "Media matrix administrator",
          id: "usr_matrix",
          username: "matrix-admin",
        },
        csrfToken: "matrix-csrf-token-0000000000000000",
        expiresAt: "2030-01-01T00:00:00Z",
      }),
    });
  });
  await page.route(/\/api\/v1\/assets\/[^/]+$/, async (route) => {
    const assetId =
      new URL(route.request().url()).pathname.split("/").at(-1) ?? "";
    if (assetId === "deleted") {
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({
          error: {
            code: "asset_not_found",
            message: "Asset not found.",
            requestId: "req_matrix_deleted",
          },
        }),
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(assetFixture(assetId)),
    });
  });
  await page.route("**/api/v1/assets/video_range/thumbnail", async (route) => {
    await route.fulfill({ contentType: "image/png", body: imageBytes });
  });
  await page.route("**/api/v1/assets/video_range/content", async (route) => {
    await fulfillRange(route, rangeRequests);
  });
  await page.route(
    /\/api\/v1\/assets\/image_(?:ready|next)\/content$/,
    async (route) => {
      await route.fulfill({ contentType: "image/png", body: imageBytes });
    },
  );
}

function assetFixture(assetId: string) {
  const video = assetId === "video_range" || assetId === "video_codec";
  const offline = assetId === "offline";
  return {
    directoryId: "dir_matrix",
    durationMs: video ? 1_000 : null,
    height: video ? 180 : 1,
    id: assetId,
    kind: video ? "video" : "image",
    libraryId,
    libraryName: "Media matrix",
    mimeType:
      assetId === "video_codec"
        ? "video/x-matroska"
        : video
          ? "video/mp4"
          : "image/png",
    modifiedAt: "2026-07-28T00:00:00Z",
    name:
      assetId === "image_ready"
        ? "ready.png"
        : assetId === "image_next"
          ? "next.png"
          : assetId === "video_range"
            ? "range-video.mp4"
            : assetId === "video_codec"
              ? "video-codec.mkv"
              : "offline.png",
    playbackStatus:
      assetId === "video_codec"
        ? "unsupported_codec"
        : video
          ? "playable"
          : "not_applicable",
    probeStatus: "ready",
    relativePath: `matrix/${assetId}`,
    sizeBytes: videoBytes.byteLength,
    sourceAvailability: offline ? "offline" : "available",
    storyboard: {
      cellHeight: null,
      cellWidth: null,
      columns: null,
      errorCode: null,
      frameCount: null,
      rows: null,
      status: "not_applicable",
      url: null,
    },
    thumbnail:
      assetId === "video_range"
        ? {
            errorCode: null,
            status: "ready",
            url: "/api/v1/assets/video_range/thumbnail",
          }
        : { errorCode: null, status: "unavailable", url: null },
    width: video ? 320 : 1,
  };
}

async function mockStoryboardBrowseBackend(
  page: Page,
  onStoryboardRequest: () => void,
) {
  await mockViewerBackend(page, []);
  const library = {
    assetCount: 1,
    directoryCount: 0,
    displayPath: "/library/matrix",
    id: libraryId,
    lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
    latestScanId: "scan_matrix",
    name: "Media matrix",
    status: "ready",
  };
  await page.route("**/api/v1/libraries?*", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ items: [library], nextCursor: null }),
    });
  });
  await page.route(
    new RegExp(`/api/v1/libraries/${libraryId}$`),
    async (route) => {
      await route.fulfill({
        contentType: "application/json",
        headers: { ETag: '"library-matrix"' },
        body: JSON.stringify(library),
      });
    },
  );
  await page.route(
    new RegExp(`/api/v1/libraries/${libraryId}/directories`),
    async (route) => {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ items: [], nextCursor: null }),
      });
    },
  );
  await page.route(
    new RegExp(`/api/v1/libraries/${libraryId}/assets`),
    async (route) => {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          items: [
            {
              directoryId: "dir_matrix",
              durationMs: 60_000,
              height: 1080,
              id: "video_storyboard",
              kind: "video",
              libraryId,
              libraryName: "Media matrix",
              mimeType: "video/mp4",
              modifiedAt: "2026-07-28T00:00:00Z",
              name: "storyboard-video.mp4",
              playbackStatus: "playable",
              probeStatus: "ready",
              relativePath: "storyboard-video.mp4",
              sizeBytes: videoBytes.byteLength,
              sourceAvailability: "available",
              storyboard: {
                cellHeight: 180,
                cellWidth: 320,
                columns: 5,
                errorCode: null,
                frameCount: 10,
                rows: 2,
                status: "ready",
                url: "/api/v1/assets/video_storyboard/thumbnail?variant=storyboard",
              },
              thumbnail: {
                errorCode: null,
                status: "ready",
                url: "/api/v1/assets/video_storyboard/thumbnail",
              },
              width: 1920,
            },
          ],
          nextCursor: null,
        }),
      });
    },
  );
  await page.route(
    "**/api/v1/assets/video_storyboard/thumbnail",
    async (route) => {
      await route.fulfill({ contentType: "image/png", body: imageBytes });
    },
  );
  await page.route(
    "**/api/v1/assets/video_storyboard/thumbnail?variant=storyboard",
    async (route) => {
      onStoryboardRequest();
      await route.fulfill({
        contentType: "image/png",
        headers: {
          "Cache-Control": "private, max-age=31536000, immutable",
          ETag: '"storyboard-matrix"',
        },
        body: imageBytes,
      });
    },
  );
}

async function fulfillRange(route: Route, rangeRequests: string[]) {
  const range = route.request().headers().range;
  if (!range) {
    await route.fulfill({
      status: 200,
      contentType: "video/mp4",
      headers: {
        "Accept-Ranges": "bytes",
        "Content-Length": String(videoBytes.byteLength),
      },
      body: videoBytes,
    });
    return;
  }

  rangeRequests.push(range);
  const match = /^bytes=(\d+)-(\d*)$/.exec(range);
  if (!match) {
    await route.fulfill({ status: 416 });
    return;
  }
  const start = Number(match[1]);
  const requestedEnd = match[2] ? Number(match[2]) : videoBytes.byteLength - 1;
  const end = Math.min(requestedEnd, videoBytes.byteLength - 1);
  const body = videoBytes.subarray(start, end + 1);
  await route.fulfill({
    status: 206,
    contentType: "video/mp4",
    headers: {
      "Accept-Ranges": "bytes",
      "Content-Length": String(body.byteLength),
      "Content-Range": `bytes ${start}-${end}/${videoBytes.byteLength}`,
    },
    body,
  });
}

async function expectNoPageOverflow(page: Page) {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
  }));
  expect(dimensions.scrollWidth).toBe(dimensions.clientWidth);
}

async function expectNoSeriousAxeViolations(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  const violations = results.violations.filter(
    (violation) =>
      violation.impact === "serious" || violation.impact === "critical",
  );
  expect(violations).toEqual([]);
}


for (const locale of ["zh-CN", "en"] as const) {
  for (const theme of ["light", "dark"] as const) {
    for (const width of [390, 768, 1280, 1920]) {
      test(`viewer retry hover ${locale}/${theme} at ${width}px keeps contrast`, async ({ page, hasTouch }) => {
        await page.setViewportSize({ width, height: 900 });
        await mockViewerBackend(page, []);
        await page.addInitScript(({ locale, theme }) => {
          localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale, theme }));
        }, { locale, theme });
        await page.goto(`/libraries/${libraryId}/media/offline`);
        const retry = page.getByRole("button", { name: locale === "zh-CN" ? "重新检查" : "Check again", exact: true });
        await expect(retry).toBeVisible();
        if (hasTouch) await retry.tap();
        else await retry.click();
        await expect(retry).toBeVisible();
        await retry.hover();
        await expect.poll(() => retry.evaluate((element) => element.matches(":hover"))).toBe(true);
        await expectNoPageOverflow(page);
        await expectNoSeriousAxeViolations(page);
        await retry.focus();
        await expect(retry).toBeFocused();
        if (locale === "zh-CN" && theme === "light") {
          await page.screenshot({ path: test.info().outputPath(`viewer-retry-hover-${width}.png`) });
        }
      });
    }
  }
}
