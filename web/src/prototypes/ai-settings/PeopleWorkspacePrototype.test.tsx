import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../components/ui";
import { LocaleProvider } from "../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { PeopleWorkspacePrototype } from "./PeopleWorkspacePrototype";

const fetchSpy = vi.fn(() => { throw new Error("The prototype must not call an API"); });
const xhrSpy = vi.fn();

beforeEach(() => {
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  fetchSpy.mockClear(); xhrSpy.mockClear();
  vi.stubGlobal("fetch", fetchSpy);
  vi.spyOn(XMLHttpRequest.prototype, "open").mockImplementation(() => { xhrSpy(); throw new Error("The prototype must not call an API"); });
});

afterEach(() => {
  expect(fetchSpy).not.toHaveBeenCalled();
  expect(xhrSpy).not.toHaveBeenCalled();
  vi.restoreAllMocks(); vi.unstubAllGlobals();
});

it("renders bounded synthetic portraits and leaves unsupported actions out of the UI", () => {
  renderPeople();
  expect(screen.getByRole("heading", { level: 1, name: "人物" })).toHaveFocus();
  expect(screen.getAllByRole("button", { name: /^查看/ })).toHaveLength(4);
  expect(screen.getAllByRole("img", { name: /合成人物头像/ })).toHaveLength(4);
  expect(screen.getByText("合成原型数据")).toBeVisible();
  expect(screen.getByText(/共 9 张 AI 合成的虚构成年人物照片/)).toBeVisible();
  expect(screen.getByText(/人物目录适用于全部媒体库/)).toBeVisible();
  expect(screen.queryByRole("button", { name: /隐藏|封面/ })).not.toBeInTheDocument();
});

it("filters unnamed people and opens that person's bounded photo collection", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: /^未命名2$/ }));
  expect(screen.getAllByRole("button", { name: /^查看/ })).toHaveLength(2);
  expect(screen.queryByRole("button", { name: /查看林夏/ })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" }));
  expect(screen.getByRole("heading", { level: 1, name: "未命名人物 1" })).toHaveFocus();
  expect(screen.getByRole("img", { name: "合成成年人物照片 azure-03.png" })).toBeVisible();
  expect(screen.getAllByRole("checkbox")).toHaveLength(1);
  expect(screen.getByRole("button", { name: "移出人物" })).toBeDisabled();
});

it("saves a trimmed name explicitly and updates the unnamed filter", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: /^未命名2$/ }));
  await user.click(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" }));
  const name = screen.getByRole("textbox", { name: "人物名称" });
  expect(screen.getByRole("button", { name: "保存名称" })).toBeDisabled();
  await user.type(name, "  ");
  expect(screen.getByRole("button", { name: "保存名称" })).toBeDisabled();
  await user.clear(name); await user.type(name, "  测试人物  ");
  expect(screen.getByRole("heading", { level: 1, name: "未命名人物 1" })).toBeVisible();
  await user.keyboard("{Enter}");
  expect(screen.getByRole("heading", { level: 1, name: "测试人物" })).toBeVisible();
  expect(screen.getByRole("textbox", { name: "人物名称" })).toHaveValue("测试人物");
  expect(screen.getByRole("button", { name: "保存名称" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "返回人物" }));
  expect(screen.getAllByRole("button", { name: /^查看/ })).toHaveLength(1);
  await user.click(screen.getByRole("button", { name: /^全部人物4$/ }));
  expect(screen.getByRole("button", { name: "查看测试人物，1 张照片" })).toBeVisible();
});

it("does not merge people just because they are given the same name", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" }));
  await user.type(screen.getByRole("textbox", { name: "人物名称" }), "林夏");
  await user.click(screen.getByRole("button", { name: "保存名称" }));
  await user.click(screen.getByRole("button", { name: "返回人物" }));
  expect(screen.getAllByRole("button", { name: /^查看/ })).toHaveLength(4);
  expect(screen.getByRole("button", { name: "查看林夏，1 张照片" })).toBeVisible();
  expect(screen.getByRole("button", { name: "查看林夏，3 张照片" })).toBeVisible();
});

it("requires an explicit different merge target and cancelling changes nothing", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" }));
  await user.click(screen.getByRole("button", { name: "合并人物" }));
  const dialog = screen.getByRole("dialog", { name: "合并人物" });
  expect(within(dialog).getByRole("button", { name: "确认合并" })).toBeDisabled();
  expect(within(dialog).queryByRole("option", { name: /未命名人物 1/ })).not.toBeInTheDocument();
  await user.selectOptions(within(dialog).getByRole("combobox", { name: "合并到哪位人物" }), "azure");
  expect(within(dialog).getByText(/保留「林夏」的名称，合计 4 张照片/)).toBeVisible();
  await user.click(within(dialog).getByRole("button", { name: "取消" }));
  expect(screen.getByRole("heading", { level: 1, name: "未命名人物 1" })).toBeVisible();
  expect(screen.getAllByRole("checkbox")).toHaveLength(1);
  await user.click(screen.getByRole("button", { name: "返回人物" }));
  expect(screen.getAllByRole("button", { name: /^查看/ })).toHaveLength(4);
});

