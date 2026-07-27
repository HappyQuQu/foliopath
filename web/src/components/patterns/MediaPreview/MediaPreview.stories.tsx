import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { MediaPreview } from "./MediaPreview";

const meta = {
  args: {
    canGoNext: true,
    canGoPrevious: true,
    item: {
      contentUrl: "/storybook-preview-photo.jpg",
      details: [
        { label: "类型", value: "图片 · image/jpeg" },
        { label: "位置", value: "旅行/京都/清水寺.jpg" },
        { label: "修改时间", value: "2026年7月28日 10:32" },
        { label: "尺寸", value: "4,032 × 3,024 px" },
        { label: "大小", value: "4.8 MB" },
      ],
      id: "photo",
      kind: "image",
      name: "清水寺.jpg",
    },
    labels: {
      close: "关闭预览",
      imageFailed: "无法显示此图片。",
      next: "下一项",
      position: "第 2 / 8 项",
      previous: "上一项",
      preview: "预览",
      resize: "调整预览宽度",
      videoFailed: "无法播放此视频。",
    },
    onClose: () => undefined,
    onNext: () => undefined,
    onPrevious: () => undefined,
    onWidthChange: () => undefined,
    width: 406,
  },
  component: MediaPreview,
  parameters: { layout: "fullscreen" },
  render: (args) => {
    const [width, setWidth] = useState(406);
    return (
      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        <MediaPreview
          {...args}
          onWidthChange={setWidth}
          width={width}
        />
      </div>
    );
  },
  title: "Patterns/MediaPreview",
} satisfies Meta<typeof MediaPreview>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Image: Story = {};

export const Video: Story = {
  args: {
    item: {
      contentUrl: "/storybook-preview-video.mp4",
      details: [
        { label: "类型", value: "视频 · video/mp4" },
        { label: "位置", value: "旅行/京都/散步.mp4" },
        { label: "尺寸", value: "1,920 × 1,080 px" },
        { label: "时长", value: "0:42" },
      ],
      id: "video",
      kind: "video",
      name: "散步.mp4",
    },
  },
};
