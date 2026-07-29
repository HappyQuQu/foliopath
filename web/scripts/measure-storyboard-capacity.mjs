import { chromium, firefox, webkit } from "@playwright/test";
import { execFile } from "node:child_process";
import { spawn } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const port = Number(process.env.FOLIOPATH_STORYBOARD_CAPACITY_PORT ?? "6008");
const baseURL = `http://127.0.0.1:${port}`;
const storyURL =
  `${baseURL}/iframe.html?id=patterns-mediacollection--storyboard-capacity-100&viewMode=story`;
const browsers = { chromium, firefox, webkit };
const server = spawn(
  process.platform === "win32"
    ? "node_modules\\.bin\\vite.cmd"
    : "node_modules/.bin/vite",
  [
    "preview",
    "--outDir",
    "storybook-static",
    "--host",
    "127.0.0.1",
    "--port",
    String(port),
  ],
  { stdio: ["ignore", "pipe", "pipe"] },
);

try {
  await waitForServer(`${baseURL}/iframe.html`);
  const results = [];
  for (const [name, browserType] of Object.entries(browsers)) {
    results.push(await measureBrowser(name, browserType));
  }
  const report = {
    schemaVersion: 1,
    fixture: "Patterns/MediaCollection/StoryboardCapacity100",
    itemCount: 100,
    viewport: { width: 1280, height: 5_000 },
    budgets: {
      maximumActiveStoryboards: 1,
      maximumMountedItems: 100,
      maximumPeakRssBytes: 1_610_612_736,
      minimumFramesPerSecond: 45,
    },
    results,
  };
  process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
  const failures = results.flatMap(budgetFailures);
  if (failures.length > 0) {
    throw new Error(`storyboard browser capacity failed: ${failures.join("; ")}`);
  }
} finally {
  server.kill("SIGTERM");
}

async function measureBrowser(name, browserType) {
  const browserServer = await browserType.launchServer({
    headless: true,
    args: name === "chromium" ? ["--enable-precise-memory-info"] : [],
  });
  try {
    const browser = await browserType.connect(browserServer.wsEndpoint());
    const page = await browser.newPage({
      viewport: { width: 1280, height: 5_000 },
    });
    await page.goto(storyURL, { waitUntil: "networkidle" });
    const cards = page.getByRole("article");
    await cards.first().waitFor();
    const mountedItems = await cards.count();
    let peakRssBytes = await processTreeRssBytes(browserServer.process().pid);

    await dispatchRapidHover(page);
    await page.waitForTimeout(200);
    const activeBeforeIntent = await activeStoryboardCount(page);
    await page.waitForTimeout(250);
    const activeAfterIntent = await activeStoryboardCount(page);

    let finished = false;
    const framePromise = page
      .evaluate(
        () =>
          new Promise((resolve) => {
            const timestamps = [];
            let startedAt;
            const run = (timestamp) => {
              startedAt ??= timestamp;
              timestamps.push(timestamp);
              if (timestamp - startedAt < 3_000) {
                requestAnimationFrame(run);
                return;
              }
              resolve({
                durationMs: timestamps.at(-1) - timestamps[0],
                frames: timestamps.length - 1,
              });
            };
            requestAnimationFrame(run);
          }),
      )
      .finally(() => {
        finished = true;
      });
    while (!finished) {
      peakRssBytes = Math.max(
        peakRssBytes,
        await processTreeRssBytes(browserServer.process().pid),
      );
      await delay(100);
    }
    const frameSample = await framePromise;

    await page.evaluate(() => {
      document
        .querySelector("article[data-storyboard-playing]")
        ?.dispatchEvent(
          new PointerEvent("pointerout", {
            bubbles: true,
            pointerType: "mouse",
          }),
        );
    });
    await page.waitForTimeout(50);
    const activeAfterLeave = await activeStoryboardCount(page);
    const usedJsHeapBytes =
      name === "chromium"
        ? await page.evaluate(
            () =>
              /** @type {{memory?: {usedJSHeapSize?: number}}} */ (performance)
                .memory?.usedJSHeapSize ?? 0,
          )
        : 0;
    await browser.close();

    return {
      activeAfterIntent,
      activeAfterLeave,
      activeBeforeIntent,
      browser: name,
      framesPerSecond: round(
        (frameSample.frames * 1_000) / frameSample.durationMs,
      ),
      mountedItems,
      peakRssBytes,
      usedJsHeapBytes,
    };
  } finally {
    await browserServer.close();
  }
}

async function dispatchRapidHover(page) {
  await page.evaluate(() => {
    for (const card of document.querySelectorAll("article")) {
      card.dispatchEvent(
        new PointerEvent("pointerover", {
          bubbles: true,
          pointerType: "mouse",
        }),
      );
    }
  });
}

async function activeStoryboardCount(page) {
  return page.locator("article[data-storyboard-playing]").count();
}

function budgetFailures(result) {
  const failures = [];
  if (result.mountedItems !== 100) {
    failures.push(`${result.browser} mounted ${result.mountedItems} != 100`);
  }
  if (result.activeBeforeIntent !== 0) {
    failures.push(
      `${result.browser} active before intent ${result.activeBeforeIntent} != 0`,
    );
  }
  if (result.activeAfterIntent !== 1) {
    failures.push(
      `${result.browser} active after intent ${result.activeAfterIntent} != 1`,
    );
  }
  if (result.activeAfterLeave !== 0) {
    failures.push(
      `${result.browser} active after leave ${result.activeAfterLeave} != 0`,
    );
  }
  if (result.framesPerSecond < 45) {
    failures.push(`${result.browser} FPS ${result.framesPerSecond} < 45`);
  }
  if (result.peakRssBytes > 1_610_612_736) {
    failures.push(`${result.browser} RSS ${result.peakRssBytes} > 1.5 GiB`);
  }
  return failures;
}

async function processTreeRssBytes(rootPid) {
  if (process.platform === "win32") return 0;
  const { stdout } = await execFileAsync("ps", ["-A", "-o", "pid=,ppid=,rss="]);
  const rows = stdout
    .trim()
    .split("\n")
    .map((line) => line.trim().split(/\s+/).map(Number))
    .filter((row) => row.length === 3 && row.every(Number.isFinite));
  const descendants = new Set([rootPid]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const [pid, parentPid] of rows) {
      if (descendants.has(parentPid) && !descendants.has(pid)) {
        descendants.add(pid);
        changed = true;
      }
    }
  }
  return (
    rows
      .filter(([pid]) => descendants.has(pid))
      .reduce((total, [, , rssKiB]) => total + rssKiB, 0) * 1_024
  );
}

async function waitForServer(url) {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // The Storybook preview server is still starting.
    }
    await delay(100);
  }
  throw new Error(`preview server did not start at ${url}`);
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

function round(value) {
  return Math.round(value * 1_000) / 1_000;
}
