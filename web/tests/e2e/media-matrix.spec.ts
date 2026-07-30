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
  ].includes(
    testInfo.project.name,
  );
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

  // Browser codec stacks vary. Chromium owns the real MP4/206 assertion, while
  // all supported engines exercise the product-owned fallback states below.
  if (chromiumBased) {
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
    expect(rangeRequests.some((value) => value.startsWith("bytes="))).toBe(true);
  }

  await page.goto(`/libraries/${libraryId}/media/video_codec`);
  await expect(
    page.getByRole("heading", {
      name: "This browser cannot play the video",
    }),
  ).toBeVisible();
  await expect(
    page.locator('video[aria-label="video-codec.mkv"]'),
  ).toHaveCount(0);
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
  await expect(page.getByRole("button", { name: "Check again" })).toHaveCount(0);
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
  await expect(page.getByRole("button", { name: "Previous item" })).toBeVisible();
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

  await page.locator("header").hover();
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
    const assetId = new URL(route.request().url()).pathname.split("/").at(-1) ?? "";
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
  await page.route(/\/api\/v1\/assets\/image_(?:ready|next)\/content$/, async (route) => {
    await route.fulfill({ contentType: "image/png", body: imageBytes });
  });
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
