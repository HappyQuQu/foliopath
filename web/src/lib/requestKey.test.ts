import { describe, expect, it, vi } from "vitest";

import { createRequestKey } from "./requestKey";

describe("createRequestKey", () => {
  it("uses randomUUID when the browser exposes it", () => {
    const randomUUID = vi.fn(() => "native-request-key");

    expect(createRequestKey({ randomUUID })).toBe("native-request-key");
    expect(randomUUID).toHaveBeenCalledOnce();
  });

  it("uses getRandomValues on non-secure LAN HTTP origins", () => {
    const getRandomValues = vi.fn((bytes: Uint8Array) => {
      bytes.fill(0xab);
      return bytes;
    });

    expect(createRequestKey({ getRandomValues })).toBe(
      "abababab-abab-4bab-abab-abababababab",
    );
    expect(getRandomValues).toHaveBeenCalledOnce();
  });

  it("still returns an accepted key when Web Crypto is unavailable", () => {
    vi.spyOn(Math, "random").mockReturnValue(0.5);

    expect(createRequestKey({})).toBe(
      "80808080-8080-4080-8080-808080808080",
    );
  });
});
