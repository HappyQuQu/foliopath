import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { Bell } from "@phosphor-icons/react";

import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { ToastProvider } from "../../ui/Toast/ToastProvider";
import { IconButton } from "../../ui/Button/IconButton";
import { HeaderAddonProvider } from "../GlobalHeader/HeaderAddonContext";
import { MediaCollection } from "../MediaCollection/MediaCollection";
import { AppShell } from "./AppShell";

const meta = {
  title: "Patterns/AppShell",
  component: AppShell,
  decorators: [
    (Story) => (
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter>
            <Story />
          </MemoryRouter>
        </ToastProvider>
      </ThemeProvider>
    ),
  ],
  parameters: { layout: "fullscreen" },
  args: {
    active: "libraries",
    homeHref: "/",
    identity: "家庭管理员",
    searchHref: "/search",
    settingsHref: "/settings/general",
    title: "媒体库",
    children: <div style={{ padding: "2rem" }}>页面内容</div>,
  },
} satisfies Meta<typeof AppShell>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const ContextualSearch: Story = {
  args: {
    active: "search",
    searchHref: "/libraries/lib_story/search?mode=semantic",
    searchInput: { label: "描述画面内容", placeholder: "描述画面，例如：汽车、海边日落、蓝色头发" },
    children: <p>合成组件状态：调用方提供搜索提示；共享 Header 不解析业务模式。</p>,
  },
};

export const FullHeader: Story = {
  decorators: [(Story) => <HeaderAddonProvider value={<IconButton label="合成通知入口"><Bell aria-hidden="true" size={20} /></IconButton>}><Story /></HeaderAddonProvider>],
  args: {
    peopleHref: "/intelligence?view=clusters",
    identity: "Synthetic administrator",
    children: <p>合成布局状态：窄屏第一行保留完整搜索，第二行排列人物、语言、主题、通知与账户；键盘顺序与视觉一致。</p>,
  },
};

export const LongLibraryName: Story = {
  parameters: {
    docs: { description: { story: "Synthetic maximum-length library name for responsive layout verification." } },
  },
  args: {
    topbarContent: <span>{"Family archive ".padEnd(128, "A")}</span>,
    children: <p>2026</p>,
  },
};

export const ImmersiveSeamless: Story = {
  name: "Scroll-hidden chrome (synthetic media)",
  parameters: { docs: { description: { story: "Production AppShell and MediaCollection with synthetic media. Scroll down to hide the global header and context bar, up to reveal; search input, keyboard navigation and menus retain access. Seamless cards omit favorite and source shortcuts." } } },
  args: {
    autoHideChrome: true,
    topbarContent: <span>合成媒体库 · 滚动体验</span>,
    children: <MediaCollection
      hasNextPage={false}
      isFetchingNextPage={false}
      layout="seamless"
      items={Array.from({ length: 80 }, (_, index) => ({
        id: `synthetic-${index}`, name: `Synthetic ${index + 1}.png`, kind: "image" as const,
        width: 600, height: index % 2 ? 900 : 500, modifiedLabel: "Synthetic fixture",
        thumbnailStatus: "ready" as const,
        thumbnailUrl: `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="600" height="900"><rect width="600" height="900" fill="hsl(${index * 37 % 360} 42% 48%)"/></svg>`)}`,
        sourceHref: "/synthetic", sourceLabel: "合成来源", favorite: index % 3 === 0,
      }))}
      labels={{ activatePreview: "预览 {name}", tileSize: "缩略图大小", animated: "动图", failedThumbnail: "缩略图生成失败", image: "图片", loadMore: "载入更多", loadMoreFailed: "更多媒体未能载入", loadingMore: "正在载入", pendingThumbnail: "正在生成", previewing: "正在预览", retryLoadMore: "重试", unavailableThumbnail: "缩略图不可用", video: "视频" }}
      onLoadMore={() => undefined}
      onItemActivate={() => undefined}
      onFavoriteToggle={() => undefined}
    />,
  },
};
