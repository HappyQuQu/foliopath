import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  MediaCollection,
  MediaCollectionSkeleton,
  type MediaCollectionItem,
} from "./MediaCollection";

const items: MediaCollectionItem[] = Array.from({ length: 80 }, (_, index) => ({
  height: index % 4 === 0 ? 1200 : 800,
  id: `asset-${index}`,
  kind: index % 5 === 0 ? "video" : "image",
  modifiedLabel: `2026-07-${String(28 - (index % 20)).padStart(2, "0")}`,
  name: `family-archive-${String(index + 1).padStart(3, "0")}.jpg`,
  thumbnailStatus: "pending",
  thumbnailUrl: null,
  width: index % 4 === 0 ? 800 : 1200,
}));

const meta = {
  args: {
    hasNextPage: true,
    isFetchingNextPage: false,
    items,
    labels: {
      activatePreview: "预览 {name}",
      animated: "动图",
      failedThumbnail: "缩略图生成失败",
      image: "图片",
      loadMore: "载入更多媒体",
      loadMoreFailed: "更多媒体未能载入，已显示的项目仍然保留。",
      loadingMore: "正在载入更多媒体",
      pendingThumbnail: "正在生成缩略图",
      previewing: "正在预览",
      retryLoadMore: "重试载入更多",
      unavailableThumbnail: "缩略图不可用",
      video: "视频",
    },
    layout: "grid",
    onLoadMore: () => undefined,
  },
  component: MediaCollection,
  parameters: { layout: "fullscreen" },
  title: "Patterns/MediaCollection",
} satisfies Meta<typeof MediaCollection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const AdaptiveGrid: Story = {};

export const Masonry: Story = {
  args: { layout: "masonry" },
};

export const ThumbnailStates: Story = {
  args: {
    hasNextPage: false,
    items: [
      {
        ...items[0]!,
        id: "pending",
        name: "pending.jpg",
        thumbnailStatus: "pending",
      },
      {
        ...items[1]!,
        id: "failed",
        name: "failed.jpg",
        thumbnailStatus: "failed",
      },
      {
        ...items[2]!,
        id: "unavailable",
        name: "unavailable.mp4",
        thumbnailStatus: "unavailable",
      },
    ],
  },
};

export const NextPageFailed: Story = {
  args: {
    hasNextPage: true,
    items: items.slice(0, 12),
    paginationError: true,
  },
};

export const Skeleton = {
  render: () => <MediaCollectionSkeleton label="正在载入媒体…" />,
};
