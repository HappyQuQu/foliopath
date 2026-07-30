import { expect, test, type Page } from "@playwright/test";
import { fileURLToPath } from "node:url";

const libraryId = "lib_visual";
const directoryId = "dir_kyoto";
const primaryAssetId = "asset_kyoto_01";
const imageFixturePath = fileURLToPath(
  new URL(
    "../../../tests/fixtures/media/viewer-blue-violet.png",
    import.meta.url,
  ),
);

const library = {
  assetCount: 24,
  directoryCount: 8,
  displayPath: "/library/family",
  id: libraryId,
  lastSuccessfulScanAt: "2026-07-28T08:00:00Z",
  latestScanId: "scan_visual",
  name: "Family archive",
  status: "ready",
} as const;

const directory = {
  breadcrumbs: [
    { id: "dir_root", name: "Family archive", relativePath: "" },
    { id: "dir_travel", name: "Travel", relativePath: "Travel" },
    {
      id: directoryId,
      name: "Kyoto",
      relativePath: "Travel/Japan/Kyoto",
    },
  ],
  directAssetCount: 24,
  hasChildren: true,
  id: directoryId,
  libraryId,
  name: "Kyoto",
  parentId: "dir_japan",
  recursiveAssetCount: 38,
  relativePath: "Travel/Japan/Kyoto",
} as const;

const childDirectories = [
  ["dir_kiyomizu", "Kiyomizu-dera", 8],
  ["dir_fushimi", "Fushimi Inari", 7],
  ["dir_kinkaku", "Kinkaku-ji", 6],
  ["dir_arashiyama", "Arashiyama", 5],
  ["dir_gion", "Gion", 4],
  ["dir_nijo", "Nijo Castle", 3],
  ["dir_other", "Other", 5],
].map(([id, name, count]) => ({
  directAssetCount: Number(count),
  hasChildren: false,
  id: String(id),
  libraryId,
  name: String(name),
  parentId: directoryId,
  recursiveAssetCount: Number(count),
  relativePath: `Travel/Japan/Kyoto/${String(name)}`,
}));

const assets = Array.from({ length: 24 }, (_, index) => {
  const suffix = String(index + 1).padStart(2, "0");
  return {
    directoryId,
    durationMs: null,
    height: 960,
    id: `asset_kyoto_${suffix}`,
    kind: "image",
    libraryId,
    libraryName: library.name,
    mimeType: "image/png",
    modifiedAt: `2026-07-${String(28 - (index % 8)).padStart(2, "0")}T${String(
      18 - (index % 9),
    ).padStart(2, "0")}:30:00Z`,
    name: `Kyoto memory ${suffix}.png`,
    playbackStatus: "not_applicable",
    probeStatus: "ready",
    relativePath: `Travel/Japan/Kyoto/Kyoto memory ${suffix}.png`,
    sizeBytes: 4_800_000 + index * 12_000,
    sourceAvailability: "available",
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
    thumbnail: {
      errorCode: null,
      status: "ready",
      url: `/api/v1/assets/asset_kyoto_${suffix}/thumbnail`,
    },
    width: 1440,
  } as const;
});

test.beforeEach(async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== "visual-chromium");
  test.skip(
    process.platform !== "linux",
    "Approved visual baselines are generated and compared on Linux.",
  );

  await page.addInitScript(() => {
    window.localStorage.setItem(
      "foliopath.preferences.v1",
      JSON.stringify({
        locale: "en",
        mediaLayout: "grid",
        previewPinned: false,
        theme: "dark",
      }),
    );
  });
  await mockVisualBackend(page);
});

test("global header keeps the approved desktop composition", async ({
  page,
}) => {
  await page.goto("/settings/general");
  await expect(
    page.getByRole("heading", { name: "General", level: 1 }),
  ).toBeVisible();

  await expect(page.getByRole("banner").first()).toHaveScreenshot(
    "global-header-dark.png",
    screenshotOptions,
  );
});

