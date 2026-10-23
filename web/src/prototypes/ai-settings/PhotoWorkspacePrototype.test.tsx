import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ToastProvider } from "../../components/ui";
import { LocaleProvider } from "../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { PhotoWorkspacePrototype, countTags } from "./PhotoWorkspacePrototype";
const network = vi.fn(() => { throw new Error("No network allowed in a disposable prototype"); });
beforeEach(() => {
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  network.mockClear(); vi.stubGlobal("fetch", network);
  vi.spyOn(XMLHttpRequest.prototype, "open").mockImplementation(network);
});
afterEach(() => { expect(network).not.toHaveBeenCalled(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
function preview(initialView: "tags" | "search" = "tags") {
  return render(<LocaleProvider><ThemeProvider><ToastProvider><MemoryRouter><PhotoWorkspacePrototype initialView={initialView} /></MemoryRouter></ToastProvider></ThemeProvider></LocaleProvider>);
}
it("derives distinct effective tag counts, excluding suppressed AI but preserving manual tags", () => {
  expect(countTags([{id:"1",src:"",name:"",path:"",ai:["A","A","B"],manual:["A","C"],excluded:["B"]}])).toEqual([["A",1],["C",1]]);
});
it("opens a bounded tag cloud with automatic tags and no suggestion approval flow", () => {
  preview();
  expect(screen.getByRole("heading", { name:"用标签发现照片" })).toBeVisible();
  expect(screen.getByRole("button", {name:"人像 · 9 张图片"})).toBeVisible();
  expect(screen.getByRole("button", {name:"绿色 · 3 张图片"})).toBeVisible();
  expect(screen.getAllByRole("button", {name:/^打开图片/})).toHaveLength(9);
  expect(screen.queryByText("候选标签")).not.toBeInTheDocument();
  expect(screen.queryByText("标签建议")).not.toBeInTheDocument();
});
it("filters by a cloud tag and clears the filter", async () => {
  const user = userEvent.setup(); preview();
  await user.click(screen.getByRole("button", {name:"绿色 · 3 张图片"}));
  expect(screen.getAllByRole("button", {name:/^打开图片/})).toHaveLength(3);
  expect(screen.getByRole("button", {name:"绿色 · 3 张图片"})).toHaveAttribute("aria-pressed","true");
  await user.click(screen.getByRole("button", {name:"清除标签筛选"}));
  expect(screen.getAllByRole("button", {name:/^打开图片/})).toHaveLength(9);
});
it("edits tags, updates counts, suppresses deleted AI tags on re-analysis and deduplicates", async () => {
  const user = userEvent.setup(); preview();
  await user.click(screen.getByRole("button", {name:"打开图片 IMG_4001.png"}));
  await user.click(screen.getByRole("button", {name:"删除标签 蓝色"}));
  expect(screen.getByRole("button", {name:"蓝色 · 2 张图片"})).toBeVisible();
  await user.type(screen.getByLabelText("添加标签"), " 夏日 ");
  await user.click(screen.getByRole("button", {name:"添加"}));
  await user.click(screen.getByRole("button", {name:"模拟重新分析"}));
  expect(screen.queryByRole("button", {name:"删除标签 蓝色"})).not.toBeInTheDocument();
  expect(screen.getByRole("button", {name:"删除标签 夏日"})).toBeVisible();
  expect(screen.getByRole("button", {name:"删除标签 精选"})).toBeVisible();
  await user.type(screen.getByLabelText("添加标签"), "人像");
  await user.click(screen.getByRole("button", {name:"添加"}));
  expect(screen.getByText("这张图片已有该标签，不会重复添加。")).toBeVisible();
  expect(screen.getByRole("button", {name:"人像 · 9 张图片"})).toBeVisible();
  await user.clear(screen.getByLabelText("添加标签"));
  await user.type(screen.getByLabelText("添加标签"), "蓝色");
  await user.click(screen.getByRole("button", {name:"添加"}));
  expect(screen.getByRole("button", {name:"蓝色 · 3 张图片"})).toBeVisible();
});
it("scopes counts and editing to the selected demo library", async () => {
  const user = userEvent.setup(); preview();
  await user.selectOptions(screen.getByRole("combobox", {name:"媒体库"}), "trip");
  expect(screen.getByRole("button", {name:"人像 · 3 张图片"})).toBeVisible();
  expect(screen.queryByRole("button", {name:/蓝色 ·/})).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", {name:"打开图片 绿色_温室_01.png"}));
  await user.click(screen.getByRole("button", {name:"删除标签 绿色"}));
  await user.selectOptions(screen.getByRole("combobox", {name:"媒体库"}), "sample");
  expect(screen.getByRole("button", {name:"绿色 · 3 张图片"})).toBeVisible();
});
it("labels filename, path and semantic matches separately and does not fabricate OCR", async () => {
  const user = userEvent.setup(); preview("search");
  expect(screen.getAllByRole("button", {name:/^打开图片/})).toHaveLength(3);
  const cards = screen.getAllByRole("article");
  expect(within(cards[0]!).getByText("文件名")).toBeVisible();
  expect(within(cards[0]!).getByText("画面语义")).toBeVisible();
  expect(within(cards[1]!).getByText("库内路径")).toBeVisible();
  expect(within(cards[2]!).queryByText("文件名")).not.toBeInTheDocument();
  expect(screen.getByRole("button", {name:"图片文字 OCR · 未接入"})).toBeDisabled();
  await user.click(screen.getByRole("button", {name:"文件名 / 路径"}));
  expect(screen.getAllByRole("button", {name:/^打开图片/})).toHaveLength(2);
  await user.click(screen.getByRole("button", {name:"打开图片 绿色_温室_01.png"}));
  expect(screen.getByRole("heading", {name:"为什么匹配"})).toBeVisible();
  expect(screen.getByText(/固定演示，非实际模型解释/)).toBeVisible();
});
it("supports searching and finding an honest empty state in the finite fixture", async () => {
  const user = userEvent.setup(); preview("search");
  await user.clear(screen.getByRole("textbox", {name:"搜索内容"}));
  await user.type(screen.getByRole("textbox", {name:"搜索内容"}), "不存在的车{Enter}");
  expect(screen.getByText("没有匹配的图片")).toBeVisible();
  expect(screen.queryByRole("button", {name:/^打开图片/})).not.toBeInTheDocument();
});
