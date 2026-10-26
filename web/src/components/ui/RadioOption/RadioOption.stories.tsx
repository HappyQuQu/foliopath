import type { Meta, StoryObj } from "@storybook/react-vite";
import { RadioOption } from "./RadioOption";
const meta = { title: "UI/RadioOption", component: RadioOption, args: { children: "当前媒体库", name: "scope", defaultChecked: true } } satisfies Meta<typeof RadioOption>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Selected: Story = {};
export const Disabled: Story = { args: { children: "当前目录", defaultChecked: false, disabled: true } };
