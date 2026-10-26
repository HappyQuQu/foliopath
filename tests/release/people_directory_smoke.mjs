// Reuse the bounded real-model group setup and verify directory reads and empty creation.
process.env.FOLIOPATH_GROUP_REVIEW_REAL = '1';
process.env.FOLIOPATH_PEOPLE_DIRECTORY_REAL = '1';
process.env.FOLIOPATH_PERSON_APP_IMAGE ??= 'foliopath:local-people-directory-20261025';
await import('./person_review_smoke.mjs');