for (const managementPage of [
  {
    name: "general",
    path: "/settings/general",
    heading: "General",
  },
  {
    name: "libraries",
    path: "/settings/libraries",
    heading: "Libraries",
  },
  {
    name: "storage",
    path: "/settings/storage",
    heading: "Scanning and cache",
  },
  {
    name: "account",
    path: "/settings/account",
    heading: "Account",
  },
] as const) {
  test(`management ${managementPage.name} page keeps its approved hierarchy`, async ({
    page,
  }) => {
    await page.goto(managementPage.path);
    await expect(
      page.getByRole("heading", { name: managementPage.heading, level: 1 }),
    ).toBeVisible();

    await expect(page).toHaveScreenshot(
      `management-${managementPage.name}-dark.png`,
      screenshotOptions,
    );
  });
}

test("browse keeps approved top, bottom, and preview states", async ({
  page,
}) => {
  await page.goto(`/libraries/${libraryId}/browse/${directoryId}`);
  await expect(
    page.getByRole("heading", { name: "Kyoto", level: 1 }),
  ).toBeAttached();
  await expect(page.getByText("Kyoto memory 01.png")).toBeVisible();
  await waitForRenderedImages(page);

  await expect(page).toHaveScreenshot("browse-top-dark.png", screenshotOptions);

  await page.evaluate(() =>
    window.scrollTo(0, document.documentElement.scrollHeight),
  );
  await expect(page).toHaveScreenshot(
    "browse-bottom-dark.png",
    screenshotOptions,
  );

  await page
    .getByRole("button", { name: "Preview Kyoto memory 01.png" })
    .click();
  await expect(
    page.getByRole("complementary", { name: "Preview" }),
  ).toBeVisible();
  await waitForRenderedImages(page);
  await page.waitForTimeout(150);
  await expect(page).toHaveScreenshot(
    "browse-preview-dark.png",
    screenshotOptions,
  );
});

test("search keeps the approved result state", async ({ page }) => {
  await page.goto(`/libraries/${libraryId}/search?q=Kyoto`);
  await expect(
    page.getByRole("heading", { name: "Search media" }),
  ).toBeVisible();
  await expect(page.getByText("Kyoto memory 01.png")).toBeVisible();
  await waitForRenderedImages(page);

  await expect(page).toHaveScreenshot(
    "search-results-dark.png",
    screenshotOptions,
  );
});

test("viewer keeps the approved available-image state", async ({ page }) => {
  await page.goto(
    `/libraries/${libraryId}/media/${primaryAssetId}?from=${encodeURIComponent(
      `/libraries/${libraryId}/browse/${directoryId}`,
    )}`,
  );
  await expect(
    page.getByRole("main", { name: "Kyoto memory 01.png" }),
  ).toBeFocused();
  await waitForRenderedImages(page);

  await expect(page).toHaveScreenshot(
    "viewer-image-dark.png",
    screenshotOptions,
  );
});

test("offline viewer keeps the approved dark desktop composition", async ({
  page,
}) => {
  await page.goto(`/libraries/${libraryId}/media/offline`);
  await expect(
    page.getByRole("heading", { name: "Library is offline" }),
  ).toBeVisible();
  await expect(page.getByRole("main", { name: "offline.png" })).toBeFocused();

  await expect(page).toHaveScreenshot(
    "offline-viewer-dark.png",
    screenshotOptions,
  );
});

const screenshotOptions = {
  animations: "disabled" as const,
  caret: "hide" as const,
  fullPage: false,
};

