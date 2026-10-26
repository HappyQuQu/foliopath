// Isolated real backend + synthetic originals. Run only against the named review fixture.
import AxeBuilder from "@axe-core/playwright";
import { readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";

const fixturePath = process.env.FOLIOPATH_LAYERED_FIXTURE;
const fixture = fixturePath ? JSON.parse(readFileSync(fixturePath, "utf8")) as {
  origin: string; libraryId: string; directoryId: string; assets: { id: string; name: string }[]; tags: { id: string }[];
} : undefined;

for (const locale of ["zh-CN", "en"]) for (const width of [390, 768, 1440, 1920]) {
  test(`${locale} real layered workspace at ${width}px`, async ({ page }, info) => {
    test.setTimeout(120_000);
    test.skip(!fixture, "Requires the isolated synthetic review fixture; never target an existing instance.");
    const data = fixture!;
    expect(data.origin).toBe("http://127.0.0.1:6012");
    const origin = process.env.FOLIOPATH_WEB_E2E_URL ?? data.origin;
    const login = await page.request.post(origin + "/api/v1/auth/login", { headers: { Origin: origin }, data: { username: "review", password: "Synthetic-UI-Review-20261025!" } });
    expect(login.ok()).toBe(true);
    await page.setViewportSize({ width, height: 1000 });
    await page.addInitScript(({ locale }) => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale, theme: locale === "en" ? "dark" : "light", mediaLayout: "seamless", mediaTileSize: 270, previewWidth: 342 })), { locale });
    const errors: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    const routes = [
      ["browse", `/libraries/${data.libraryId}/browse/${data.directoryId}`],
      ["search", `/libraries/${data.libraryId}/search?q=forest`],
      ["favorites", `/favorites?libraryId=${data.libraryId}`],
      ["tags", `/tags/${data.tags[0]!.id}?libraryId=${data.libraryId}&kind=image`],
      ["people", `/intelligence?view=people&libraryId=${data.libraryId}`],
      ["groups", `/intelligence?view=clusters&libraryId=${data.libraryId}`],
      ["libraries", "/settings/libraries"],
      ["new-library", "/settings/libraries/new"],
      ["scans", `/settings/libraries/${data.libraryId}/status`],
      ["ai", "/settings/ai"],
      ["jobs", `/settings/ai?view=tasks&library=${data.libraryId}`],
      ["models", "/settings/ai?view=service"],
      ["general", "/settings/general"],
      ["schedule", "/settings/storage?section=scan"],
      ["cache", "/settings/storage?section=cache"],
      ["account", "/settings/account"],
      ["logs", "/settings/logs"],
      ["about", "/settings/about"],
      ["viewer", `/libraries/${data.libraryId}/media/${data.assets[0]!.id}`],
    ];
    for (const [name, route] of routes) {
      await page.goto(origin + route);
      await expect(page.getByRole("main")).toBeVisible();
      if (name !== "viewer") await expect(page.getByRole("heading", { level: 1 }).first()).toBeAttached();
      if (["browse", "favorites", "tags"].includes(name!)) {
        const image = page.getByRole("button", { name: /^(预览|Preview) / }).first();
        await image.click();
        await expect(page.locator('aside').filter({ has: page.locator('img[src*="/content"]') }).last()).toBeVisible();
        await expect(page.locator('img[src*="/content"]')).toHaveJSProperty("complete", true);
      }
      if (name === "browse" && width <= 480) {
        // A no-overflow check alone misses Chinese labels squeezed into one glyph per line.
        const types = page.getByRole("radiogroup", { name: locale === "en" ? "Media type" : "媒体类型" });
        await expect(types.getByRole("radio")).toHaveCount(3);
        const oneLine = await types.locator("button > span:first-child").evaluateAll(labels => labels.every(label => {
          const line = parseFloat(getComputedStyle(label).lineHeight);
          return label.getBoundingClientRect().height <= line + 1;
        }));
        expect(oneLine, "media type labels must remain on one readable line").toBe(true);
      }
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      await page.evaluate(() => window.scrollTo({ top: 0, behavior: "instant" }));
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), `${name}: horizontal overflow`).toBe(true);
      if (name !== "viewer") {
        const header = await page.getByRole("banner").first().boundingBox();
        const h1 = await page.getByRole("heading", { level: 1 }).first().boundingBox();
        expect(h1!.y, `${name}: header must not cover title`).toBeGreaterThanOrEqual(header!.height - 1);
      }
      const violations = (await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze()).violations;
      expect(violations.map(v => ({ id: v.id, targets: v.nodes.map(n => n.target) })), name).toEqual([]);
      await page.screenshot({ path: info.outputPath(`${name}-${locale}-${width}.png`), fullPage: false });
    }
    if (locale === "zh-CN") {
      for (const name of ["browse", "tags", "general", "ai"]) {
        await page.goto(`http://host.docker.internal:4190/#${name}`);
        await expect(page.getByRole("heading", { level: 1 }).first()).toBeVisible();
        await page.waitForFunction(() => Array.from(document.images).every(image => image.complete));
        await page.screenshot({ path: info.outputPath(`source-${name}-${width}.png`), fullPage: false });
      }
    }
    expect(errors).toEqual([]);
  });
}
