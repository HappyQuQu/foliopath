import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Button } from "../Button/Button";
import { ToastProvider, useToast } from "./ToastProvider";

function Harness() {
  const toast = useToast();
  return (
    <Button onClick={() => toast.show({ message: "设置已保存", tone: "success" })}>
      显示通知
    </Button>
  );
}

describe("ToastProvider", () => {
  it("announces and dismisses feedback", async () => {
    render(
      <ToastProvider>
        <Harness />
      </ToastProvider>,
    );

    await userEvent.click(screen.getByRole("button", { name: "显示通知" }));
    expect(screen.getByRole("status")).toHaveTextContent("设置已保存");

    await userEvent.click(screen.getByRole("button", { name: "关闭通知" }));
    expect(screen.queryByText("设置已保存")).not.toBeInTheDocument();
  });
});
