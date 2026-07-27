import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Button } from "../Button/Button";
import {
  TOAST_AUTO_DISMISS_MS,
  ToastProvider,
  useToast,
} from "./ToastProvider";

function Harness() {
  const toast = useToast();
  return (
    <Button onClick={() => toast.show({ message: "设置已保存", tone: "success" })}>
      显示通知
    </Button>
  );
}

describe("ToastProvider", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("announces and dismisses feedback", async () => {
    render(
      <ToastProvider>
        <Harness />
      </ToastProvider>,
    );

    expect(screen.getByRole("region", { name: "通知" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "显示通知" }));
    expect(screen.getByRole("status")).toHaveTextContent("设置已保存");

    await userEvent.click(screen.getByRole("button", { name: "关闭通知" }));
    expect(screen.queryByText("设置已保存")).not.toBeInTheDocument();
  });

  it("automatically clears feedback so it cannot obstruct later actions", () => {
    vi.useFakeTimers();
    render(
      <ToastProvider>
        <Harness />
      </ToastProvider>,
    );

    act(() => {
      screen.getByRole("button", { name: "显示通知" }).click();
    });
    expect(screen.getByRole("status")).toHaveTextContent("设置已保存");

    act(() => {
      vi.advanceTimersByTime(TOAST_AUTO_DISMISS_MS);
    });
    expect(screen.queryByText("设置已保存")).not.toBeInTheDocument();
  });
});
