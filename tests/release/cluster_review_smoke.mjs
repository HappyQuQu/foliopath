// Reuse the same isolated setup, fixed models, source protection and cleanup.
process.env.FOLIOPATH_GROUP_REVIEW_REAL = '1';
process.env.FOLIOPATH_PERSON_APP_IMAGE ??= 'foliopath:local-pending-groups-20261025';
await import('./person_review_smoke.mjs');
