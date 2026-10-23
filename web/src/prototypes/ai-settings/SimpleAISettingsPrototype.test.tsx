import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../components/ui";
import { LocaleProvider } from "../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../lib/theme/ThemeProvider";
import { SimpleAISettingsPrototype } from "./SimpleAISettingsPrototype";

const fetchSpy = vi.fn(() => { throw new Error("The disposable prototype must not call an API"); });
const xhrOpenSpy = vi.fn();

beforeEach(() => {
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  fetchSpy.mockClear();
  xhrOpenSpy.mockClear();
  vi.stubGlobal("fetch", fetchSpy);
  vi.spyOn(XMLHttpRequest.prototype, "open").mockImplementation(() => {
    xhrOpenSpy();
    throw new Error("The disposable prototype must not call an API");
  });
});

afterEach(() => {
  // Every interaction is an in-memory simulation, including confirmed maintenance
  // and the proposed automatic-tag switch. No API provider is mounted either.
  expect(fetchSpy).not.toHaveBeenCalled();
  expect(xhrOpenSpy).not.toHaveBeenCalled();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("shows three per-library switches, marks automatic tags as a proposal, and hides maintenance", () => {
  renderPrototype();
  expect(screen.getByRole("heading", { level: 1, name: "智能分析" })).toBeVisible();
  expect(screen.getAllByRole("switch")).toHaveLength(3);
  expect(screen.getByRole("switch", { name: "开启家庭影像的智能搜索" })).toBeChecked();
  expect(screen.getByRole("switch", { name: "开启家庭影像的人物识别" })).toBeChecked();
  expect(screen.getByRole("switch", { name: "开启家庭影像的自动标签" })).not.toBeChecked();
  expect(screen.getByText("提案 · 未接入")).toBeVisible();
  expect(screen.getByText(/尚无后端契约，此开关不会运行分析或修改照片标签/)).toBeVisible();
  expect(screen.getByText(/无需预选标签。识别后自动添加到照片/)).toBeVisible();
  expect(screen.getByText("高级维护").closest("details")).not.toHaveAttribute("open");
  expect(screen.getByRole("button", { name: "清理索引" })).not.toBeVisible();
  expect(screen.getByText("交互原型 · 演示数据")).toBeVisible();
  expect(screen.getByText(/原型预览，不保存真实设置/)).toBeVisible();
});

it("routes the old tags initial view to settings without a candidate or review workflow", () => {
  renderPrototype({ initialView: "tags" });
  expect(screen.getByRole("heading", { level: 1, name: "智能分析" })).toBeVisible();
  expect(screen.getAllByRole("switch")).toHaveLength(3);
  expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  expect(screen.queryByText("候选标签")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /生成缺失建议|保存候选标签|确认建议|管理标签建议/ })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "查看标签" })).toHaveAttribute("href", "/iframe.html?id=prototypes-intelligence-photo-workspace--tags&viewMode=story");
});

it("uses the optional in-memory library navigation callback for View tags", async () => {
  const user = userEvent.setup();
  const onOpenLibrary = vi.fn();
  renderPrototype({ onOpenLibrary });
  expect(screen.queryByRole("link", { name: "查看标签" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "查看标签" }));
  expect(onOpenLibrary).toHaveBeenCalledTimes(1);
  expect(onOpenLibrary).toHaveBeenCalledWith();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});

it("keeps search and people switches library-scoped and retains results when switched off", async () => {
  const user = userEvent.setup(); renderPrototype();
  await user.click(screen.getByRole("switch", { name: "开启家庭影像的智能搜索" }));
  expect(screen.getByText(/已关闭「家庭影像」的功能，保留已有分析结果和人工整理/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  expectProgress("1860", "420");
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "关闭" }));
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "travel");
  expect(screen.getByRole("switch", { name: "开启旅行记录的智能搜索" })).toBeChecked();
  expect(screen.getByRole("switch", { name: "开启旅行记录的人物识别" })).not.toBeChecked();
  await user.click(screen.getByRole("switch", { name: "开启旅行记录的人物识别" }));
  expect(screen.getByText(/开启功能并提交补充分析；不代表分析完成/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  const dialog = screen.getByRole("dialog");
  expectProgress("640", "0");
  expect(within(dialog).getByText("已就绪")).toBeVisible();
  expect(within(dialog).getByText("等待处理")).toBeVisible();
  await user.click(within(dialog).getByRole("button", { name: "关闭" }));
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "family");
  expect(screen.getByRole("switch", { name: "开启家庭影像的智能搜索" })).not.toBeChecked();
  expect(screen.getByRole("switch", { name: "开启家庭影像的人物识别" })).toBeChecked();
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "travel");
  expect(screen.getByRole("switch", { name: "开启旅行记录的人物识别" })).toBeChecked();
});

it("keeps the automatic-tag proposal library-scoped without queuing analysis or fabricating results", async () => {
  const user = userEvent.setup(); renderPrototype();
  await user.click(screen.getByRole("switch", { name: "开启家庭影像的自动标签" }));
  expect(screen.getByRole("switch", { name: "开启家庭影像的自动标签" })).toBeChecked();
  expect(screen.getByText(/已开启「家庭影像」的自动标签意愿。尚无后端契约，未执行识别或写入标签/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  const dialog = screen.getByRole("dialog");
  expectProgress("1860", "420");
  expect(within(dialog).getAllByRole("progressbar")).toHaveLength(2);
  expect(within(dialog).getByText("自动标签：仅交互提案，未接入分析，没有处理进度。")).toBeVisible();
  expect(within(dialog).queryByText("等待处理")).not.toBeInTheDocument();
  await user.click(within(dialog).getByRole("button", { name: "关闭" }));
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "travel");
  expect(screen.getByRole("switch", { name: "开启旅行记录的自动标签" })).not.toBeChecked();
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "family");
  expect(screen.getByRole("switch", { name: "开启家庭影像的自动标签" })).toBeChecked();
  await user.click(screen.getByRole("switch", { name: "开启家庭影像的自动标签" }));
  expect(screen.getByRole("switch", { name: "开启家庭影像的自动标签" })).not.toBeChecked();
  expect(screen.getByText(/已关闭「家庭影像」的自动标签意愿。尚无后端契约，未执行识别或写入标签/)).toBeVisible();
});

