import type { Meta, StoryObj } from "@storybook/react-vite";

import { InlineStatus } from "./InlineStatus";

const meta = {
  title: "UI/InlineStatus",
  component: InlineStatus,
  args: {
    children: "为了保护您的媒体库，会话已过期。请重新登录。",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof InlineStatus>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Information: Story = {};

export const Dismissible: Story = {
  args: {
    onDismiss: () => undefined,
  },
};

export const Danger: Story = {
  args: {
    children: "暂时无法完成操作，请稍后重试。",
    tone: "danger",
  },
};
