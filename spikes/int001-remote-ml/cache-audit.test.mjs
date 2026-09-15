import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, symlink, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { inventory, verify } from './cache-audit.mjs';

async function fixture(t) {
  const root = await mkdtemp(join(tmpdir(), 'foliopath-cache-audit-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await mkdir(join(root, 'textual'));
  await writeFile(join(root, 'textual', 'model.onnx'), 'synthetic weights');
  await writeFile(join(root, '.metadata'), 'synthetic metadata');
  return root;
}

test('stable full inventory includes hidden configuration', async t => {
  const root = await fixture(t);
  const expected = await inventory(root);
  assert.equal(expected.length, 2);
  assert.equal((await verify(root, expected)).files, 2);
  assert.deepEqual(await inventory(root), expected);
});
for (const variant of ['modified', 'missing', 'extra', 'symlink']) {
  test(`reject ${variant} cache`, async t => {
    const root = await fixture(t);
    const expected = await inventory(root);
    const path = join(root, 'textual', 'model.onnx');
    if (variant === 'modified') await writeFile(path, 'different weights');
    if (variant === 'missing') await rm(path);
    if (variant === 'extra') await writeFile(join(root, '.unexpected'), 'extra');
    if (variant === 'symlink') await symlink(path, join(root, 'link'));
    await assert.rejects(verify(root, expected));
  });
}
