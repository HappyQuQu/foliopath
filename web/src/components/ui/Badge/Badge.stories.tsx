import type { Meta, StoryObj } from "@storybook/react-vite";
import { Badge } from "./Badge";
const meta = { title: "UI/Badge", component: Badge, args: { children: "文件名" } } satisfies Meta<typeof Badge>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Neutral: Story = {};
export const Accent: Story = { args: { children: "画面语义", tone: "accent" } };
