// Local diagnostic only: not a race-safe production filesystem boundary or attestation.
import { createHash } from 'node:crypto';
import { createReadStream } from 'node:fs';
import { lstat, readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

export async function inventory(root) {
  const files = [];
  let bytes = 0;
  async function walk(relative = '', depth = 0) {
    if (depth > 16) throw new Error('cache depth limit');
    const directory = resolve(root, relative);
    if (!(await lstat(directory)).isDirectory()) throw new Error('cache directory required');
    const names = (await readdir(directory)).sort();
    if (names.length > 1000) throw new Error('cache entry limit');
    for (const name of names) {
      const path = relative ? `${relative}/${name}` : name;
      const full = resolve(root, path);
      const before = await lstat(full);
      if (before.isSymbolicLink()) throw new Error('cache symlink rejected');
      if (before.isDirectory()) { await walk(path, depth + 1); continue; }
      if (!before.isFile()) throw new Error('cache regular file required');
      bytes += before.size;
      if (files.length >= 1000 || bytes > 2 ** 31) throw new Error('cache size limit');
      const hash = createHash('sha256');
      let read = 0;
      for await (const chunk of createReadStream(full)) {
        read += chunk.length;
        if (read > before.size) throw new Error('cache changed during audit');
        hash.update(chunk);
      }
      const after = await lstat(full);
      if (read !== before.size || before.ino !== after.ino || before.size !== after.size ||
          before.mtimeMs !== after.mtimeMs || before.ctimeMs !== after.ctimeMs) {
        throw new Error('cache changed during audit');
      }
      files.push({ path, size: read, sha256: hash.digest('hex') });
    }
  }
  await walk();
  return files;
}

export function fingerprint(files) {
  return createHash('sha256').update(JSON.stringify(files)).digest('hex');
}

export async function verify(root, expected) {
  const files = await inventory(root);
  if (JSON.stringify(files) !== JSON.stringify(expected)) throw new Error('cache baseline mismatch');
  return { files: files.length, bytes: files.reduce((sum, file) => sum + file.size, 0), sha256: fingerprint(files) };
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try {
    const root = process.env.FOLIOPATH_TEST_MODEL_CACHE;
    if (!root) throw new Error('set FOLIOPATH_TEST_MODEL_CACHE');
    const baseline = JSON.parse(await readFile(new URL('./cache-baseline.json', import.meta.url), 'utf8'));
    console.log(JSON.stringify(await verify(root, baseline)));
  } catch {
    console.error('Local model cache verification failed; baseline or cache requires review.');
    process.exitCode = 1;
  }
}