it("merges only after confirmation, keeps the target name, and preserves all photos", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" }));
  await user.click(screen.getByRole("button", { name: "合并人物" }));
  const dialog = screen.getByRole("dialog", { name: "合并人物" });
  await user.selectOptions(within(dialog).getByRole("combobox", { name: "合并到哪位人物" }), "azure");
  await user.click(within(dialog).getByRole("button", { name: "确认合并" }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(screen.getByRole("heading", { level: 1, name: "林夏" })).toHaveFocus();
  expect(screen.getByRole("textbox", { name: "人物名称" })).toHaveValue("林夏");
  expect(screen.getAllByRole("checkbox")).toHaveLength(4);
  expect(screen.getByRole("img", { name: "合成成年人物照片 azure-03.png" })).toBeVisible();
  expect(screen.getByText("演示人物已合并，所有照片仍保留。")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "返回人物" }));
  expect(screen.getAllByRole("button", { name: /^查看/ })).toHaveLength(3);
  expect(screen.getByRole("button", { name: "查看林夏，4 张照片" })).toBeVisible();
  expect(screen.getByText(/共 9 张 AI 合成/)).toBeVisible();
});

it("retains selected photos when removal is cancelled", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: "查看林夏，3 张照片" }));
  await user.click(screen.getByRole("checkbox", { name: "选择 crimson-03.png" }));
  expect(screen.getByText("已选 1 张")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "移出人物" }));
  const dialog = screen.getByRole("dialog", { name: "移出选中的照片？" });
  expect(within(dialog).getByText("只解除人物关联，不删除照片，不改动原始文件。")).toBeVisible();
  await user.click(within(dialog).getByRole("button", { name: "取消" }));
  expect(screen.getAllByRole("checkbox")).toHaveLength(3);
  expect(screen.getByRole("checkbox", { name: "选择 crimson-03.png" })).toBeChecked();
});

it("removes a wrong association while keeping the photo available in the prototype", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: "查看林夏，3 张照片" }));
  await user.click(screen.getByRole("checkbox", { name: "选择 crimson-03.png" }));
  await user.click(screen.getByRole("button", { name: "移出人物" }));
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "确认移出" }));
  expect(screen.getAllByRole("checkbox")).toHaveLength(2);
  expect(screen.queryByRole("checkbox", { name: "选择 crimson-03.png" })).not.toBeInTheDocument();
  expect(screen.getByText(/已移出 1 张照片。照片仍保留在媒体库/)).toBeVisible();
  expect(screen.getByRole("button", { name: "移出人物" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "返回人物" }));
  expect(screen.getByRole("button", { name: "查看林夏，2 张照片" })).toBeVisible();
  expect(screen.getByRole("heading", { name: "已移出的照片 · 1 张照片" })).toBeVisible();
  expect(screen.getByRole("img", { name: "已移出的合成照片 crimson-03.png" })).toBeVisible();
  expect(screen.getByText(/共 9 张 AI 合成/)).toBeVisible();
});

it("keeps a named person after all their associations are removed", async () => {
  const user = userEvent.setup(); renderPeople();
  await user.click(screen.getByRole("button", { name: "查看明月，2 张照片" }));
  for (const checkbox of screen.getAllByRole("checkbox")) await user.click(checkbox);
  await user.click(screen.getByRole("button", { name: "移出人物" }));
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "确认移出" }));
  expect(screen.getByText("暂时没有关联照片")).toBeVisible();
  expect(screen.getByRole("heading", { level: 1, name: "明月" })).toBeVisible();
  await user.click(screen.getByRole("button", { name: "返回人物" }));
  expect(screen.getByRole("button", { name: "查看明月，0 张照片" })).toBeVisible();
  expect(screen.getAllByRole("img", { name: /已移出的合成照片/ })).toHaveLength(2);
});

it("renders the same synthetic scope and primary actions in English", async () => {
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"en"}');
  const user = userEvent.setup(); renderPeople();
  expect(screen.getByRole("heading", { level: 1, name: "People" })).toBeVisible();
  expect(screen.getByText("Synthetic prototype data")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "Open Unnamed person 1, 1 photo" }));
  expect(screen.getByRole("textbox", { name: "Person name" })).toBeVisible();
  expect(screen.getByRole("button", { name: "Save name" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Merge people" })).toBeEnabled();
  expect(screen.getByText(/No real recognition, API calls or persistent changes/)).toBeVisible();
});

it("does not persist demo names across remounts", async () => {
  const user = userEvent.setup(); const view = renderPeople();
  await user.click(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" }));
  await user.type(screen.getByRole("textbox", { name: "人物名称" }), "临时名称");
  await user.click(screen.getByRole("button", { name: "保存名称" }));
  view.unmount(); renderPeople();
  expect(screen.getByRole("button", { name: "查看未命名人物 1，1 张照片" })).toBeVisible();
  expect(screen.queryByRole("button", { name: /查看临时名称/ })).not.toBeInTheDocument();
});

function renderPeople() {
  return render(<LocaleProvider><ThemeProvider><ToastProvider><MemoryRouter>
    <PeopleWorkspacePrototype />
  </MemoryRouter></ToastProvider></ThemeProvider></LocaleProvider>);
}
