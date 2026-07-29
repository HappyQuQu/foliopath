import { render } from "@testing-library/react";
import { expect, it } from "vitest";

import { BrandMark } from "./BrandMark";

it("renders the canonical decorative brand asset", () => {
  const { container } = render(<BrandMark size="large" />);

  const wrapper = container.querySelector("span");
  const mark = container.querySelector("img");
  expect(wrapper).toHaveAttribute("aria-hidden", "true");
  expect(mark).toHaveAttribute("src", "/foliopath-mark-tree.svg");
  expect(mark).toHaveAttribute("alt", "");
});

it("uses the crisp compact asset at navigation size", () => {
  const { container } = render(<BrandMark size="small" />);

  expect(container.querySelector("img")).toHaveAttribute(
    "src",
    "/foliopath-mark-tree.svg",
  );
  expect(container.querySelectorAll("span")).toHaveLength(1);
});
