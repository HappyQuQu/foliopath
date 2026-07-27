import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../Button/Button";
import { ToastProvider, useToast } from "./ToastProvider";

const meta = {
  title: "UI/Toast",
  component: ToastProvider,
  tags: ["autodocs"],
} satisfies Meta<typeof ToastProvider>;

export default meta;
type Story = StoryObj<typeof meta>;

function ToastExample() {
  const toast = useToast();
  return (
    <div style={{ display: "flex", gap: "var(--space-2)" }}>
      <Button onClick={() => toast.show({ message: "设置已保存", tone: "success" })}>
        成功通知
      </Button>
      <Button
        variant="danger"
        onClick={() => toast.show({ message: "暂时无法保存", tone: "danger" })}
      >
        错误通知
      </Button>
    </div>
  );
}

export const Tones: Story = {
  args: { children: null },
  render: () => (
    <ToastProvider>
      <ToastExample />
    </ToastProvider>
  ),
};
