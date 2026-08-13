import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import type { MediaFailure } from "../../../lib/api/diagnostics";
import {
  readClearedMediaFailureRevision,
} from "../../../lib/storage/preferences";
import { useMediaFailureQuery, useMediaFailuresQuery } from "../../diagnostics";
import { LibraryProcessingResults } from "./LibraryRecords";

vi.mock("../../diagnostics", () => ({
  useMediaFailureQuery: vi.fn(),
  useMediaFailuresQuery: vi.fn(),
}));

const failure: MediaFailure = {
  assetId: "asset_1",
  attempts: 1,
  errorCode: "invalid_media",
  finishedAt: "2026-08-13T01:02:03.004Z",
  id: "mjob_9",
  libraryId: "lib_1",
  libraryName: "照片",
  relativePath: "broken.jpg",
  variant: "grid",
};

beforeEach(() => {
  window.localStorage.clear();
  vi.mocked(useMediaFailuresQuery).mockReturnValue({
    data: {
      pageParams: [undefined],
      pages: [{
        items: [failure],
        nextCursor: null,
        revision: "mfailrev_1786582923004_9",
      }],
    },
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isError: false,
    isFetchingNextPage: false,
    isPending: false,
    isSuccess: true,
    refetch: vi.fn(),
  } as never);
  vi.mocked(useMediaFailureQuery).mockReturnValue({
    data: undefined,
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  } as never);
});

it("clears current processing results without deleting diagnostics and can restore them", async () => {
  const user = userEvent.setup();
  const view = render(
    <MemoryRouter>
      <LibraryProcessingResults libraries={[]} />
    </MemoryRouter>,
  );

  expect(screen.getByText("broken.jpg")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "清除当前结果" }));

  expect(screen.queryByText("broken.jpg")).not.toBeInTheDocument();
  expect(screen.getByText("当前结果已清除")).toBeVisible();
  expect(readClearedMediaFailureRevision()).toBe("mfailrev_1786582923004_9");

  const newerFailure = {
    ...failure,
    finishedAt: "2026-08-13T01:03:03.004Z",
    id: "mjob_10",
    relativePath: "new-failure.jpg",
  };
  vi.mocked(useMediaFailuresQuery).mockReturnValue({
    data: {
      pageParams: [undefined],
      pages: [{
        items: [newerFailure, failure],
        nextCursor: null,
        revision: "mfailrev_1786582983004_10",
      }],
    },
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isError: false,
    isFetchingNextPage: false,
    isPending: false,
    isSuccess: true,
    refetch: vi.fn(),
  } as never);
  view.rerender(
    <MemoryRouter>
      <LibraryProcessingResults libraries={[]} />
    </MemoryRouter>,
  );
  expect(screen.getByText("new-failure.jpg")).toBeVisible();
  expect(screen.queryByText("broken.jpg")).not.toBeInTheDocument();

  await user.click(screen.getByRole("button", { name: "恢复已清除结果" }));
  expect(screen.getByText("broken.jpg")).toBeVisible();
  expect(readClearedMediaFailureRevision()).toBeUndefined();
});
