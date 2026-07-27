import { describe, expect, it } from "vitest";

import {
  browseUrl,
  defaultBrowseUrlState,
  parseBrowseUrlState,
  serializeBrowseUrlState,
} from "./urlState";

describe("browse URL state", () => {
  it("uses different canonical defaults for direct and recursive browsing", () => {
    expect(parseBrowseUrlState(new URLSearchParams())).toEqual({
      order: "asc",
      recursive: false,
      sort: "name",
    });
    expect(parseBrowseUrlState(new URLSearchParams("recursive=1"))).toEqual({
      order: "desc",
      recursive: true,
      sort: "modifiedAt",
    });
    expect(serializeBrowseUrlState(defaultBrowseUrlState())).toBe("");
    expect(serializeBrowseUrlState(defaultBrowseUrlState(true))).toBe(
      "recursive=1",
    );
  });

  it("normalizes invalid values and preserves an explicit non-default sort", () => {
    expect(
      parseBrowseUrlState(
        new URLSearchParams("recursive=1&sort=name&order=asc"),
      ),
    ).toEqual({
      order: "asc",
      recursive: true,
      sort: "name",
    });
    expect(
      serializeBrowseUrlState({
        order: "asc",
        recursive: true,
        sort: "name",
      }),
    ).toBe("recursive=1&sort=name&order=asc");
    expect(
      parseBrowseUrlState(
        new URLSearchParams("recursive=yes&sort=unknown&order=sideways"),
      ),
    ).toEqual({
      order: "asc",
      recursive: false,
      sort: "name",
    });
  });

  it("builds a copyable deep-directory URL without cursor state", () => {
    expect(
      browseUrl("lib_family", "dir_japan", defaultBrowseUrlState(true)),
    ).toBe("/libraries/lib_family/browse/dir_japan?recursive=1");
  });
});