it("blocks all three offline switches and maintenance while preserving existing progress", async () => {
  const user = userEvent.setup(); renderPrototype({ initialLibrary: "archive" });
  expect(screen.getByText(/媒体库离线，暂时不能更改或运行分析/)).toBeVisible();
  expect(screen.getAllByRole("switch")).toHaveLength(3);
  for (const toggle of screen.getAllByRole("switch")) expect(toggle).toBeDisabled();
  await user.click(screen.getByText("高级维护"));
  expect(screen.getByRole("button", { name: "重新分析" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "清理索引" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  expectProgress("900", "300");
});

it("keeps models read-only, globally shared and owned by Compose", async () => {
  const user = userEvent.setup(); renderPrototype();
  await user.click(screen.getByText("高级维护"));
  expect(screen.getByText("由 Docker Compose 管理 · 只读")).toBeVisible();
  expect(screen.getByText("ViT-B-32__openai")).toBeVisible();
  expect(screen.getByText("buffalo_l")).toBeVisible();
  expect(screen.getAllByRole("combobox")).toHaveLength(1);
  expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /下载|激活|保存模型/ })).not.toBeInTheDocument();
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "travel");
  expect(screen.getByText("ViT-B-32__openai")).toBeVisible();
  expect(screen.getByText("buffalo_l")).toBeVisible();
});

it.each(["清理索引", "重新分析"])("requires confirmation and makes cancelling %s a no-op", async action => {
  const user = userEvent.setup(); renderPrototype();
  await user.click(screen.getByText("高级维护"));
  await user.click(screen.getByRole("button", { name: action }));
  const dialog = screen.getByRole("dialog");
  expect(within(dialog).getByText(/仅影响当前媒体库的搜索索引/)).toBeVisible();
  expect(within(dialog).getByRole("button", { name: "确认演示操作" })).toBeVisible();
  await user.click(within(dialog).getByRole("button", { name: "取消" }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(screen.queryByText(/演示操作已确认/)).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  expectProgress("1860", "420");
  expect(within(screen.getByRole("dialog")).queryByText("等待处理")).not.toBeInTheDocument();
});

it("clears only the selected search index, retaining people results and the automatic-tag proposal", async () => {
  const user = userEvent.setup(); renderPrototype();
  await user.click(screen.getByRole("switch", { name: "开启家庭影像的自动标签" }));
  await user.click(screen.getByText("高级维护"));
  await user.click(screen.getByRole("button", { name: "清理索引" }));
  expect(screen.getByRole("dialog", { name: "清理「家庭影像」的搜索索引？" })).toBeVisible();
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "确认演示操作" }));
  expect(screen.getByRole("switch", { name: "开启家庭影像的智能搜索" })).toBeChecked();
  expect(screen.getByRole("switch", { name: "开启家庭影像的人物识别" })).toBeChecked();
  expect(screen.getByRole("switch", { name: "开启家庭影像的自动标签" })).toBeChecked();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  expectProgress("0", "420");
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "关闭" }));
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), "travel");
  expect(screen.getByRole("switch", { name: "开启旅行记录的自动标签" })).not.toBeChecked();
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  expectProgress("640", "0");
});

it("queues a confirmed search rebuild without queuing people or claiming completion", async () => {
  const user = userEvent.setup(); renderPrototype();
  await user.click(screen.getByText("高级维护"));
  await user.click(screen.getByRole("button", { name: "重新分析" }));
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "确认演示操作" }));
  await user.click(screen.getByRole("button", { name: "处理进度" }));
  const dialog = screen.getByRole("dialog");
  expectProgress("1860", "420");
  expect(within(dialog).getAllByText("等待处理")).toHaveLength(1);
  expect(within(dialog).getByText("分析中")).toBeVisible();
});

it("keeps the proposal boundary visible in English as well", () => {
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"en"}');
  renderPrototype();
  expect(screen.getAllByRole("switch")).toHaveLength(3);
  expect(screen.getByRole("switch", { name: "Enable automatic tags for Family photos" })).not.toBeChecked();
  expect(screen.getByText("Proposal · not connected")).toBeVisible();
  expect(screen.getByText(/Interaction proposal only: no backend contract/)).toBeVisible();
  expect(screen.getByRole("link", { name: "View tags" })).toBeVisible();
});

function expectProgress(search: string, people: string) {
  const dialog = within(screen.getByRole("dialog"));
  expect(dialog.getByRole("progressbar", { name: "智能搜索处理进度" })).toHaveAttribute("value", search);
  expect(dialog.getByRole("progressbar", { name: "人物识别处理进度" })).toHaveAttribute("value", people);
}

function renderPrototype(props: ComponentProps<typeof SimpleAISettingsPrototype> = {}) {
  return render(<LocaleProvider><ThemeProvider><ToastProvider><MemoryRouter initialEntries={["/settings/ai"]}>
    <SimpleAISettingsPrototype {...props} />
  </MemoryRouter></ToastProvider></ThemeProvider></LocaleProvider>);
}
