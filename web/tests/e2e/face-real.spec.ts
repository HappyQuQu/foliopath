import { expect, test } from "@playwright/test";

test("real Docker face results support manual people workflows", async ({ page }) => {
  test.skip(process.env.FOLIOPATH_FACE_REAL_E2E !== "1", "Run face-business-probe.mjs.");
  const libraryId = process.env.FOLIOPATH_FACE_LIBRARY_ID!;
  const assetId = process.env.FOLIOPATH_FACE_ASSET_ID!;
  await page.goto("/");
  await page.getByLabel("Username *").fill("face-startup-test");
  await page.getByLabel("Password *").fill(process.env.FOLIOPATH_FACE_PASSWORD!);
  await page.getByRole("button", { name: "Log in", exact: true }).click();
  await expect(page).not.toHaveURL(/login/);
  if (process.env.FOLIOPATH_FACE_OFFLINE_E2E === "1") {
    await page.goto(`/intelligence?view=clusters&libraryId=${libraryId}`);
    await expect(page.getByText("This library is unavailable. Existing content is kept. Reconnect and try again.")).toBeVisible();
    await expect(page.getByText(/Face analysis is not ready/)).toHaveCount(0);
    return;
  }
  await page.goto("/intelligence?view=people");
  await page.getByRole("button", { name: "Next page", exact: true }).click();
  await expect(page).toHaveURL(/cursor=/);
  await page.getByRole("button", { name: "First page", exact: true }).click();
  await expect(page).not.toHaveURL(/cursor=/);
  await page.getByLabel("Administrator-supplied name", { exact: true }).fill("Browser synthetic");
  await page.getByRole("button", { name: "Create person", exact: true }).click();
  await expect(page).toHaveURL(/person=person_/);
  const personURL = page.url();
  const personId = new URL(personURL).searchParams.get("person")!;
  await page.getByLabel("Administrator-supplied name", { exact: true }).fill("Browser renamed");
  await page.getByRole("button", { name: "Rename", exact: true }).click();
  await expect(page.getByRole("heading", { name: /Browser renamed/ })).toBeVisible();

  await page.goto(`/intelligence?view=clusters&libraryId=${libraryId}`);
  await expect(page.getByText(/Each action saves one face, never an entire batch/)).toBeVisible();
  await page.getByRole("button", { name: /Person to confirm.*2 faces/ }).first().click();
  await expect(page.getByRole("button", { name: "Create person", exact: true })).toBeEnabled();
  await expect(page.getByRole("radio")).toHaveCount(2);
  const markedFaces = page.getByRole("button", { name: /^Photo \d+ · Marker \d+$/ });
  await expect(markedFaces).toHaveCount(2);
  await expect(markedFaces.first()).toBeVisible();
  await page.screenshot({ path: test.info().outputPath("face-cluster.png"), fullPage: true });

  await page.goto(`/intelligence?view=faces&libraryId=${libraryId}&asset=${assetId}`);
  const faceButton = page.getByRole("button", { name: /Face .*state:/ }).last();
  await faceButton.focus();
  await expect(faceButton).toBeFocused();
  await page.keyboard.press("Enter");
  await page.getByRole("combobox", { name: "Move to person" }).selectOption(personId);
  const submitted = page.waitForResponse(response => response.url().endsWith("/ai/face-reviews") && response.request().method() === "POST");
  await page.getByRole("button", { name: "Move to person", exact: true }).click();
  const response = await submitted;
  expect(response.status()).toBe(200);
  expect(response.request().postDataJSON().action).toBe("assign_face");
  await page.goto(personURL);
  await expect(page.getByText("1 confirmed faces · 1 assets")).toBeVisible();
  await page.getByRole("button", { name: "Remove from person" }).click();
  await page.getByRole("dialog", { name: "Remove this face from the person?", exact: true }).getByRole("button", { name: "Confirm assignment change", exact: true }).click();
  await expect(page.getByText("0 confirmed faces · 0 assets")).toBeVisible();
  await page.reload();
  await expect(page.getByRole("heading", { name: /Browser renamed/ })).toBeVisible();
  await page.goto(`/intelligence?view=clusters&libraryId=${libraryId}`);
  await page.getByRole("button", { name: /Person to confirm.*2 faces/ }).first().click();
  await page.getByRole("combobox", { name: "Target person" }).selectOption(personId);
  for (const index of [0, 1]) {
    await page.getByRole("radio").nth(index).check();
    await page.getByRole("button", { name: "Confirm person assignment", exact: true }).click();
    await page.getByRole("dialog", { name: "Confirm this face belongs to the selected person?", exact: true }).getByRole("button", { name: "Confirm save", exact: true }).click();
    await expect(page.getByRole("radio").nth(index)).toBeDisabled();
  }
  await page.goto(personURL);
  await expect(page.getByText("2 confirmed faces · 2 assets")).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: test.info().outputPath("face-person-mobile.png"), fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.goto(`/settings/ai?view=faces&libraryId=${libraryId}`);
  const libraryCard = page.getByRole("article").filter({ has: page.getByRole("heading", { name: "Synthetic two", exact: true }) });
  await libraryCard.getByText("Index maintenance", { exact: true }).click();
  for (const name of ["Clear rebuildable face data", "Clear people app state"]) {
    const trigger = libraryCard.getByRole("button", { name, exact: true });
    await trigger.click();
    const dialog = page.getByRole("dialog", { name, exact: true });
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(trigger).toBeFocused();
  }
});
