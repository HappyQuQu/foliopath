import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { Slider, type SliderProps } from "./Slider";

const meta = {
  title: "UI/Slider",
  component: Slider,
  args: { label: "缩略图大小", min: 120, max: 480, step: 10, value: 210, onChange: () => undefined },
} satisfies Meta<typeof Slider>;
export default meta;
type Story = StoryObj<typeof meta>;
function Controlled(args: SliderProps) {
  const [value, setValue] = useState(args.value);
  return <Slider {...args} value={value} onChange={setValue} />;
}
export const Default: Story = { render: (args) => <Controlled {...args} /> };
