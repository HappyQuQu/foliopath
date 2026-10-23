import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "vitest";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { readTagLanguagePreference } from "../../../lib/storage/preferences";
import { TagLanguageControl } from "./TagLanguageControl";

it("uses a labelled native select for a browser-only preference", async () => {
  window.localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale: "zh-CN" }));
  render(<LocaleProvider><TagLanguageControl /></LocaleProvider>);
  const control = screen.getByRole("combobox", { name: "AI 标签显示语言" });
  expect(control).toHaveValue("interface");
  await userEvent.selectOptions(control, "bilingual");
  expect(control).toHaveValue("bilingual");
  expect(readTagLanguagePreference()).toBe("bilingual");
});
