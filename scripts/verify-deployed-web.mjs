import { createHash } from "node:crypto";
import { readFile, readdir } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

// Compare a freshly built dist with the bytes served by the candidate or deployment.
// A healthy Go process or an image tag alone cannot prove the embedded SPA version.
export async function verifyDeployedWeb(base, directory, routes = []) {
  const origin = new URL(base);
  if (!['http:', 'https:'].includes(origin.protocol) || origin.username || origin.password ||
      origin.pathname !== '/' || origin.search || origin.hash) {
    throw new Error('Expected an HTTP(S) origin without credentials or a path');
  }
  const files = (await readdir(directory, { recursive: true, withFileTypes: true }))
    .filter((entry) => entry.isFile() && !entry.name.startsWith('.'))
    .map((entry) => resolve(entry.parentPath, entry.name));
  const root = resolve(directory);
  const index = await readFile(resolve(root, 'index.html'));
  const checked = [];

  async function compare(path, expected) {
    const response = await fetch(new URL(path, origin), {
      redirect: 'error',
      headers: { 'Cache-Control': 'no-cache' },
      signal: AbortSignal.timeout(15000),
    });
    if (response.status !== 200) throw new Error(`${path}: HTTP ${response.status}`);
    const actual = Buffer.from(await response.arrayBuffer());
    if (!actual.equals(expected)) throw new Error(`${path}: deployed bytes differ from fresh build`);
    checked.push({ path, bytes: actual.length, sha256: createHash('sha256').update(actual).digest('hex') });
  }

  for (const file of files.sort()) {
    const relative = file.slice(root.length + 1);
    // Go's FileServer canonically redirects /index.html to /.
    const path = relative === 'index.html' ? '/' : `/${relative.split('/').map(encodeURIComponent).join('/')}`;
    await compare(path, await readFile(file));
  }
  for (const route of routes) {
    if (!/^\/(?!\/)[a-zA-Z0-9/_-]*$/.test(route)) throw new Error('Expected an application route');
    await compare(route, index);
  }
  return checked;
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const [base, directory, ...routes] = process.argv.slice(2);
  if (!base || !directory) throw new Error('Usage: node scripts/verify-deployed-web.mjs ORIGIN DIST [ROUTE ...]');
  const result = await verifyDeployedWeb(base, directory, routes);
  process.stdout.write(`${JSON.stringify({ status: 'verified', files: result }, null, 2)}\n`);
}
