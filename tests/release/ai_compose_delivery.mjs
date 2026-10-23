// F03/G02: stopped-state paired upgrade/rollback and security checks on actual Compose.
import assert from 'node:assert/strict';
import {cpSync, mkdirSync, readFileSync, symlinkSync, writeFileSync} from 'node:fs';
import {join} from 'node:path';
import {randomUUID, createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';

export async function exerciseDelivery(e) {
  const {root, env, compose, run, inspect, until, request, refreshOrigin} = e;
  assert(process.env.FOLIOPATH_TEST_UPGRADE_IMAGE, 'explicit upgrade image required');
  const nextImage = run('image', 'inspect', process.env.FOLIOPATH_TEST_UPGRADE_IMAGE, '--format', '{{.Id}}');
  const oldImage = env.FOLIOPATH_IMAGE;
  assert.notEqual(nextImage, oldImage, 'upgrade must exercise two different actual images');
  const json = async (path, method = 'GET', body, headers = {}) => {
    const response = await request(path, method, body, true, headers);
    const value = await response.json();
    assert(response.ok, `${method} ${path}: ${response.status} ${JSON.stringify(value)}`);
    return value;
  };
  const ready = () => until('two models ready', () => json('/ai/models'), p => p.items.filter(m => m.active && m.state === 'available').length === 2);
  async function start(image, data) {
    env.FOLIOPATH_IMAGE = image; env.FOLIOPATH_DATA_PATH = data;
    compose('up', '-d', '--force-recreate', 'foliopath');
    await until('core healthy', () => inspect(compose('ps', '-q', 'foliopath')).State.Health.Status, s => s === 'healthy');
    refreshOrigin(); await ready();
  }
  const originalHashes = ['synthetic.png', 'faces.png'].map(n => createHash('sha256').update(readFileSync(join(root, 'media', n))).digest('hex'));
  const library = (await json('/libraries', 'POST', {name: 'P5 delivery', rootPath: ''}, {'Idempotency-Key': randomUUID()})).library;
  const base = '/libraries/' + library.id;
  const assets = (await until('scan', () => json(base + '/assets'), v => v.items.length === 2)).items;
  async function done(operation) {
    const value = await until('operation', () => json('/ai/operations/' + operation.id), v => ['succeeded', 'failed', 'cancelled'].includes(v.state));
    assert.equal(value.state, 'succeeded', JSON.stringify(value)); return value;
  }
  async function analyze(kind) {
    const result = await done(await json(base + `/ai/${kind}/jobs`, 'POST', {mode: 'all'}, {'Idempotency-Key': randomUUID()}));
    assert.equal(result.completedItems, 2);
  }
  async function enable(kind) {
    const current = await request(base + '/ai/' + kind);
    await json(base + '/ai/' + kind, 'PATCH', {enabled: true}, {'If-Match': current.headers.get('etag')});
  }
  for (const kind of ['semantic', 'faces']) { await enable(kind); await analyze(kind); }
  const asset = assets.find(a => a.name === 'faces.png');
  const face = (await json('/assets/' + asset.id + '/faces')).items[0]; assert(face);
  const personName = 'P5-private-name-' + randomUUID();
  const person = await json('/people', 'POST', {name: personName}, {'Idempotency-Key': randomUUID()});
  await json('/ai/face-reviews', 'POST', {action: 'assign_face', faceId: face.faceId,
    expectedFaceRevision: face.revision, personId: person.id, expectedPersonRevision: person.revision}, {'Idempotency-Key': randomUUID()});
  const tag = await json('/tags', 'POST', {name: 'P5 manual'});
  const curation = await request('/assets/' + asset.id + '/curation');
  await json('/assets/' + asset.id + '/tags', 'PUT', {tagIds: [tag.id]}, {'If-Match': curation.headers.get('etag')});
  const oldGeneration = (await json(base + '/ai/semantic')).activeGenerationId;
  const faceGeneration = (await json(base + '/ai/faces')).activeGenerationId;
  async function preserved(confirmed = 1) {
    assert.equal((await json('/people/' + person.id)).confirmedFaceCount, confirmed);
    assert((await json('/assets/' + asset.id + '/curation')).tags.some(t => t.id === tag.id));
    assert.equal((await json(base + '/ai/faces')).activeGenerationId, faceGeneration);
    assert.deepEqual(['synthetic.png', 'faces.png'].map(n => createHash('sha256').update(readFileSync(join(root, 'media', n))).digest('hex')), originalHashes);
  }
  await preserved();
  const data = env.FOLIOPATH_DATA_PATH, backup = join(root, 'pre-upgrade'), restored = join(root, 'restored');
  compose('stop', 'foliopath');
  assert.equal(inspect(compose('ps', '-a', '-q', 'foliopath')).State.Running, false);
  cpSync(data, backup, {recursive: true, errorOnExist: true, force: false});
  await start(nextImage, data);
  const upgraded = await until('new semantic generation', () => json(base + '/ai/semantic'), s => s.activeGenerationId && s.activeGenerationId !== oldGeneration);
  assert.equal(upgraded.coverage.completed, 0, 'old vectors cannot count toward new preprocessing generation');
  await preserved(); await analyze('semantic');
  await start(nextImage, data); await preserved();
  assert.equal((await json(base + '/ai/semantic')).activeGenerationId, upgraded.activeGenerationId);
  console.log('PASS F03 stopped backup, actual image upgrade, fresh generation, curation and restart persistence');
  compose('stop', 'foliopath');
  cpSync(backup, restored, {recursive: true, errorOnExist: true, force: false});
  await start(oldImage, restored); await preserved();
  assert.equal((await json(base + '/ai/semantic')).activeGenerationId, oldGeneration);
  assert.equal((await json(base + '/ai/semantic')).coverage.completed, 2);
  console.log('PASS F03 old image plus pre-upgrade backup restores original generation and manual data');
  compose('stop', 'foliopath');
  await start(nextImage, data); await preserved();

  // All G02 HTTP and persisted-state checks below use the latest image.
  for (const path of ['/people', '/assets/' + asset.id + '/faces', '/assets/' + asset.id + '/content', '/system-logs', '/ai/models']) {
    assert.equal((await request(path, 'GET', undefined, false)).status, 401, path);
  }
  for (const headers of [{'X-CSRF-Token': ''}, {'X-CSRF-Token': 'wrong-token', Origin: 'http://untrusted.invalid'}]) {
    const response = await request('/people', 'POST', {name: 'must-not-create'}, true, {...headers, 'Idempotency-Key': randomUUID()});
    assert.equal(response.status, 403);
    assert.equal((await response.json()).error.code, 'csrf_invalid');
  }
  for (const path of ['/auth/setup', '/auth/login']) {
    const response = await request(path, 'POST', {}, false, {Origin: 'http://untrusted.invalid'});
    assert.equal(response.status, 403);
    assert.equal((await response.json()).error.code, 'origin_invalid');
  }
  assert.equal((await json('/people')).items.length, 1);
  for (const path of ['../escape', '%2e%2e/escape', '/etc', 'a\u0000b']) {
    const response = await request('/libraries', 'POST', {name: 'invalid', rootPath: path}, true, {'Idempotency-Key': randomUUID()});
    assert([400, 422].includes(response.status), `unsafe root ${response.status}`);
  }
  // Create only a synthetic symlink escape; the outside directory contains no user data.
  const outside = join(root, 'outside'); mkdirSync(outside); writeFileSync(join(outside, 'secret.txt'), 'P5-boundary-canary');
  symlinkSync(outside, join(root, 'media', 'escape'));
  const escaped = await request('/library-paths?parent=escape');
  assert.equal(escaped.status, 409, 'symlink browse must fail closed');
  assert.equal((await escaped.json()).error.code, 'library_root_symlink');
  const range = await request('/assets/' + asset.id + '/content', 'GET', undefined, true, {Range: 'bytes=0-15'});
  assert.equal(range.status, 206); assert.equal((await range.arrayBuffer()).byteLength, 16);
  console.log('PASS G02 authentication, CSRF/origin, traversal/symlink rejection and read-only original Range');

  const query = 'P5-private-query-' + randomUUID();
  const search = await json('/semantic/assets?libraryId=' + library.id + '&q=' + encodeURIComponent(query) + '&limit=10');
  const facePage = await json('/assets/' + asset.id + '/faces');
  const unsafe = /"(?:embedding|vector|landmarks|sourcePath|absolutePath)"\s*:/i;
  for (const value of [search, facePage, await json('/ai/models')]) {
    assert(!unsafe.test(JSON.stringify(value)), 'public API leaked sensitive inference fields');
    assert(!JSON.stringify(value).includes(root));
  }
  for (const path of ['/system-logs', '/diagnostics/media-failures']) {
    const value = JSON.stringify(await json(path));
    for (const marker of [personName, query, root]) assert(!value.includes(marker), `${path} leaked marker`);
    assert(!unsafe.test(value));
  }
  for (const service of ['foliopath', 'machine-learning']) {
    const output = spawnSync('docker', ['logs', compose('ps', '-q', service)], {encoding: 'utf8', timeout: 30000});
    assert.equal(output.status, 0);
    const logs = output.stdout + output.stderr;
    for (const marker of [personName, query, root]) assert(!logs.includes(marker), service + ' log leak');
  }
  console.log('PASS G02 public vector boundary and log/diagnostic name/query/path canaries');

  let settings = await request(base + '/ai/faces');
  await done(await json(base + '/ai/faces/derived-clear', 'POST', {confirmation: 'clear_derived_face_data'},
    {'If-Match': settings.headers.get('etag'), 'Idempotency-Key': randomUUID()}));
  const cleared = await request('/assets/' + asset.id + '/faces');
  assert.equal(cleared.status, 409);
  assert.equal((await cleared.json()).error.code, 'face_not_ready');
  assert.equal((await json('/people/' + person.id)).name, personName, 'derived clear must preserve manual people');
  compose('stop', 'foliopath');
  const counts = run('run', '--rm', '--network', 'none', '--read-only', '--tmpfs', '/tmp:size=256m', '--entrypoint', 'python',
    '--mount', `type=bind,src=${data},dst=/state,readonly`, env.FOLIOPATH_AI_IMAGE, '-c',
    'import sqlite3,json,shutil; shutil.copytree("/state","/tmp/state"); c=sqlite3.connect("file:/tmp/state/foliopath.db?mode=ro",uri=True); print(json.dumps({t:c.execute("SELECT count(*) FROM "+t).fetchone()[0] for t in ["face_observations","person_face_anchors","people"]}))');
  const rows = JSON.parse(counts); assert.equal(rows.face_observations, 0); assert.equal(rows.person_face_anchors, 1); assert.equal(rows.people, 1);
  await start(nextImage, data); await enable('faces'); await analyze('faces'); await preserved();
  settings = await request(base + '/ai/faces');
  await done(await json(base + '/ai/faces/manual-clear', 'POST', {confirmation: 'clear_manual_face_relationships',
    expectedPersonCount: 1, expectedAssignmentCount: 1, expectedConstraintCount: 0},
    {'If-Match': settings.headers.get('etag'), 'Idempotency-Key': randomUUID()}));
  await preserved(0);
  const manualCleared = await request('/assets/' + asset.id + '/faces');
  assert.equal(manualCleared.status, 409);
  assert.equal((await manualCleared.json()).error.code, 'face_not_ready');
  // Both clear operations invalidate the public ready projection. Retention is
  // verified below against the stopped database, not inferred from HTTP status.
  compose('stop', 'foliopath');
  const manualRows = JSON.parse(run('run', '--rm', '--network', 'none', '--read-only', '--tmpfs', '/tmp:size=256m', '--entrypoint', 'python',
    '--mount', `type=bind,src=${data},dst=/state,readonly`, env.FOLIOPATH_AI_IMAGE, '-c',
    'import sqlite3,json,shutil; shutil.copytree("/state","/tmp/state"); c=sqlite3.connect("file:/tmp/state/foliopath.db?mode=ro",uri=True); result={t:c.execute("SELECT count(*) FROM "+t).fetchone()[0] for t in ["face_observations","person_face_anchors","people"]}; result["face_audit_events"]=c.execute("SELECT count(*) FROM face_audit_events WHERE library_id IS NOT NULL").fetchone()[0]; result["face_review_undo_snapshots"]=c.execute("SELECT count(*) FROM face_review_undo_snapshots WHERE event_id IN (SELECT id FROM face_audit_events WHERE library_id IS NOT NULL)").fetchone()[0]; print(json.dumps(result))'));
  assert(manualRows.face_observations > 0);
  for (const table of ['person_face_anchors', 'face_audit_events', 'face_review_undo_snapshots']) assert.equal(manualRows[table], 0, table);
  assert.equal(manualRows.people, 1);
  await start(nextImage, data); await preserved(0);
  console.log('PASS G02 derived clear removes live vectors but retains anchors/people; rebuild and separate manual clear preserve media/tags');
  console.log(JSON.stringify({oldImage, nextImage, oldGeneration, newGeneration: upgraded.activeGenerationId, deliveryAndSecurityPassed: true}));
}
