import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "vitest";

import { Switch } from "./Switch";

it("exposes checked state through the switch role", async () => {
  const user = userEvent.setup();
  render(<Switch aria-label="固定预览" />);

  const control = screen.getByRole("switch", { name: "固定预览" });
  expect(control).not.toBeChecked();

  await user.click(control);

  expect(control).toBeChecked();
});
