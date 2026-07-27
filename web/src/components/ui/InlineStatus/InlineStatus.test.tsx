import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { InlineStatus } from "./InlineStatus";

describe("InlineStatus", () => {
  it("exposes danger feedback as an alert", () => {
    render(<InlineStatus tone="danger">操作失败</InlineStatus>);
    expect(screen.getByRole("alert")).toHaveTextContent("操作失败");
  });

  it("provides a named dismiss action", async () => {
    const user = userEvent.setup();
    const onDismiss = vi.fn();
    render(<InlineStatus onDismiss={onDismiss}>会话已过期</InlineStatus>);

    await user.click(screen.getByRole("button", { name: "关闭提示" }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
