import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "vitest";
import { LocaleProvider } from "./LocaleProvider";
import { displayTagName, useTagDisplay } from "./tagDisplay";
import { readTagLanguagePreference } from "../storage/preferences";
import type { Tag } from "../api/curation";

const tag: Tag = { id: "tag_1", name: "car", assetCount: 2, createdAt: "2026-10-23T00:00:00Z", updatedAt: "2026-10-23T00:00:00Z", localization: { key: "car", zhCN: "汽车", en: "Car" } };
it("formats only server-reviewed metadata without changing manual or legacy text", () => {
  expect(displayTagName(tag, "interface", "zh-CN")).toBe("汽车");
  expect(displayTagName(tag, "interface", "en")).toBe("Car");
  expect(displayTagName(tag, "bilingual", "en")).toBe("汽车 / Car");
  expect(displayTagName({ ...tag, source: "manual" }, "zh-CN", "zh-CN")).toBe("car");
  const { localization: _, ...legacy } = tag;
  expect(displayTagName(legacy, "zh-CN", "zh-CN")).toBe("car");
  expect(tag.name).toBe("car");
});

it("synchronizes mounted consumers and preserves unrelated browser preferences", async () => {
  window.localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale: "zh-CN", theme: "dark" }));
  function Consumer({ control = false }: { control?: boolean }) {
    const { tagName, setLanguage } = useTagDisplay();
    return <><output>{tagName(tag)}</output>{control && <button onClick={() => setLanguage("bilingual")}>Both</button>}</>;
  }
  const view = render(<LocaleProvider><Consumer control /><Consumer /></LocaleProvider>);
  expect(screen.getAllByText("汽车")).toHaveLength(2);
  await userEvent.click(screen.getByText("Both"));
  expect(screen.getAllByText("汽车 / Car")).toHaveLength(2);
  expect(readTagLanguagePreference()).toBe("bilingual");
  expect(JSON.parse(window.localStorage.getItem("foliopath.preferences.v1")!)).toMatchObject({ theme: "dark", locale: "zh-CN" });
  view.unmount();
  render(<LocaleProvider><Consumer /></LocaleProvider>);
  expect(screen.getByText("汽车 / Car")).toBeVisible();
});
