import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";

import { BrandMark } from "./BrandMark";

it("renders the canonical decorative brand asset", () => {
  render(<BrandMark size="large" />);

  const mark = screen.getByRole("presentation");
  expect(mark).toHaveAttribute("src", "/foliopath-mark.svg");
  expect(mark).toHaveAttribute("alt", "");
});
