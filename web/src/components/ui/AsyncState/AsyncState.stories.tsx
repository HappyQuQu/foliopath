import type { Meta, StoryObj } from "@storybook/react-vite";

import { ErrorState, LoadingState } from "./AsyncState";

const meta = {
  title: "UI/AsyncState",
  component: LoadingState,
  tags: ["autodocs"],
} satisfies Meta<typeof LoadingState>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Loading: Story = {
  args: { label: "正在确认安全状态…" },
};

export const Error = {
  render: () => (
    <ErrorState
      message="FolioPath 暂时无法响应。原始媒体没有被修改。"
      onRetry={() => undefined}
    />
  ),
};
