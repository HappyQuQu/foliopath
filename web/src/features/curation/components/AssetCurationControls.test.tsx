import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Tag } from "../../../lib/api/curation";
import {
  useAssetCurationQuery,
  useCreateAndAssignTagMutation,
  useFavoriteMutation,
  useReplaceAssetTagsMutation,
  useTagsQuery,
} from "../queries";
import { AssetCurationControls } from "./AssetCurationControls";

vi.mock("../queries", () => ({
  useAssetCurationQuery: vi.fn(),
  useCreateAndAssignTagMutation: vi.fn(),
  useFavoriteMutation: vi.fn(),
  useReplaceAssetTagsMutation: vi.fn(),
  useTagsQuery: vi.fn(),
}));

const createTag = vi.fn();
const fetchNextPage = vi.fn().mockResolvedValue(undefined);
const replaceTags = vi.fn();
const setFavorite = vi.fn();

function tag(id: string, name: string): Tag {
  return {
    assetCount: 0,
    createdAt: "2026-08-12T00:00:00Z",
    id,
    name,
    updatedAt: "2026-08-12T00:00:00Z",
  };
}

function mockState(tags: Tag[] = [tag("tag-selected", "已选择")]) {
  vi.mocked(useAssetCurationQuery).mockReturnValue({
    data: {
      assetId: "asset-1",
      favorite: false,
      favoritedAt: null,
      revision: 7,
      tags,
    },
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  } as never);
}

beforeEach(() => {
  vi.clearAllMocks();
  fetchNextPage.mockResolvedValue(undefined);
  mockState();
  vi.mocked(useTagsQuery).mockReturnValue({
    data: {
      pageParams: [undefined],
      pages: [{
        items: [tag("tag-selected", "已选择"), tag("tag-travel", "旅行")],
        nextCursor: "next-page",
      }],
    },
    fetchNextPage,
    hasNextPage: true,
    isError: false,
    isFetchNextPageError: false,
    isFetchingNextPage: false,
    isPending: false,
    refetch: vi.fn(),
  } as never);
  vi.mocked(useCreateAndAssignTagMutation).mockReturnValue({
    isError: false,
    isPending: false,
    mutate: createTag,
  } as never);
  vi.mocked(useFavoriteMutation).mockReturnValue({
    isError: false,
    isPending: false,
    mutate: setFavorite,
  } as never);
  vi.mocked(useReplaceAssetTagsMutation).mockReturnValue({
    isError: false,
    isPending: false,
    mutate: replaceTags,
  } as never);
});

describe("AssetCurationControls", () => {
  it("adds an unselected existing tag directly and can load more tags", async () => {
    const user = userEvent.setup();
    render(<AssetCurationControls assetId="asset-1" csrfToken="csrf-token" />);

    expect(
      screen.queryByRole("button", { name: "添加已有标签 已选择" }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "添加已有标签 旅行" }));

    expect(replaceTags).toHaveBeenCalledWith({
      assetId: "asset-1",
      csrfToken: "csrf-token",
      revision: 7,
      tagIds: ["tag-selected", "tag-travel"],
    });

    await user.click(screen.getByRole("button", { name: "载入更多标签" }));
    expect(fetchNextPage).toHaveBeenCalledOnce();
  });

  it("keeps the text field for creating a new tag and removes selected tags", async () => {
    const user = userEvent.setup();
    render(<AssetCurationControls assetId="asset-1" csrfToken="csrf-token" />);

    expect(screen.getByText("输入仅用于新建标签；已有标签请直接选择。")).toBeVisible();
    const input = screen.getByRole("textbox", { name: "新标签名称" });
    await user.type(input, "家人");
    await user.click(screen.getByRole("button", { name: "新建标签" }));

    expect(createTag).toHaveBeenCalledWith(
      { assetId: "asset-1", csrfToken: "csrf-token", name: "家人" },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );

    await user.click(screen.getByRole("button", { name: "移除标签 已选择" }));
    expect(replaceTags).toHaveBeenCalledWith({
      assetId: "asset-1",
      csrfToken: "csrf-token",
      revision: 7,
      tagIds: [],
    });
  });

  it("disables adding or creating tags at the 20-tag limit", () => {
    mockState(Array.from({ length: 20 }, (_, index) => tag(`tag-${index}`, `标签${index}`)));
    vi.mocked(useTagsQuery).mockReturnValue({
      data: {
        pageParams: [undefined],
        pages: [{ items: [tag("tag-extra", "更多")], nextCursor: null }],
      },
      fetchNextPage,
      hasNextPage: false,
      isError: false,
      isFetchNextPageError: false,
      isFetchingNextPage: false,
      isPending: false,
      refetch: vi.fn(),
    } as never);

    render(<AssetCurationControls assetId="asset-1" csrfToken="csrf-token" />);

    expect(screen.getByRole("button", { name: "添加已有标签 更多" })).toBeDisabled();
    expect(screen.getByRole("textbox", { name: "新标签名称" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "新建标签" })).toBeDisabled();
  });
});
