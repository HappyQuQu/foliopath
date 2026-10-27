// CR-2026-053: production Storybook fixtures, with responsive and accessibility checks.
import { chromium } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { mkdir, writeFile } from 'node:fs/promises';
const output = process.env.FOLIOPATH_UI_EVIDENCE ?? '/tmp/foliopath-managed-ui';
await mkdir(output, { recursive: true });
const browser = await chromium.launch();
const context = await browser.newContext();
const page = await context.newPage();
const results = [];
try {
  for (const locale of ["zh-CN", "en"]) for (const theme of ["light", "dark"]) for (const width of [390, 768, 1280, 1920]) {
    await page.setViewportSize({ width, height: 1000 });
    for (const state of ['models-downloading', 'model-download-failed', 'models-ready']) {
      await page.goto(`http://127.0.0.1:6006/iframe.html?id=features-intelligence-settings--${state}&viewMode=story&globals=locale:${locale};theme:${theme}`);
      await page.getByRole('heading', { name: locale === 'en' ? 'Global AI service' : '全局 AI 服务' }).waitFor();
      if (state === 'models-downloading') await page.getByRole('progressbar', { name: locale === 'en' ? 'AI model download progress' : 'AI 模型下载进度' }).waitFor();
      if (state === 'model-download-failed') await page.getByText(locale === "en" ? /will retry automatically/ : /系统会自动重试/).waitFor();
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
      if (overflow) throw new Error(`Horizontal overflow at ${width}/${state}`);
      await page.waitForTimeout(1200);
      const axe = await new AxeBuilder({ page }).analyze();
      const serious = axe.violations.filter(v => v.impact === 'serious' || v.impact === 'critical');
      if (serious.length) throw new Error(JSON.stringify(serious));
      await page.screenshot({ path: `${output}/${state}-${width}-${locale}-${theme}.png`, fullPage: true });
      results.push({ locale, theme, width, state, overflow, seriousAccessibilityViolations: 0 });
    }
  }
  await writeFile(`${output}/checks.json`, JSON.stringify(results, null, 2));
  console.log(`${results.length} responsive/accessibility checks passed`);
} finally { await browser.close(); }
