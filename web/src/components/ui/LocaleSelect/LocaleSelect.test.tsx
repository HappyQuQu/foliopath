import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { LocaleSelect } from "./LocaleSelect";

describe("LocaleSelect", () => {
  it("changes the document language with a semantic select", async () => {
    window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
    render(
      <LocaleProvider>
        <LocaleSelect />
      </LocaleProvider>,
    );

    await userEvent.selectOptions(screen.getByRole("combobox", { name: "语言" }), "en");

    expect(document.documentElement).toHaveAttribute("lang", "en");
    expect(screen.getByRole("combobox", { name: "Language" })).toHaveValue("en");
  });
});
