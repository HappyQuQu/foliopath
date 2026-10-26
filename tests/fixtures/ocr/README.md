# Synthetic OCR fixture data

CR-2026-045. Created by `spikes/ocr/create_fixtures.py` using locally supplied
STHeiti Medium for synthetic Chinese/English strings on white backgrounds.
No real media, personal information, model output or source font is included.
The source font is neither required to use these fixtures nor redistributed.

`bilingual.png`: four horizontal lines; `exif-oriented.jpg`: equivalent text
with EXIF orientation; `wide.png`: longest-edge reduction; `blank.png`: zero text;
`vertical.png`: 90-degree observation, **not an accuracy pass**;
`over-pixels.png`: six-million-pixel rejected input;
`corrupt.png`: deliberate invalid synthetic bytes, not a damaged real file.

Run the isolated Docker probe using this directory. Hashes are recorded in
`docs/evidence/ocr/2026-10-25-ocr-feasibility-local.json`; the probe verifies
that fixture and model bytes remain unchanged. OCR index/search is not enabled.
