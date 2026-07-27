import type { Meta, StoryObj } from "@storybook/react-vite";

import { FormField } from "./FormField";

const meta = {
  title: "UI/FormField",
  component: FormField,
  args: {
    description: "名称需要在此 FolioPath 实例中保持唯一。",
    label: "媒体库名称",
    placeholder: "例如：家庭影像",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof FormField>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Required: Story = {
  args: { required: true },
};

export const Invalid: Story = {
  args: {
    defaultValue: "家庭影像",
    error: "该名称已经存在。",
    required: true,
  },
};

export const Disabled: Story = {
  args: { defaultValue: "摄影归档", disabled: true },
};
