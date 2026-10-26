import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button, Input, Select } from "../../ui";
import { ContextSidebar } from "./ContextSidebar";

const meta = {
  title: "Patterns/Context sidebar", component: ContextSidebar,
  args: { title: "人物工具", description: "合成示例 · 已建立人物跨媒体库共享。", children: <><label>查找人物<Input type="search" /></label><label>视图<Select><option>已建立人物</option><option>未命名组</option></Select></label><nav aria-label="合成人物"><Button variant="secondary">青岚 <small>4</small></Button><Button variant="quiet">白露 <small>3</small></Button></nav></> },
  parameters: { docs: { description: { component: "Layout pattern only. Real page stories own synthetic contract fixtures and functional URL controls." } } },
} satisfies Meta<typeof ContextSidebar>;
export default meta;
type Story = StoryObj<typeof meta>;
export const PeopleTools: Story = {};

export const CompactFilters: Story = { args: { title: "媒体筛选", density: "compact", description: "" } };
