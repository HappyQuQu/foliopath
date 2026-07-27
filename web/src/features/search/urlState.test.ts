import { describe, expect, it } from "vitest";

import {
  defaultSearchUrlState,
  kindsForSearch,
  modifiedRangeForSearch,
  parseSearchUrlState,
  searchUrl,
  serializeSearchUrlState,
} from "./urlState";

describe("search URL state", () => {
  it("defaults to the current library and canonical modification-time order", () => {
    expect(parseSearchUrlState(new URLSearchParams(), "lib_family")).toEqual(
      defaultSearchUrlState("lib_family"),
    );
    expect(parseSearchUrlState(new URLSearchParams())).toEqual(
      defaultSearchUrlState(),
    );
  });

  it("restores directory scope, filters, and sorting from a copyable URL", () => {
    const state = parseSearchUrlState(
      new URLSearchParams(
        "q=%E4%BA%AC%E9%83%BD&scope=directory&directoryId=dir_japan&recursive=1&kind=video&date=30d&sort=name&order=asc",
      ),
      "lib_family",
    );

    expect(state).toEqual({
      date: "30d",
      directoryId: "dir_japan",
      kind: "video",
      order: "asc",
      q: "京都",
      recursive: true,
      scope: "directory",
      sort: "name",
    });
    expect(searchUrl("lib_family", state)).toBe(
      "/libraries/lib_family/search?q=%E4%BA%AC%E9%83%BD&scope=directory&directoryId=dir_japan&recursive=1&kind=video&date=30d&sort=name&order=asc",
    );
  });

  it("normalizes impossible directory and global route combinations", () => {
    expect(
      parseSearchUrlState(
        new URLSearchParams("scope=directory&recursive=1"),
        "lib_family",
      ),
    ).toMatchObject({
      recursive: false,
      scope: "library",
    });
    expect(
      parseSearchUrlState(
        new URLSearchParams(
          "scope=directory&directoryId=dir_private&recursive=1",
        ),
      ),
    ).toMatchObject({
      recursive: false,
      scope: "all",
    });
  });

  it("maps type and relative date filters to exact API values", () => {
    expect(kindsForSearch("image")).toEqual(["image"]);
    expect(kindsForSearch("all")).toBeUndefined();
    expect(
      modifiedRangeForSearch("30d", new Date("2026-07-28T12:00:00.000Z")),
    ).toEqual({
      modifiedBefore: "2026-07-28T12:00:00.000Z",
      modifiedFrom: "2026-06-28T12:00:00.000Z",
    });
    expect(
      modifiedRangeForSearch("year", new Date("2026-07-28T12:00:00.000Z")),
    ).toEqual({
      modifiedBefore: "2026-07-28T12:00:00.000Z",
      modifiedFrom: "2026-01-01T00:00:00.000Z",
    });
  });

  it("omits default filters from the canonical query string", () => {
    expect(
      serializeSearchUrlState({
        ...defaultSearchUrlState("lib_family"),
        q: "京都",
      }),
    ).toBe("q=%E4%BA%AC%E9%83%BD");
  });
});
