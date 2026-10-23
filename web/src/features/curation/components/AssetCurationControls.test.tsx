import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Tag, AssetTag } from "../../../lib/api/curation";
import {
  useAddManualAssetTagMutation,
  useAssetCurationQuery,
  useCreateAndAssignTagMutation,
  useFavoriteMutation,
  useReplaceAssetTagsMutation,
  useTagsQuery,
} from "../queries";
import { AssetCurationControls } from "./AssetCurationControls";

vi.mock("../queries", () => ({
  useAddManualAssetTagMutation: vi.fn(),
  useAssetCurationQuery: vi.fn(),
  useCreateAndAssignTagMutation: vi.fn(),
  useFavoriteMutation: vi.fn(),
  useReplaceAssetTagsMutation: vi.fn(),
  useTagsQuery: vi.fn(),
}));

const createTag = vi.fn();
const addManualTag = vi.fn();
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

function mockState(tags: AssetTag[] = [tag("tag-selected", "已选择")]) {
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
  vi.mocked(useAddManualAssetTagMutation).mockReturnValue({ isError: false, isPending: false, mutate: addManualTag } as never);
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

it("distinguishes automatic and protected manual tags without an approval workflow", () => {
  mockState([{ ...tag("tag_ai", "flower"), source: "ai" }, { ...tag("tag_manual", "旅行"), source: "manual" }]);
  render(<AssetCurationControls assetId="asset-1" csrfToken="token" />);
  expect(screen.getByTitle("AI 识别的标签，可以手动删除")).toHaveTextContent("AI");
  expect(screen.getByTitle("人工标签，重新识别时保留")).toHaveTextContent("手动");
  expect(screen.getByText(/删除的标签不会在下次识别时自动加回/)).toBeInTheDocument();
});

describe("AssetCurationControls", () => {
  it("adds an unselected existing tag directly and can load more tags", async () => {
    const user = userEvent.setup();
    render(<AssetCurationControls assetId="asset-1" csrfToken="csrf-token" />);

    expect(
      screen.queryByRole("button", { name: "添加已有标签 已选择" }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "添加已有标签 旅行" }));

    expect(addManualTag).toHaveBeenCalledWith({
      assetId: "asset-1",
      csrfToken: "csrf-token",
      revision: 7,
      tagId: "tag-travel",
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

  it("can protect an existing AI tag at the 20-tag limit without replacing the full set", async () => {
    mockState(Array.from({ length: 20 }, (_, index) => ({ ...tag(`tag-${index}`, `标签${index}`), source: index === 0 ? "ai" as const : "manual" as const })));
    render(<AssetCurationControls assetId="asset-1" csrfToken="csrf-token" />);
    const protect = screen.getByRole("button", { name: "将标签 标签0 保留为手动标签" });
    expect(protect).toBeEnabled();
    await userEvent.click(protect);
    expect(addManualTag).toHaveBeenCalledExactlyOnceWith({ assetId: "asset-1", csrfToken: "csrf-token", revision: 7, tagId: "tag-0" });
    expect(replaceTags).not.toHaveBeenCalled();
  });

  it("does not carry an unfinished tag input to the next photo", async () => {
    const view = render(<AssetCurationControls assetId="asset-1" csrfToken="token" />);
    await userEvent.type(screen.getByRole("textbox", { name: "新标签名称" }), "仅这张图");
    view.rerender(<AssetCurationControls assetId="asset-2" csrfToken="token" />);
    expect(screen.getByRole("textbox", { name: "新标签名称" })).toHaveValue("");
  });

  it.each([
    {
      action: "protecting an AI tag",
      button: "将标签 flower 保留为手动标签",
      before: [{ ...tag("tag_ai", "flower"), source: "ai" as const }],
      after: [{ ...tag("tag_ai", "flower"), source: "manual" as const }],
      mutation: addManualTag,
    },
    {
      action: "removing a tag",
      button: "移除标签 已选择",
      before: [tag("tag-selected", "已选择")],
      after: [],
      mutation: replaceTags,
    },
    {
      action: "adding an existing tag",
      button: "添加已有标签 旅行",
      before: [tag("tag-selected", "已选择")],
      after: [tag("tag-selected", "已选择"), tag("tag-travel", "旅行")],
      mutation: addManualTag,
    },
  ])("preserves keyboard focus when $action removes the initiating button", async ({ button, before, after, mutation }) => {
    const user = userEvent.setup();
    mockState(before);
    const view = render(<AssetCurationControls assetId="asset-1" csrfToken="token" />);
    const heading = screen.getByRole("heading", { level: 3 });
    const trigger = screen.getByRole("button", { name: button });
    mutation.mockImplementationOnce(() => expect(heading).toHaveFocus());

    trigger.focus();
    await user.keyboard("{Enter}");
    expect(mutation).toHaveBeenCalledOnce();
    expect(heading).toHaveAttribute("tabindex", "-1");
    expect(heading).toHaveFocus();

    mockState(after);
    view.rerender(<AssetCurationControls assetId="asset-1" csrfToken="token" />);
    expect(screen.queryByRole("button", { name: button })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3 })).toBe(heading);
    expect(heading).toHaveFocus();
  });

  it("moves focus before disabling the new-tag field without reclaiming it on completion", async () => {
    const user = userEvent.setup();
    const content = <><AssetCurationControls assetId="asset-1" csrfToken="token" /><button>下一张照片</button></>;
    const view = render(content);
    const heading = screen.getByRole("heading", { level: 3 });
    createTag.mockImplementationOnce(() => expect(heading).toHaveFocus());
    await user.type(screen.getByRole("textbox", { name: "新标签名称" }), "家人{Enter}");
    expect(heading).toHaveFocus();

    vi.mocked(useCreateAndAssignTagMutation).mockReturnValue({ isError: false, isPending: true, mutate: createTag } as never);
    view.rerender(<><AssetCurationControls assetId="asset-1" csrfToken="token" /><button>下一张照片</button></>);
    expect(screen.getByRole("textbox", { name: "新标签名称" })).toBeDisabled();
    expect(heading).toHaveFocus();
    const next = screen.getByRole("button", { name: "下一张照片" });
    next.focus();
    const completed = createTag.mock.calls[0]?.[1] as { onSuccess: () => void };
    act(() => completed.onSuccess());
    expect(next).toHaveFocus();
  });

  it("does not move focus back to the old photo or the new heading after an asset switch", async () => {
    const user = userEvent.setup();
    const view = render(<><AssetCurationControls assetId="asset-1" csrfToken="token" /><button>下一张照片</button></>);
    await user.type(screen.getByRole("textbox", { name: "新标签名称" }), "仅这张图{Enter}");
    const oldHeading = screen.getByRole("heading", { level: 3 });
    expect(oldHeading).toHaveFocus();
    const next = screen.getByRole("button", { name: "下一张照片" });
    next.focus();

    view.rerender(<><AssetCurationControls assetId="asset-2" csrfToken="token" /><button>下一张照片</button></>);
    expect(oldHeading).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3 })).not.toHaveFocus();
    expect(next).toHaveFocus();
    const completed = createTag.mock.calls[0]?.[1] as { onSuccess: () => void };
    act(() => completed.onSuccess());
    expect(next).toHaveFocus();
    expect(screen.getByRole("textbox", { name: "新标签名称" })).toHaveValue("");
  });
});
