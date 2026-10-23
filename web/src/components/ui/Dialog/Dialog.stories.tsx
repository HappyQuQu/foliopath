import { useRef, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../Button/Button";
import { Dialog } from "./Dialog";

const meta = {
  title: "UI/Dialog",
  component: Dialog,
  tags: ["autodocs"],
} satisfies Meta<typeof Dialog>;

export default meta;
type Story = StoryObj<typeof meta>;

function DialogExample({ disableTriggerOnConfirm = false }: { disableTriggerOnConfirm?: boolean }) {
  const [open, setOpen] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);
  return (
    <>
      <h2 ref={heading} tabIndex={-1}>任务确认示例</h2>
      <Button disabled={disableTriggerOnConfirm && submitted} onClick={() => setOpen(true)}>打开确认对话框</Button>
      <Dialog
        fallbackFocusRef={heading}
        actions={
          <>
            <Button onClick={() => setOpen(false)}>取消</Button>
            <Button variant="danger" onClick={() => { setOpen(false); setSubmitted(true); }}>
              确认移除
            </Button>
          </>
        }
        description="原始媒体文件不会被删除。"
        onOpenChange={setOpen}
        open={open}
        title="从 FolioPath 移除媒体库？"
      >
        只会移除媒体库配置、索引、任务和可重建缓存。
      </Dialog>
    </>
  );
}

export const Confirmation: Story = {
  args: {
    children: null,
    onOpenChange: () => undefined,
    open: false,
    title: "确认操作",
  },
  render: () => <DialogExample />,
};
export const DisabledTriggerFocus: Story = {
  ...Confirmation,
  render: () => <DialogExample disableTriggerOnConfirm />,
};
