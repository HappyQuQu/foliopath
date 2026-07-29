import type { Meta, StoryObj } from "@storybook/react-vite";
import { useRef } from "react";

import { Button } from "../../ui";
import {
  MediaCollection,
  MediaCollectionSkeleton,
  mediaCollectionCapacityBudget,
  type MediaCollectionHandle,
  type MediaCollectionItem,
} from "./MediaCollection";

const labels = {
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
};

const storyboardSprite = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`
  <svg xmlns="http://www.w3.org/2000/svg" width="1600" height="360">
    <rect width="1600" height="360" fill="#182235"/>
    ${Array.from({ length: 10 }, (_, index) => {
      const x = (index % 5) * 320;
      const y = Math.floor(index / 5) * 180;
      const hue = 205 + index * 7;
      return `<g transform="translate(${x} ${y})">
        <rect width="320" height="180" fill="hsl(${hue} 55% 35%)"/>
        <circle cx="${60 + index * 12}" cy="82" r="32" fill="hsl(${hue + 35} 70% 68%)"/>
        <text x="24" y="156" fill="white" font-family="sans-serif" font-size="28">${index + 1}</text>
      </g>`;
    }).join("")}
  </svg>
`)}`;

const portraitStoryboardSprite = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`
  <svg xmlns="http://www.w3.org/2000/svg" width="720" height="320">
    ${Array.from({ length: 4 }, (_, index) => `
      <g transform="translate(${index * 180} 0)">
        <rect width="180" height="320" fill="hsl(${160 + index * 18} 48% 32%)"/>
        <rect x="28" y="${45 + index * 28}" width="124" height="92" rx="18" fill="hsl(${205 + index * 12} 68% 70%)"/>
        <text x="20" y="292" fill="white" font-family="sans-serif" font-size="28">${index + 1}</text>
      </g>
    `).join("")}
  </svg>
`)}`;

const items: MediaCollectionItem[] = Array.from({ length: 80 }, (_, index) => {
  const video = index % 5 === 0;
  return {
    height: index % 4 === 0 ? 1200 : 800,
    id: `asset-${index}`,
    kind: video ? "video" : "image",
    modifiedLabel: `2026-07-${String(28 - (index % 20)).padStart(2, "0")}`,
    name: `family-archive-${String(index + 1).padStart(3, "0")}.${
      video ? "mp4" : "jpg"
    }`,
    ...(video
      ? {
          storyboard: {
            cellHeight: 180,
            cellWidth: 320,
            columns: 5,
            frameCount: 10,
            rows: 2,
            url: storyboardSprite,
          },
        }
      : {}),
    thumbnailStatus: video ? "ready" : "pending",
    thumbnailUrl: video ? "/api/v1/assets/example/thumbnail" : null,
    width: index % 4 === 0 ? 800 : 1200,
  };
});

const capacityItems: MediaCollectionItem[] = Array.from(
  { length: mediaCollectionCapacityBudget.primaryTierItems },
  (_, index) => ({
    height: index % 4 === 0 ? 1200 : 800,
    id: `capacity-${index}`,
    kind: index % 11 === 0 ? "video" : "image",
    modifiedLabel: "2026-07-28",
    name: `capacity-${String(index + 1).padStart(6, "0")}.jpg`,
    thumbnailStatus: "pending",
    thumbnailUrl: null,
    width: index % 4 === 0 ? 800 : 1200,
  }),
);

const storyboardCapacityItems: MediaCollectionItem[] = Array.from(
  { length: 100 },
  (_, index) => ({
    height: 1080,
    id: `storyboard-capacity-${index}`,
    kind: "video",
    modifiedLabel: "2026-07-29",
    name: `storyboard-capacity-${String(index + 1).padStart(3, "0")}.mp4`,
    storyboard: {
      cellHeight: 180,
      cellWidth: 320,
      columns: 5,
      frameCount: 10,
      rows: 2,
      url: storyboardSprite,
    },
    thumbnailStatus: "ready",
    thumbnailUrl: storyboardSprite,
    width: 1920,
  }),
);

const meta = {
  args: {
    hasNextPage: true,
    isFetchingNextPage: false,
    items,
    labels,
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

export const StoryboardStates: Story = {
  args: {
    hasNextPage: false,
    items: [
      {
        ...items[0]!,
        id: "storyboard-ready-10",
        name: "ready-landscape-10-frames.mp4",
        storyboard: {
          cellHeight: 180,
          cellWidth: 320,
          columns: 5,
          frameCount: 10,
          rows: 2,
          url: storyboardSprite,
        },
      },
      {
        ...items[5]!,
        height: 1920,
        id: "storyboard-ready-4",
        name: "ready-portrait-4-frames.mp4",
        storyboard: {
          cellHeight: 320,
          cellWidth: 180,
          columns: 4,
          frameCount: 4,
          rows: 1,
          url: portraitStoryboardSprite,
        },
        width: 1080,
      },
      {
        ...withoutStoryboard(items[10]!),
        id: "storyboard-pending",
        name: "pending-silently-uses-poster.mp4",
      },
      {
        ...withoutStoryboard(items[15]!),
        id: "storyboard-failed",
        name: "failed-silently-uses-poster-with-a-very-long-filename.mp4",
      },
    ],
  },
};

export const StoryboardMasonry: Story = {
  args: {
    hasNextPage: false,
    items: [
      {
        ...items[0]!,
        height: 1080,
        id: "storyboard-masonry-landscape",
        name: "masonry-landscape.mp4",
        width: 1920,
      },
      {
        ...items[5]!,
        height: 1920,
        id: "storyboard-masonry-portrait",
        name: "masonry-portrait.mp4",
        storyboard: {
          cellHeight: 320,
          cellWidth: 180,
          columns: 4,
          frameCount: 4,
          rows: 1,
          url: portraitStoryboardSprite,
        },
        width: 1080,
      },
    ],
    layout: "masonry",
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

export const Capacity100k = {
  parameters: {
    controls: { disable: true },
  },
  render: () => <CapacityTier />,
};

export const StoryboardCapacity100: Story = {
  args: {
    hasNextPage: false,
    items: storyboardCapacityItems,
  },
  parameters: {
    controls: { disable: true },
  },
};

function CapacityTier() {
  const collectionRef = useRef<MediaCollectionHandle>(null);

  return (
    <>
      <Button
        onClick={() =>
          collectionRef.current?.restoreItem(
            `capacity-${mediaCollectionCapacityBudget.primaryTierItems - 1}`,
          )
        }
      >
        恢复最后一项焦点
      </Button>
      <MediaCollection
        ref={collectionRef}
        hasNextPage={false}
        isFetchingNextPage={false}
        items={capacityItems}
        labels={labels}
        layout="grid"
        onItemActivate={() => undefined}
        onLoadMore={() => undefined}
      />
    </>
  );
}

function withoutStoryboard(
  item: MediaCollectionItem,
): MediaCollectionItem {
  const { storyboard: _storyboard, ...fallback } = item;
  return fallback;
}
