import { chromium, firefox, webkit } from "@playwright/test";
import { execFile } from "node:child_process";
import { spawn } from "node:child_process";
import { writeFile } from "node:fs/promises";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const outputPath = process.env.FOLIOPATH_BROWSER_CAPACITY_OUTPUT;
const enforceBudget = process.env.FOLIOPATH_BROWSER_CAPACITY_ENFORCE === "1";
const port = Number(process.env.FOLIOPATH_BROWSER_CAPACITY_PORT ?? "6007");
const baseURL = `http://127.0.0.1:${port}`;
const storyURL =
  `${baseURL}/iframe.html?id=patterns-mediacollection--capacity-100-k&viewMode=story`;
const browsers = { chromium, firefox, webkit };
const server = spawn(
  process.platform === "win32" ? "node_modules\\.bin\\vite.cmd" : "node_modules/.bin/vite",
  ["preview", "--outDir", "storybook-static", "--host", "127.0.0.1", "--port", String(port)],
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
    fixture: "Patterns/MediaCollection/Capacity100k",
    itemCount: 100_000,
    viewport: { width: 1280, height: 720 },
    budgets: {
      minimumFramesPerSecond: 45,
      maximumP95FrameIntervalMs: 34,
      maximumPeakRssBytes: 1_610_612_736,
      maximumMountedItems: 64,
    },
    results,
  };
  const encoded = `${JSON.stringify(report, null, 2)}\n`;
  process.stdout.write(encoded);
  if (outputPath) {
    await writeFile(outputPath, encoded);
  }

  if (enforceBudget) {
    const failures = results.flatMap((result) => budgetFailures(result));
    if (failures.length > 0) {
      throw new Error(`browser capacity budget failed: ${failures.join("; ")}`);
    }
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
    const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
    await page.goto(storyURL, { waitUntil: "networkidle" });
    const collectionItems = page.locator('li[aria-setsize="100000"]');
    await collectionItems.first().waitFor();

    let finished = false;
    let peakRssBytes = await processTreeRssBytes(browserServer.process().pid);
    const scrollPromise = page
      .evaluate(
        () =>
          new Promise((resolve) => {
            const timestamps = [];
            let startedAt;
            const run = (timestamp) => {
              startedAt ??= timestamp;
              timestamps.push(timestamp);
              window.scrollBy(0, 2_000);
              if (timestamp - startedAt < 5_000) {
                requestAnimationFrame(run);
                return;
              }
              const intervals = timestamps
                .slice(1)
                .map((value, index) => value - timestamps[index]);
              intervals.sort((left, right) => left - right);
              const percentileIndex = Math.max(
                0,
                Math.ceil(intervals.length * 0.95) - 1,
              );
              resolve({
                durationMs: timestamps.at(-1) - timestamps[0],
                frames: intervals.length,
                p95FrameIntervalMs: intervals[percentileIndex] ?? 0,
                longFrames: intervals.filter((interval) => interval > 50).length,
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
    const scroll = await scrollPromise;
    const mountedItems = await collectionItems.count();
    const finalPosition = await page.evaluate(() => ({
      scrollY: window.scrollY,
      scrollHeight: document.documentElement.scrollHeight,
    }));
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
      browser: name,
      framesPerSecond: round((scroll.frames * 1_000) / scroll.durationMs),
      p95FrameIntervalMs: round(scroll.p95FrameIntervalMs),
      longFrames: scroll.longFrames,
      mountedItems,
      peakRssBytes,
      usedJsHeapBytes,
      ...finalPosition,
    };
  } finally {
    await browserServer.close();
  }
}

async function processTreeRssBytes(rootPid) {
  if (process.platform === "win32") {
    return 0;
  }
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

function budgetFailures(result) {
  const failures = [];
  if (result.framesPerSecond < 45) {
    failures.push(`${result.browser} FPS ${result.framesPerSecond} < 45`);
  }
  if (result.p95FrameIntervalMs > 34) {
    failures.push(
      `${result.browser} frame P95 ${result.p95FrameIntervalMs} ms > 34 ms`,
    );
  }
  if (result.peakRssBytes > 1_610_612_736) {
    failures.push(`${result.browser} RSS ${result.peakRssBytes} > 1.5 GiB`);
  }
  if (result.mountedItems > 64) {
    failures.push(`${result.browser} mounted ${result.mountedItems} > 64`);
  }
  return failures;
}

async function waitForServer(url) {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
    } catch {
      // The preview server is still starting.
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
