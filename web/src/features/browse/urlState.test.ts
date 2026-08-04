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
      q: "",
      recursive: false,
      sort: "name",
    });
    expect(parseBrowseUrlState(new URLSearchParams("recursive=1"))).toEqual({
      kind: "all",
      order: "desc",
      q: "",
      recursive: true,
      sort: "modifiedAt",
    });
    expect(serializeBrowseUrlState(defaultBrowseUrlState())).toBe("");
    expect(serializeBrowseUrlState(defaultBrowseUrlState(true))).toBe(
      "recursive=1",
    );
  });

  it("applies a configured default sort only when the URL does not specify one", () => {
    expect(
      parseBrowseUrlState(new URLSearchParams(), "size:desc"),
    ).toMatchObject({ order: "desc", sort: "size" });
    expect(
      parseBrowseUrlState(new URLSearchParams(), "name:desc"),
    ).toMatchObject({ order: "desc", sort: "name" });
    expect(
      parseBrowseUrlState(
        new URLSearchParams("sort=name&order=asc"),
        "size:desc",
      ),
    ).toMatchObject({ order: "asc", sort: "name" });
    expect(
      serializeBrowseUrlState(
        parseBrowseUrlState(new URLSearchParams(), "size:desc"),
      ),
    ).toBe("sort=size&order=desc");
  });

  it("normalizes invalid values and preserves an explicit non-default sort", () => {
    expect(
      parseBrowseUrlState(
        new URLSearchParams("recursive=1&sort=name&order=asc"),
      ),
    ).toEqual({
      kind: "all",
      order: "asc",
      q: "",
      recursive: true,
      sort: "name",
    });
    expect(
      serializeBrowseUrlState({
        kind: "all",
        order: "asc",
        q: "",
        recursive: true,
        sort: "name",
      }),
    ).toBe("recursive=1&sort=name&order=asc");
    expect(
      parseBrowseUrlState(
        new URLSearchParams("recursive=1&sort=size&order=asc"),
      ),
    ).toMatchObject({ order: "asc", sort: "size" });
    expect(
      serializeBrowseUrlState({
        kind: "all",
        order: "desc",
        q: "",
        recursive: false,
        sort: "size",
      }),
    ).toBe("sort=size&order=desc");
    expect(
      parseBrowseUrlState(
        new URLSearchParams("recursive=yes&sort=unknown&order=sideways"),
      ),
    ).toEqual({
      kind: "all",
      order: "asc",
      q: "",
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
    const allMediaState = parseBrowseUrlState(new URLSearchParams("view=all"));

    expect(allMediaState).toEqual({
      allMedia: true,
      kind: "all",
      order: "desc",
      q: "",
      recursive: true,
      sort: "modifiedAt",
    });
    expect(serializeBrowseUrlState(allMediaState)).toBe("recursive=1&view=all");
    expect(browseUrl("lib_family", "dir_japan", allMediaState)).toBe(
      "/libraries/lib_family/browse/dir_japan?recursive=1",
    );
  });

  it("preserves the three-state media filter in canonical browse links", () => {
    expect(parseBrowseUrlState(new URLSearchParams("kind=image"))).toEqual({
      kind: "image",
      order: "asc",
      q: "",
      recursive: false,
      sort: "name",
    });
    expect(
      serializeBrowseUrlState({
        kind: "video",
        order: "desc",
        q: "",
        recursive: true,
        sort: "modifiedAt",
      }),
    ).toBe("recursive=1&kind=video");
    expect(parseBrowseUrlState(new URLSearchParams("kind=animated")).kind).toBe(
      "all",
    );
  });

  it("keeps current-directory keywords in the URL and defaults filtered results to newest first", () => {
    expect(parseBrowseUrlState(new URLSearchParams("q=京都"))).toEqual({
      kind: "all",
      order: "desc",
      q: "京都",
      recursive: false,
      sort: "modifiedAt",
    });
    expect(
      serializeBrowseUrlState({
        kind: "image",
        order: "desc",
        q: "京都",
        recursive: false,
        sort: "modifiedAt",
      }),
    ).toBe("q=%E4%BA%AC%E9%83%BD&kind=image");
  });
});
