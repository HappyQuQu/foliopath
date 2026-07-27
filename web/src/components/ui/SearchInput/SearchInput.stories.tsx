import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { SearchInput } from "./SearchInput";

const meta = {
  component: SearchInput,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  title: "UI/SearchInput",
} satisfies Meta<typeof SearchInput>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: "搜索文件名或路径",
    onChange: () => undefined,
    onSubmit: (event) => event.preventDefault(),
    placeholder: "输入文件名或路径，例如 京都",
    submitLabel: "搜索",
    value: "",
  },
  render: (args) => {
    const [value, setValue] = useState("京都");
    return (
      <div style={{ minWidth: "min(42rem, 90vw)" }}>
        <SearchInput
          {...args}
          onChange={(event) => setValue(event.target.value)}
          value={value}
        />
      </div>
    );
  },
};
