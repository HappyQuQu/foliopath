import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "./Button";

const meta = {
  title: "UI/Button",
  component: Button,
  args: {
    children: "继续",
  },
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: { variant: "primary" },
};

export const Secondary: Story = {};

export const Quiet: Story = {
  args: { variant: "quiet" },
};

export const Danger: Story = {
  args: { children: "确认移除", variant: "danger" },
};

export const Loading: Story = {
  args: { children: "正在保存", loading: true, variant: "primary" },
};

export const Disabled: Story = {
  args: { disabled: true },
};