async function mockVisualBackend(page: Page) {
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;

    if (path === "/health/ready") {
      return fulfillJson(route, { reasonCode: null, status: "ready" });
    }
    if (path === "/api/v1/auth/session") {
      return fulfillJson(route, {
        administrator: {
          displayName: "Visual administrator",
          id: "usr_visual",
          username: "visual-admin",
        },
        csrfToken: "visual-csrf-token-0000000000000000",
        expiresAt: "2030-01-01T00:00:00Z",
      });
    }

    if (path === "/api/v1/settings") {
      return fulfillJson(
        route,
        {
          automaticDiscoveryEnabled: true,
          language: "en",
          scheduledScanIntervalHours: 24,
          thumbnailCacheQuotaBytes: 10 * 1024 ** 3,
          updatedAt: "2026-07-28T08:00:00Z",
        },
        { ETag: '"settings-visual"' },
      );
    }
    if (path === "/api/v1/account") {
      return fulfillJson(
        route,
        {
          displayName: "Visual administrator",
          id: "usr_visual",
          revision: 3,
          updatedAt: "2026-07-28T08:00:00Z",
          username: "visual-admin",
        },
        { ETag: '"account-visual"' },
      );
    }
    if (path === "/api/v1/cache") {
      return fulfillJson(route, {
        availableBytes: 82 * 1024 ** 3,
        cleanup: {
          deletedEntries: 0,
          errorCode: null,
          finishedAt: null,
          initialUsageBytes: 0,
          reclaimedBytes: 0,
          remainingUsageBytes: 0,
          requestedAt: null,
          startedAt: null,
          status: "idle",
        },
        highWatermarkBytes: 9 * 1024 ** 3,
        lowWatermarkBytes: 8 * 1024 ** 3,
        pressure: "normal",
        quotaBytes: 10 * 1024 ** 3,
        safeFreeSpaceBytes: 5 * 1024 ** 3,
        usageBytes: 3 * 1024 ** 3,
      });
    }
    if (path === "/api/v1/libraries") {
      return fulfillJson(route, { items: [library], nextCursor: null });
    }
    if (path === `/api/v1/libraries/${libraryId}`) {
      return fulfillJson(route, library, { ETag: '"library-visual"' });
    }
    if (path === `/api/v1/directories/${directoryId}`) {
      return fulfillJson(route, directory);
    }
    if (path === `/api/v1/libraries/${libraryId}/directories`) {
      const parentId = url.searchParams.get("parentId");
      return fulfillJson(route, {
        items: parentId === directoryId ? childDirectories : [],
        nextCursor: null,
      });
    }
    if (
      path === `/api/v1/libraries/${libraryId}/assets` ||
      path === "/api/v1/assets"
    ) {
      return fulfillJson(route, { items: assets, nextCursor: null });
    }
    if (path.endsWith("/thumbnail")) {
      return route.fulfill({
        contentType: "image/png",
        path: imageFixturePath,
      });
    }
    if (path.endsWith("/content")) {
      return route.fulfill({
        contentType: "image/png",
        path: imageFixturePath,
      });
    }
    if (path === "/api/v1/assets/offline") {
      return fulfillJson(route, {
        ...assets[0],
        height: 1,
        id: "offline",
        name: "offline.png",
        relativePath: "Travel/Japan/Kyoto/offline.png",
        sizeBytes: 68,
        sourceAvailability: "offline",
        thumbnail: { errorCode: null, status: "unavailable", url: null },
        width: 1,
      });
    }
    if (path.startsWith("/api/v1/assets/")) {
      const assetId = path.split("/")[4];
      const asset = assets.find((candidate) => candidate.id === assetId);
      if (asset) return fulfillJson(route, asset);
    }

    if (path.startsWith("/api/")) {
      return fulfillJson(
        route,
        {
          error: {
            code: "visual_fixture_missing",
            message: `No fixture for ${path}`,
          },
        },
        {},
        404,
      );
    }
    return route.continue();
  });
}

function fulfillJson(
  route: Parameters<Parameters<Page["route"]>[1]>[0],
  body: unknown,
  headers: Record<string, string> = {},
  status = 200,
) {
  return route.fulfill({
    body: JSON.stringify(body),
    contentType: "application/json",
    headers,
    status,
  });
}

async function waitForRenderedImages(page: Page) {
  await page.waitForFunction(() =>
    Array.from(document.images).every(
      (image) => image.complete && image.naturalWidth > 0,
    ),
  );
  await page.evaluate(async () => {
    await Promise.all(
      Array.from(document.images, (image) =>
        image.decode().catch(() => undefined),
      ),
    );
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
    );
  });
}
