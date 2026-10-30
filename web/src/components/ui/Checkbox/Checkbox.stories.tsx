import type { Meta, StoryObj } from "@storybook/react-vite";
import { Checkbox } from "./Checkbox";
const meta = { title: "UI/Checkbox", component: Checkbox, args: { "aria-label": "选择合成相册" }, parameters: { layout: "centered" } } satisfies Meta<typeof Checkbox>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Unchecked: Story = {};
export const Checked: Story = { args: { defaultChecked: true } };
export const Disabled: Story = { args: { disabled: true } };
