import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";

import { Select } from "./Select";

it("keeps native select semantics and reports value changes", async () => {
  const user = userEvent.setup();
  const onChange = vi.fn();
  render(
    <label>
      媒体库
      <Select defaultValue="all" onChange={onChange}>
        <option value="all">全部媒体库</option>
        <option value="family">家庭影像</option>
      </Select>
    </label>,
  );

  const select = screen.getByRole("combobox", { name: "媒体库" });
  await user.selectOptions(select, "family");

  expect(select).toHaveValue("family");
  expect(onChange).toHaveBeenCalledOnce();
});

it("exposes invalid and disabled states on the native control", () => {
  render(
    <Select aria-label="失败原因" disabled invalid>
      <option>全部原因</option>
    </Select>,
  );

  expect(screen.getByRole("combobox", { name: "失败原因" })).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "失败原因" })).toHaveAttribute(
    "aria-invalid",
    "true",
  );
});
