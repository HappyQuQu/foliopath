import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Button } from "./Button";

describe("Button", () => {
  it("invokes its action", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>继续</Button>);

    await userEvent.click(screen.getByRole("button", { name: "继续" }));

    expect(onClick).toHaveBeenCalledOnce();
  });

  it("prevents interaction while loading", async () => {
    const onClick = vi.fn();
    render(
      <Button loading onClick={onClick}>
        正在保存
      </Button>,
    );

    const button = screen.getByRole("button", { name: "正在保存" });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
    await userEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });
});
