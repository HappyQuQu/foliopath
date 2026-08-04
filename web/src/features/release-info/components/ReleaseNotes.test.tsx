import { describe, expect, it } from "vitest";

import { releaseNoteBlocks } from "./ReleaseNotes";

describe("releaseNoteBlocks", () => {
  it("keeps user-facing release content without links or commit references", () => {
    expect(
      releaseNoteBlocks(
        "## [1.0.1](https://example.test/compare) (2026-08-03)\n\n" +
          "### 🐛 修复\n\n" +
          "* Docker 镜像标签与应用版本保持一致 " +
          "([46ce9b6](https://example.test/commit/46ce9b6))",
        "v1.0.1",
      ),
    ).toEqual([
      { kind: "heading", text: "🐛 修复" },
      { kind: "list", items: ["Docker 镜像标签与应用版本保持一致"] },
    ]);
  });
});
