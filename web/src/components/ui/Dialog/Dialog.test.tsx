import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "./Dialog";

describe("Dialog", () => {
  it("renders a labelled modal and requests close", async () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog onOpenChange={onOpenChange} open title="确认移除">
        原始媒体不会被删除。
      </Dialog>,
    );

    expect(screen.getByRole("dialog", { name: "确认移除" })).toBeVisible();
    await userEvent.click(screen.getByRole("button", { name: "关闭对话框" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

});
