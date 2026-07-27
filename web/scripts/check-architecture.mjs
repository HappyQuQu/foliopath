import { readFile, readdir } from "node:fs/promises";
import { extname, join, relative } from "node:path";

const webRoot = new URL("../", import.meta.url);
const sourceRoot = new URL("../src/", import.meta.url);
const checkedExtensions = new Set([".css", ".ts", ".tsx"]);
const findings = [];

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const path = join(directory, entry.name);

    if (entry.isDirectory()) {
      files.push(...(await walk(path)));
    } else if (checkedExtensions.has(extname(entry.name))) {
      files.push(path);
    }
  }

  return files;
}

function report(path, message) {
  findings.push(`${relative(webRoot.pathname, path)}: ${message}`);
}

for (const path of await walk(sourceRoot.pathname)) {
  const source = await readFile(path, "utf8");
  const normalizedPath = relative(sourceRoot.pathname, path);
  const isTestOrStory =
    normalizedPath.startsWith("test/") || /\.(stories|test)\.tsx?$/.test(path);

  if (
    normalizedPath.startsWith("components/") &&
    !isTestOrStory &&
    /from\s+["'][^"']*(?:features|routes|app|lib\/api)[^"']*["']/.test(source)
  ) {
    report(path, "shared components cannot import app, routes, features, or lib/api");
  }

  if (
    normalizedPath.startsWith("components/") &&
    !isTestOrStory &&
    /from\s+["']@tanstack\/react-query["']/.test(source)
  ) {
    report(path, "shared components cannot import TanStack Query");
  }

  if (
    !normalizedPath.startsWith("lib/api/") &&
    !isTestOrStory &&
    /\bfetch\s*\(|from\s+["']openapi-fetch["']/.test(source)
  ) {
    report(path, "HTTP access must go through src/lib/api");
  }

  if (
    normalizedPath.endsWith(".css") &&
    !normalizedPath.startsWith("styles/") &&
    /#[\da-f]{3,8}\b|(?:rgb|hsl)a?\(/i.test(source)
  ) {
    report(path, "component styles must use central semantic color tokens");
  }

  if (
    !normalizedPath.startsWith("lib/storage/") &&
    !isTestOrStory &&
    /\b(?:localStorage|sessionStorage)\b/.test(source)
  ) {
    report(path, "browser persistence must go through src/lib/storage");
  }
}

if (findings.length > 0) {
  console.error("Frontend architecture check failed:\n");
  console.error(findings.map((finding) => `- ${finding}`).join("\n"));
  process.exitCode = 1;
} else {
  console.log("Frontend architecture check passed.");
}
