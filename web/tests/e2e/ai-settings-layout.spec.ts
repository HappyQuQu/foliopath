import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

for (const locale of ["zh-CN", "en"]) for (const width of [320, 390, 768, 1024, 1280, 1440, 1920]) {
  test(`${locale} four capability cards reflow at ${width}px`, async ({ page }, info) => {
    test.skip(process.env.FOLIOPATH_AI_SETTINGS_LAYOUT_STORY !== "1", "Isolated production-component Storybook only.");
    await page.setViewportSize({ width, height: 1000 });
    const english = locale === "en";
    const titles = english ? ["Smart search", "People analysis", "Automatic tags", "Text recognition"] : ["智能搜索", "人物分析", "自动标签", "文字识别"];
    await page.goto(`/iframe.html?id=features-intelligence-settings--four-capabilities&viewMode=story&globals=${encodeURIComponent(`locale:${locale};theme:${english ? "dark" : "light"}`)}`);
    for (const name of titles) await expect(page.getByRole("region", { name, exact: true })).toBeVisible();
    const check = async (columns: number) => {
      const boxes = await Promise.all(titles.map(name => page.getByRole("region", { name, exact: true }).boundingBox()));
      const cards = boxes.map(box => { expect(box).not.toBeNull(); return box!; });
      expect(await page.getByRole("region", { name: titles[0], exact: true }).evaluate(el => el.parentElement?.querySelectorAll(":scope > section").length)).toBe(4);
      for (const card of cards) expect(Math.abs(card.width - cards[0]!.width)).toBeLessThan(1);
      if (columns === 2) {
        expect(Math.abs(cards[0]!.y - cards[1]!.y)).toBeLessThan(1);
        expect(Math.abs(cards[2]!.y - cards[3]!.y)).toBeLessThan(1);
        expect(cards[2]!.y).toBeGreaterThan(cards[0]!.y);
        expect(Math.abs(cards[0]!.x - cards[2]!.x)).toBeLessThan(1);
        expect(Math.abs(cards[0]!.height - cards[1]!.height)).toBeLessThan(1);
        expect(Math.abs(cards[2]!.height - cards[3]!.height)).toBeLessThan(1);
      } else {
        for (let i = 1; i < cards.length; i++) {
          expect(cards[i]!.y).toBeGreaterThan(cards[i - 1]!.y);
          expect(Math.abs(cards[i]!.x - cards[0]!.x)).toBeLessThan(1);
        }
      }
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      expect(await page.getByRole("main").evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
    };
    await check(width >= 1280 ? 2 : 1);
    if (width === 1280) {
      const resizer = page.getByRole("separator", { name: english ? "Resize directory sidebar" : "调整目录栏宽度" });
      await resizer.focus(); await page.keyboard.press("Home");
      await expect(resizer).toHaveAttribute("aria-valuenow", "192");
      await check(2);
      await page.keyboard.press("End");
      await expect(resizer).toHaveAttribute("aria-valuenow", "480");
      await check(1);
    }
    const violations = (await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze()).violations;
    expect(violations.map(v => ({ id: v.id, targets: v.nodes.map(n => n.target) }))).toEqual([]);
    await page.screenshot({ path: info.outputPath(`ai-features-${locale}-${width}.png`), fullPage: true });
  });
}
