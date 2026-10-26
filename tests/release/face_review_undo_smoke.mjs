// Actual model/API/UI, synthetic originals and independent Linux Docker data only.
process.env.FOLIOPATH_GROUP_REVIEW_REAL = '1';
process.env.FOLIOPATH_PHOTO_FACE_REAL = '1';
process.env.FOLIOPATH_FACE_UNDO_REAL = '1';
process.env.FOLIOPATH_PERSON_APP_IMAGE ??= 'foliopath:local-face-undo-20261025';
await import('./person_review_smoke.mjs');
