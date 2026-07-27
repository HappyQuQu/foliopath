import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useSubmissionGuard } from "./useSubmissionGuard";

describe("useSubmissionGuard", () => {
  it("drops repeated submissions until the active request settles", async () => {
    let resolveFirst: (() => void) | undefined;
    const firstAction = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveFirst = resolve;
        }),
    );
    const repeatedAction = vi.fn(async () => undefined);
    const { result } = renderHook(() => useSubmissionGuard());

    let firstSubmission: Promise<void | undefined> | undefined;
    await act(async () => {
      firstSubmission = result.current(firstAction);
      await result.current(repeatedAction);
    });

    expect(firstAction).toHaveBeenCalledOnce();
    expect(repeatedAction).not.toHaveBeenCalled();

    await act(async () => {
      resolveFirst?.();
      await firstSubmission;
      await result.current(repeatedAction);
    });
    expect(repeatedAction).toHaveBeenCalledOnce();
  });
});
