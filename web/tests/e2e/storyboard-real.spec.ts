import { expect, test } from "@playwright/test";

const enabled = process.env.FOLIOPATH_STORYBOARD_REAL_E2E === "1";
const libraryId = process.env.FOLIOPATH_STORYBOARD_LIBRARY_ID ?? "";

test("real scanned video reaches browse and search storyboard hover", async ({
  page,
}) => {
  test.skip(!enabled, "Run through tests/release/storyboard_vertical_smoke.sh.");
  expect(libraryId).not.toBe("");

  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Log in to FolioPath" }),
  ).toBeVisible();
  await page.getByLabel("Username *").fill("StoryboardAdmin");
  await page.getByLabel("Password *").fill("correct horse battery staple");
  await page.getByRole("button", { name: "Log in" }).click();
  await expect(page).toHaveURL(/\/settings\/libraries$/);

  const storyboardRequests: string[] = [];
  page.on("request", (request) => {
    if (request.url().includes("variant=storyboard")) {
      storyboardRequests.push(request.url());
    }
  });

  await page.goto(`/libraries/${libraryId}/browse`);
  const browseCard = page.getByRole("article", { name: "clip.mp4 · Video" });
  await expect(browseCard).toBeVisible();
  await expect(
    browseCard.locator('img[src*="/thumbnail"]:not([src*="storyboard"])'),
  ).toBeVisible();
  expect(storyboardRequests).toHaveLength(0);

  await browseCard.hover();
  await page.waitForTimeout(200);
  expect(storyboardRequests).toHaveLength(0);
  await expect(browseCard).toHaveAttribute("data-storyboard-playing", "true");
  expect(storyboardRequests.length).toBeGreaterThan(0);
  await expect(
    browseCard.locator('img[src*="variant=storyboard"]'),
  ).toBeVisible();

  await page.locator("header").first().hover();
  await expect(browseCard).not.toHaveAttribute("data-storyboard-playing");
  await expect(
    browseCard.locator('img[src*="variant=storyboard"]'),
  ).toHaveCount(0);

  const previewTrigger = page.getByRole("button", {
    name: "Preview clip.mp4",
  });
  await previewTrigger.click();
  await expect(
    page.getByRole("complementary", { name: "Preview: clip.mp4" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close preview" }).click();
  await expect(previewTrigger).toBeFocused();

  storyboardRequests.length = 0;
  await page.goto(`/libraries/${libraryId}/search?q=clip`);
  const searchCard = page.getByRole("article", { name: "clip.mp4 · Video" });
  await expect(searchCard).toBeVisible();
  expect(storyboardRequests).toHaveLength(0);
  await searchCard.hover();
  await expect(searchCard).toHaveAttribute("data-storyboard-playing", "true");
  expect(storyboardRequests.length).toBeGreaterThan(0);
  await page.locator("header").first().hover();
  await expect(searchCard).not.toHaveAttribute("data-storyboard-playing");
});
