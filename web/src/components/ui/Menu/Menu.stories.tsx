import type { Meta, StoryObj } from "@storybook/react-vite";
import { useRef, useState } from "react";

import { Button } from "../Button/Button";
import { Menu } from "./Menu";

function Example() {
  const ref = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  return <><Button ref={ref} onClick={() => setOpen(value => !value)}>操作 ▾</Button>
    {open && ref.current && <Menu label="目录操作" heading="合成示例：旅行照片" anchor={ref.current} returnFocusTo={ref.current} onClose={() => setOpen(false)} items={[
      { id: "expand", label: "全部展开", onSelect: () => {} },
      { id: "collapse", label: "全部收起", onSelect: () => {} },
      { id: "unavailable", label: "暂不可用", disabled: true, separatorBefore: true, onSelect: () => {} },
    ]} />}</>;
}

const meta = { title: "UI/Menu", component: Example, parameters: { layout: "centered" }, tags: ["autodocs"] } satisfies Meta<typeof Example>;
export default meta;
type Story = StoryObj<typeof meta>;
export const DirectoryActions: Story = {};
