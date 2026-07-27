import { spawnSync } from "node:child_process";
import { readFile, readdir } from "node:fs/promises";

const webRoot = new URL("..", import.meta.url);

async function assertNoRSCMode() {
  const sourceRoot = new URL("src/", webRoot);
  const entries = await readdir(sourceRoot, { recursive: true, withFileTypes: true });
  const sourceFiles = entries.filter(
    (entry) => entry.isFile() && /\.(?:ts|tsx|js|jsx)$/.test(entry.name),
  );

  for (const entry of sourceFiles) {
    const file = new URL(entry.parentPath.replace(`${sourceRoot.pathname}/`, "") + "/", sourceRoot);
    const contents = await readFile(new URL(entry.name, file), "utf8");
    if (/react-router\/rsc|RSCHydratedRouter|createCallServer/.test(contents)) {
      throw new Error(`The React Router RSC audit exception is invalidated by ${entry.name}`);
    }
  }
}

await assertNoRSCMode();

const acceptedAdvisories = new Map([
  [
    "https://github.com/advisories/GHSA-qwww-vcr4-c8h2",
    "React Router RSC server actions are not used by this BrowserRouter SPA.",
  ],
]);

const audit = spawnSync("npm", ["audit", "--json", "--audit-level=high"], {
  cwd: webRoot,
  encoding: "utf8",
});

if (audit.error) throw audit.error;

let report;
try {
  report = JSON.parse(audit.stdout);
} catch {
  process.stderr.write(audit.stdout);
  process.stderr.write(audit.stderr);
  throw new Error("npm audit did not return JSON");
}

const vulnerabilities = report.vulnerabilities ?? {};

function unresolvedAdvisories(name, visited = new Set()) {
  if (visited.has(name)) return [];
  visited.add(name);

  const vulnerability = vulnerabilities[name];
  if (!vulnerability || !["high", "critical"].includes(vulnerability.severity)) return [];

  return vulnerability.via.flatMap((entry) => {
    if (typeof entry === "string") return unresolvedAdvisories(entry, visited);
    if (acceptedAdvisories.has(entry.url)) return [];
    return [{ package: name, severity: entry.severity, title: entry.title, url: entry.url }];
  });
}

const unresolved = Object.keys(vulnerabilities).flatMap((name) => unresolvedAdvisories(name));
const uniqueUnresolved = [...new Map(unresolved.map((finding) => [finding.url, finding])).values()];

if (uniqueUnresolved.length > 0) {
  process.stderr.write(`${JSON.stringify(uniqueUnresolved, null, 2)}\n`);
  process.exit(1);
}

const acceptedPresent = Object.values(vulnerabilities).some((vulnerability) =>
  vulnerability.via.some(
    (entry) => typeof entry === "object" && acceptedAdvisories.has(entry.url),
  ),
);

if (audit.status !== 0 && !acceptedPresent) {
  process.stderr.write(audit.stdout);
  process.exit(audit.status ?? 1);
}

for (const [url, reason] of acceptedAdvisories) {
  if (audit.stdout.includes(url)) {
    process.stdout.write(`Accepted non-applicable advisory ${url}: ${reason}\n`);
  }
}
process.stdout.write("No applicable high-severity npm audit findings.\n");
