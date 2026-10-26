// Synthetic originals, real same-photo pair save/replay/merge blocking/undo only.
process.env.FOLIOPATH_GROUP_REVIEW_REAL = '1';
process.env.FOLIOPATH_PHOTO_FACE_REAL = '1';
process.env.FOLIOPATH_FACE_UNDO_REAL = '1';
process.env.FOLIOPATH_PERSON_RECOVERY_REAL = '1';
process.env.FOLIOPATH_CANNOT_LINK_REAL = '1';
process.env.FOLIOPATH_PERSON_APP_IMAGE ??= 'foliopath:local-cannot-link-20261025';
await import('./person_review_smoke.mjs');
