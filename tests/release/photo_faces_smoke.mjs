// Real model, real guarded API and UI, synthetic originals and independent Docker data only.
process.env.FOLIOPATH_GROUP_REVIEW_REAL = '1';
process.env.FOLIOPATH_PHOTO_FACE_REAL = '1';
process.env.FOLIOPATH_PERSON_APP_IMAGE ??= 'foliopath:local-photo-faces-20261025';
await import('./person_review_smoke.mjs');
