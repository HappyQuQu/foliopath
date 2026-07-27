import type { Meta, StoryObj } from "@storybook/react-vite";

import { MediaViewer } from "./MediaViewer";

const meta = {
  args: {
    canGoNext: true,
    canGoPrevious: true,
    item: {
      contentUrl: "/storybook-preview-photo.jpg",
      details: [
        { label: "位置", value: "旅行/日本/京都/八坂塔.jpg" },
        { label: "修改时间", value: "2026年7月21日 19:12" },
        { label: "尺寸", value: "6,000 × 3,376 px" },
        { label: "大小", value: "20.3 MB" },
      ],
      id: "pagoda",
      kind: "image",
      name: "2026-07-21 19-12-43.jpg",
    },
    labels: {
      close: "关闭",
      exitFullscreen: "退出全屏",
      fit: "适应窗口",
      fullscreen: "全屏",
      imageFailed: "无法显示此图片。",
      info: "显示基本信息",
      information: "基本信息",
      loadFailedDescription: "原始文件没有被修改。",
      next: "下一项",
      originalSize: "按 1:1 显示",
      previous: "上一项",
      retry: "重新检查",
      shortcutHint: "按钮缩放 · 拖动平移 · Esc 退出",
      videoFailed: "无法播放此视频。",
      zoomIn: "放大",
      zoomOut: "缩小",
    },
    onClose: () => undefined,
    onNext: () => undefined,
    onPrevious: () => undefined,
    position: "第 1 / 312 项",
  },
  component: MediaViewer,
  parameters: { layout: "fullscreen" },
  title: "Patterns/MediaViewer",
} satisfies Meta<typeof MediaViewer>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Image: Story = {};

export const Video: Story = {
  args: {
    item: {
      contentUrl: "/storybook-preview-video.mp4",
      details: [
        { label: "位置", value: "旅行/日本/京都/散步.mp4" },
        { label: "尺寸", value: "1,920 × 1,080 px" },
        { label: "时长", value: "0:42" },
      ],
      id: "video",
      kind: "video",
      name: "京都散步.mp4",
      posterUrl: "/storybook-preview-photo.jpg",
    },
  },
};

export const Offline: Story = {
  args: {
    availability: {
      actionLabel: "重新检查",
      description:
        "媒体库挂载当前不可用。FolioPath 保留了上次可靠索引，不会把离线误判为空目录。",
      kind: "offline",
      onAction: () => undefined,
      title: "媒体库当前离线",
    },
  },
};

export const UnsupportedCodec: Story = {
  args: {
    availability: {
      description:
        "此视频容器已被索引，但浏览器无法播放它的编码。MVP 不会转码或修改原视频。",
      kind: "unsupportedCodec",
      title: "浏览器无法播放此视频",
    },
    item: {
      contentUrl: "/storybook-preview-video.mkv",
      details: [
        { label: "位置", value: "旅行/日本/京都/归档视频.mkv" },
        { label: "类型", value: "视频 · video/x-matroska" },
      ],
      id: "unsupported-video",
      kind: "video",
      name: "归档视频.mkv",
      posterUrl: "/storybook-preview-photo.jpg",
    },
  },
};
