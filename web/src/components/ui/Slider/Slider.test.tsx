import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";

import { Slider } from "./Slider";

it("labels a focusable native range and reports numeric changes", async () => {
  const onChange = vi.fn();
  render(<Slider label="缩略图大小" min={120} max={480} step={10} value={210} onChange={onChange} />);
  const slider = screen.getByRole("slider", { name: "缩略图大小" });
  await userEvent.tab();
  expect(slider).toHaveFocus();
  expect(slider).toHaveAttribute("min", "120");
  expect(slider).toHaveAttribute("max", "480");
  fireEvent.change(slider, { target: { value: "350" } });
  expect(onChange).toHaveBeenCalledWith(350);
});
