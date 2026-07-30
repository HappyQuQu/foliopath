import { describe, expect, it } from "vitest";

import { serializeAPIQuery } from "./client";

describe("apiClient query serialization", () => {
  it("serializes array filters as the comma-delimited OpenAPI contract", () => {
    const query = serializeAPIQuery({
      directoryId: "dir_travel",
      kind: ["image", "animated"],
      limit: 50,
      order: "desc",
      recursive: true,
      sort: "modifiedAt",
    });

    const search = new URLSearchParams(query);
    expect(search.getAll("kind")).toEqual(["image,animated"]);
  });
});
