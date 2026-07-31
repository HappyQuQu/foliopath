import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { LocaleToggle } from "./LocaleToggle";

describe("LocaleToggle", () => {
  it("switches locale immediately and persists the explicit preference", async () => {
    window.localStorage.setItem(
      "foliopath.preferences.v1",
      '{"locale":"zh-CN"}',
    );
    const user = userEvent.setup();

    render(
      <LocaleProvider>
        <LocaleToggle />
      </LocaleProvider>,
    );

    await user.click(screen.getByRole("button", { name: "切换到 English" }));

    expect(document.documentElement).toHaveAttribute("lang", "en");
    expect(
      screen.getByRole("button", { name: "Switch to 简体中文" }),
    ).toBeVisible();
    expect(window.localStorage.getItem("foliopath.preferences.v1")).toBe(
      '{"locale":"en"}',
    );
  });
});
