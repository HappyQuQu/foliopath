import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FormField } from "./FormField";

describe("FormField", () => {
  it("connects label, help and validation error", () => {
    render(
      <FormField
        description="名称必须唯一"
        error="该名称已经存在"
        label="媒体库名称"
        required
      />,
    );

    const input = screen.getByRole("textbox", { name: /媒体库名称/ });
    expect(input).toBeRequired();
    expect(input).toHaveAccessibleDescription("名称必须唯一 该名称已经存在");
    expect(input).toHaveAttribute("aria-invalid", "true");
  });
});
