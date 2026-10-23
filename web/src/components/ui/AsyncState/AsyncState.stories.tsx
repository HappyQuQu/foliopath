import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../Button/Button";
import {
  EmptyState,
  ErrorState,
  LoadingState,
  OfflineState,
} from "./AsyncState";

const meta = {
  title: "UI/AsyncState",
  component: LoadingState,
  parameters: { layout: "fullscreen" },
  tags: ["autodocs"],
} satisfies Meta<typeof LoadingState>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Loading: Story = {
  args: { label: "正在确认安全状态…" },
};

export const Compact = {
  render: () => <><LoadingState density="compact" label="正在载入任务…" /><ErrorState density="compact" message="任务暂时不可用" onRetry={() => undefined} /><EmptyState density="compact" title="暂无可用任务记录" description="启动处理后显示进度与结果。" /></>,
};

export const Error = {
  render: () => (
    <ErrorState
      message="FolioPath 暂时无法响应。原始媒体没有被修改。"
      onRetry={() => undefined}
    />
  ),
};

export const Empty = {
  render: () => (
    <EmptyState
      action={<Button>包含子目录</Button>}
      description="当前目录没有媒体，但子目录里有 18 项。"
      title="当前目录没有媒体"
    />
  ),
};

export const Offline = {
  render: () => (
    <OfflineState
      description="保留索引中没有可显示的媒体，这不表示原目录为空。请恢复挂载后重试扫描。"
      title="媒体库当前离线"
    />
  ),
};
