import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { Badge } from "./Badge";
it("keeps evidence labels readable without announcing them as statuses or actions", () => {
  render(<><Badge>文件名</Badge><Badge tone="accent">画面语义</Badge></>);
  expect(screen.getByText("文件名")).toBeVisible();
  expect(screen.getByText("画面语义")).toHaveAttribute("data-tone", "accent");
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
