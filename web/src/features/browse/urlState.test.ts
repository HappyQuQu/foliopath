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
      kind: "all",
      order: "asc",
      recursive: false,
      sort: "name",
    });
    expect(parseBrowseUrlState(new URLSearchParams("recursive=1"))).toEqual({
      kind: "all",
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
      kind: "all",
      order: "asc",
      recursive: true,
      sort: "name",
    });
    expect(
      serializeBrowseUrlState({
        kind: "all",
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
      kind: "all",
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

  it("keeps an explicit all-media destination distinct from root recursion", () => {
    const allMediaState = parseBrowseUrlState(
      new URLSearchParams("view=all"),
    );

    expect(allMediaState).toEqual({
      allMedia: true,
      kind: "all",
      order: "desc",
      recursive: true,
      sort: "modifiedAt",
    });
    expect(serializeBrowseUrlState(allMediaState)).toBe(
      "recursive=1&view=all",
    );
    expect(
      browseUrl("lib_family", "dir_japan", allMediaState),
    ).toBe("/libraries/lib_family/browse/dir_japan?recursive=1");
  });

  it("preserves the three-state media filter in canonical browse links", () => {
    expect(
      parseBrowseUrlState(new URLSearchParams("kind=image")),
    ).toEqual({
      kind: "image",
      order: "asc",
      recursive: false,
      sort: "name",
    });
    expect(
      serializeBrowseUrlState({
        kind: "video",
        order: "desc",
        recursive: true,
        sort: "modifiedAt",
      }),
    ).toBe("recursive=1&kind=video");
    expect(
      parseBrowseUrlState(new URLSearchParams("kind=animated")).kind,
    ).toBe("all");
  });
});
