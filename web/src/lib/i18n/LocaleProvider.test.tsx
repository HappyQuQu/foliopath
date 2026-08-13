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
      <span>{t("logs.attempts").replace("{count}", "2")}</span>
      <span>{t("diagnostics.reason.decode_failed")}</span>
      <span>{t("curation.availableTags")}</span>
      <span>{t("curation.addExistingTag").replace("{name}", "旅行")}</span>
      <span>{t("curation.newTagHint")}</span>
      <span>{t("curation.createTag")}</span>
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
    expect(screen.getByText("本轮已尝试 2 次")).toBeVisible();
    expect(screen.getByText("解码时发现截断或损坏的媒体数据")).toBeVisible();
    expect(screen.getByText("可添加的已有标签")).toBeVisible();
    expect(screen.getByText("添加已有标签 旅行")).toBeVisible();
    expect(screen.getByText("输入仅用于新建标签；已有标签请直接选择。")).toBeVisible();
    expect(screen.getByText("新建标签")).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "English" }));
    expect(screen.getByText("General settings")).toBeVisible();
    expect(screen.getByText("2 attempts in this run")).toBeVisible();
    expect(
      screen.getByText(
        "Truncated or damaged media data was found while decoding",
      ),
    ).toBeVisible();
    expect(screen.getByText("Existing tags available to add")).toBeVisible();
    expect(screen.getByText("Add existing tag 旅行")).toBeVisible();
    expect(
      screen.getByText(
        "Use this field only to create a new tag; select an existing tag directly.",
      ),
    ).toBeVisible();
    expect(screen.getByText("Create tag")).toBeVisible();
    expect(document.documentElement).toHaveAttribute("lang", "en");
    expect(window.localStorage.getItem("foliopath.preferences.v1")).toContain('"locale":"en"');
    languages.mockRestore();
  });
});
