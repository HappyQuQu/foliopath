import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { useRef, useState } from "react";

import { Dialog } from "./Dialog";

describe("Dialog", () => {
  it.each(["disabled", "removed", "available"])("restores focus with a stable fallback when the trigger is %s", async state => {
    function Example() {
      const [open, setOpen] = useState(false);
      const [submitted, setSubmitted] = useState(false);
      const heading = useRef<HTMLHeadingElement>(null);
      return <><h2 ref={heading} tabIndex={-1}>任务</h2>
        {!(submitted && state === "removed") && <button disabled={submitted && state === "disabled"} onClick={() => setOpen(true)}>重跑</button>}
        <Dialog open={open} onOpenChange={setOpen} fallbackFocusRef={heading} title="确认任务" actions={<button onClick={() => { setSubmitted(true); setOpen(false); }}>开始</button>}>说明</Dialog>
      </>;
    }
    render(<Example />);
    await userEvent.click(screen.getByRole("button", { name: "重跑" }));
    await userEvent.click(screen.getByRole("button", { name: "开始" }));
    expect(state === "available" ? screen.getByRole("button", { name: "重跑" }) : screen.getByRole("heading", { name: "任务" })).toHaveFocus();
  });
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
