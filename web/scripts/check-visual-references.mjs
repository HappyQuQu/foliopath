import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const repositoryRoot = path.resolve(webRoot, "..");
const manifestPath = path.join(webRoot, "qa", "visual-reference-manifest.json");
const manifest = JSON.parse(await readFile(manifestPath, "utf8"));

const expectedViewports = [
  { width: 390, height: 844 },
  { width: 768, height: 1024 },
  { width: 1265, height: 800 },
  { width: 1440, height: 900 },
];
const requiredEntries = [
  "account",
  "auth-login",
  "auth-setup",
  "browse",
  "general",
  "libraries",
  "library-new",
  "library-status",
  "search",
  "storage",
  "viewer",
  "welcome",
];

assert(manifest.schemaVersion === 1, "schemaVersion must be 1");
assert(
  manifest.sourceOfTruth === "prototypes/apple-redesign",
  "sourceOfTruth must remain the accepted Apple redesign prototype",
);
assert(
  JSON.stringify(manifest.viewports) === JSON.stringify(expectedViewports),
  "viewport matrix must be exactly 390/768/1265/1440 with approved heights",
);
assert(
  JSON.stringify(manifest.locales) === JSON.stringify(["zh-CN", "en"]),
  "locale matrix must be exactly zh-CN/en",
);
assert(
  JSON.stringify(manifest.themes) === JSON.stringify(["light", "dark"]),
  "theme matrix must be exactly light/dark",
);
assert(Array.isArray(manifest.entries), "entries must be an array");

const entryIds = manifest.entries.map((entry) => entry.id);
assert(
  JSON.stringify(entryIds) === JSON.stringify(requiredEntries),
  "entries must cover every approved production page in stable id order",
);
assert(new Set(entryIds).size === entryIds.length, "entry ids must be unique");

for (const entry of manifest.entries) {
  assert(
    typeof entry.productionRoute === "string" &&
      entry.productionRoute.startsWith("/"),
    `${entry.id}: productionRoute must be an application-relative route`,
  );
  assert(
    typeof entry.source === "string" &&
      entry.source.startsWith("prototypes/apple-redesign/") &&
      entry.source.endsWith(".html"),
    `${entry.id}: source must be an accepted prototype HTML page`,
  );
  await requireFile(entry.source, `${entry.id}: source`);

  assert(
    Array.isArray(entry.fixtures) && entry.fixtures.length > 0,
    `${entry.id}: at least one deterministic fixture is required`,
  );
  for (const fixture of entry.fixtures) {
    assert(
      fixture.startsWith("web/") &&
        (fixture.endsWith(".stories.tsx") ||
          fixture.endsWith(".test.tsx") ||
          fixture.endsWith(".spec.ts")),
      `${entry.id}: fixture must be a Storybook, component-test, or E2E source`,
    );
    await requireFile(fixture, `${entry.id}: fixture`);
  }

  assert(
    Array.isArray(entry.states) &&
      entry.states.length > 0 &&
      entry.states.every((state) => typeof state === "string" && state.length > 0),
    `${entry.id}: states must be a non-empty string list`,
  );
  assert(
    new Set(entry.states).size === entry.states.length,
    `${entry.id}: states must not contain duplicates`,
  );
}

console.log(
  `Visual reference manifest passed: ${manifest.entries.length} pages, ` +
    `${manifest.viewports.length} viewports, ${manifest.locales.length} locales, ` +
    `${manifest.themes.length} themes.`,
);

async function requireFile(relativePath, label) {
  const absolutePath = path.resolve(repositoryRoot, relativePath);
  assert(
    absolutePath.startsWith(`${repositoryRoot}${path.sep}`),
    `${label} escapes the repository`,
  );
  const metadata = await stat(absolutePath).catch(() => null);
  assert(metadata?.isFile(), `${label} does not exist: ${relativePath}`);
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}
