import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { Button } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { AIProcessingRow } from "./AIProcessingRow";

it("exposes maintenance by keyboard without triggering processing or losing the disclosure focus", async () => {
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  const process = vi.fn();
  const user = userEvent.setup();
  render(<LocaleProvider><AIProcessingRow title="人物分析" description="处理当前库" icon={null} action={<Button onClick={process}>补齐缺失</Button>} maintenance={<Button>全部重建</Button>}><p>无当前进度记录</p></AIProcessingRow></LocaleProvider>);
  expect(screen.getByRole("region", { name: "人物分析" })).toBeVisible();
  const more = screen.getByRole("button", { name: "更多：人物分析" });
  await user.tab(); await user.tab();
  expect(more).toHaveFocus();
  await user.keyboard("{Enter}");
  expect(more).toHaveAttribute("aria-expanded", "true");
  expect(screen.getByRole("button", { name: "全部重建" })).toBeVisible();
  expect(process).not.toHaveBeenCalled();
  await user.keyboard(" ");
  expect(more).toHaveAttribute("aria-expanded", "false");
  expect(more).toHaveFocus();
  expect(screen.queryByRole("button", { name: "全部重建" })).not.toBeInTheDocument();
});
