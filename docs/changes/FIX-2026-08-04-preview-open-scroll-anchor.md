# FIX-2026-08-04: Preserve the media anchor when preview opens

- Type: routine fix inside the accepted non-modal media preview slice
- Requirement / quality ID: S3-106 / UIF-405
- Target: MVP, Stage 3 browse and Stage 4 search
- Owner: shared Web media preview controller and virtual media collection
- Delivery gate: `docs/gates/MVP-2026-07-23/s4-frontend-media-matrix.md`
- Affected invariant: virtual recycling and responsive preview reflow must preserve
  the selected media and visible focus instead of returning the document to the top.

## Problem

Opening the first image or video preview narrows the desktop media collection. At a
deep document scroll position, the resulting virtual-grid reflow can unmount the
focused card before the browser establishes a new scroll anchor, leaving the user at
the top and unable to find the selected media.

## Change

After the first preview is committed and the grid has had two animation frames to
reflow, ask the canonical virtual collection controller to restore and focus the
activated asset. Later selections do not force a scroll because they do not open or
resize the preview layout. Closing the preview continues to use the same restoration
path.

## Evidence

- Controller regression: opening a preview restores the activated asset only after
  layout settlement; switching an already-open preview does not schedule another
  restoration.
- Existing collection coverage verifies that restoration uses virtualizer positioning
  and focuses the remounted trigger without browser-driven scrolling.
