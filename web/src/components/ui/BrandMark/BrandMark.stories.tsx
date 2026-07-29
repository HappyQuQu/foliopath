import type { Meta, StoryObj } from "@storybook/react-vite";

import { BrandMark } from "./BrandMark";

const meta = {
  title: "UI/Brand mark",
  component: BrandMark,
  args: {
    size: "large",
  },
  argTypes: {
    size: {
      control: "inline-radio",
      options: ["small", "medium", "large"],
    },
  },
  tags: ["autodocs"],
} satisfies Meta<typeof BrandMark>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
