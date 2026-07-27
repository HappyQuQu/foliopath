import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { LocaleProvider, useLocale } from "./LocaleProvider";

function Harness() {
  const { locale, setLocale, t } = useLocale();
  return (
    <>
      <span>{locale}</span>
      <span>{t("account.title")}</span>
      <button onClick={() => setLocale("en")} type="button">
        English
      </button>
    </>
  );
}

describe("LocaleProvider", () => {
  it("follows a Chinese browser and persists an explicit English choice", async () => {
    const languages = vi.spyOn(window.navigator, "languages", "get").mockReturnValue(["zh-CN"]);

    render(
      <LocaleProvider>
        <Harness />
      </LocaleProvider>,
    );

    expect(screen.getByText("zh-CN")).toBeVisible();
    expect(screen.getByText("通用设置")).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "English" }));
    expect(screen.getByText("General settings")).toBeVisible();
    expect(document.documentElement).toHaveAttribute("lang", "en");
    expect(window.localStorage.getItem("foliopath.preferences.v1")).toContain('"locale":"en"');
    languages.mockRestore();
  });
});
