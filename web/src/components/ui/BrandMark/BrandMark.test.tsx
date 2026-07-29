import { render } from "@testing-library/react";
import { expect, it } from "vitest";

import { BrandMark } from "./BrandMark";

it("renders the canonical decorative brand asset", () => {
  const { container } = render(<BrandMark size="large" />);

  const mark = container.querySelector("img");
  expect(mark).toHaveAttribute("src", "/foliopath-mark.svg");
  expect(mark).toHaveAttribute("alt", "");
  expect(mark).toHaveAttribute("aria-hidden", "true");
});
