import type { Meta, StoryObj } from "@storybook/react-vite";

import { Select } from "./Select";

const meta = {
  title: "UI/Select",
  component: Select,
  args: {
    "aria-label": "媒体库",
    children: (
      <>
        <option value="all">全部媒体库</option>
        <option value="family">家庭影像</option>
        <option value="archive">摄影归档</option>
      </>
    ),
    defaultValue: "all",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof Select>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const Disabled: Story = { args: { disabled: true } };
export const Invalid: Story = { args: { invalid: true } };
