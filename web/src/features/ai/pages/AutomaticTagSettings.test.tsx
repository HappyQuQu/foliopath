import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ApiError } from "../../../lib/api/errors";
import { AutomaticTagSettings } from "./AutomaticTagSettings";
import asyncStyles from "../../../components/ui/AsyncState/AsyncState.module.css";

const mocks = vi.hoisted(() => ({ query: vi.fn(), save: vi.fn(), job: vi.fn(), remember: vi.fn() }));
vi.mock("../automaticTagQueries", () => ({ useAutomaticTagSettings: mocks.query,
  useUpdateAutomaticTags: () => ({ mutateAsync: mocks.save, isPending: false }),
  useRequestAutomaticTags: () => ({ mutateAsync: mocks.job, isPending: false }),
}));
const value = { libraryId: "lib_2", enabled: true, revision: 2, state: "ready", offline: false,
  model: { identity: "a".repeat(64), displayName: "JoyTag", protocol: "foliopath-tag-v1" },
  coverage: { eligible: 4, completed: 4, failed: 0, stale: 0, revision: 3 }, activeOperationId: null, etag: '"autotag-library-2-r2"' };
const show = (mode: "settings" | "tasks" = "settings", onSettings = vi.fn()) => render(<MemoryRouter><LocaleProvider><AutomaticTagSettings libraryId="lib_2" csrfToken="csrf" onOperation={mocks.remember} mode={mode} onSettings={onSettings} /></LocaleProvider></MemoryRouter>);
beforeEach(() => {
  Object.values(mocks).forEach(m => m.mockReset());
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  mocks.query.mockReturnValue({ data: value, isPending: false, isError: false, refetch: vi.fn() });
  mocks.job.mockResolvedValue({ id: "aio_tags" }); mocks.save.mockResolvedValue(value);
});
it("keeps enablement out of task rows and routes disabled work to settings without writing", async () => {
  const onSettings = vi.fn();
  mocks.query.mockReturnValue({ data: { ...value, enabled: false, state: "disabled" }, isPending: false, isError: false });
  show("tasks", onSettings);
  expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "去开启" }));
  expect(onSettings).toHaveBeenCalledOnce();
  expect(mocks.save).not.toHaveBeenCalled();
  expect(mocks.job).not.toHaveBeenCalled();
});
it("keeps task-row reprocessing behind maintenance and its protection confirmation", async () => {
  const user = userEvent.setup();
  show("tasks");
  expect(screen.queryByRole("button", { name: "重新识别" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "更多：自动标签" }));
  await user.click(screen.getByRole("button", { name: "重新识别" }));
  expect(mocks.job).not.toHaveBeenCalled();
  expect(screen.getByRole("dialog")).toHaveTextContent(/人工添加的标签和手动删除记录会保留/);
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "取消" }));
  expect(screen.getByRole("button", { name: "重新识别" })).toHaveFocus();
  expect(mocks.job).not.toHaveBeenCalled();
});
it("does not show a stale completion or enable work after a task-row status failure", () => {
  mocks.query.mockReturnValue({ data: value, isPending: false, isError: true, refetch: vi.fn() });
  show("tasks");
  expect(screen.getByRole("alert")).toHaveClass(asyncStyles.compact!);
  expect(screen.getByRole("button", { name: "补齐未识别" })).toBeDisabled();
  expect(screen.queryByText(/已完成 · 新图片/)).not.toBeInTheDocument();
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
});
it("keeps task-row loading compact without exposing maintenance or enablement", () => {
  mocks.query.mockReturnValue({ data: undefined, isPending: true, isError: false });
  show("tasks");
  expect(screen.getByRole("status")).toHaveClass(asyncStyles.compact!);
  expect(screen.getByRole("button", { name: "补齐未识别" })).toBeDisabled();
  expect(screen.queryByRole("button", { name: /更多|去开启/ })).not.toBeInTheDocument();
});
it("falls back to the task heading after admission disables the confirmation trigger", async () => {
  mocks.remember.mockImplementation(() => mocks.query.mockReturnValue({ data: { ...value, state: "building", activeOperationId: "aio_tags" }, isPending: false, isError: false }));
  show("tasks");
  await userEvent.click(screen.getByRole("button", { name: "更多：自动标签" }));
  await userEvent.click(screen.getByRole("button", { name: "重新识别" }));
  await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "重新识别" }));
  expect(screen.getByRole("button", { name: "重新识别" })).toBeDisabled();
  expect(screen.getByRole("heading", { name: "自动标签" })).toHaveFocus();
});
it("uses one library toggle and a scoped real-tag link without vocabulary controls", () => {
  show(); expect(screen.getAllByRole("switch")).toHaveLength(1);
  expect(screen.getByRole("link", { name: "查看图片标签" })).toHaveAttribute("href", "/tags?libraryId=lib_2");
  expect(screen.getByText(/手动添加的标签会保留/)).toBeVisible();
  expect(screen.queryByRole("button", { name: /候选|审核/ })).not.toBeInTheDocument();
});
it("disables using the loaded ETag and never clears existing tags", async () => {
  show(); await userEvent.click(screen.getByRole("switch"));
  expect(mocks.save).toHaveBeenCalledWith({ libraryId: "lib_2", csrfToken: "csrf", enabled: false, etag: value.etag });
  expect(mocks.job).not.toHaveBeenCalled();
});
it("requires confirmation before reprocessing and describes manual protection", async () => {
  show(); await userEvent.click(screen.getByRole("button", { name: "重新识别" }));
  expect(mocks.job).not.toHaveBeenCalled();
  const dialog = screen.getByRole("dialog"); expect(dialog).toHaveTextContent(/人工添加的标签和手动删除记录会保留/);
  await userEvent.click(within(dialog).getByRole("button", { name: "重新识别" }));
  expect(mocks.job).toHaveBeenCalledWith(expect.objectContaining({ libraryId: "lib_2", mode: "all", csrfToken: "csrf" }));
  expect(mocks.remember).toHaveBeenCalledWith({ id: "aio_tags" });
});
it.each(["awaiting_model", "disabled"])("does not launch jobs in %s state", state => {
  mocks.query.mockReturnValue({ data: { ...value, state, enabled: state !== "disabled" }, isPending: false, isError: false });
  show(); expect(screen.getByRole("button", { name: "重新识别" })).toBeDisabled();
});
it("tracks a server-created background operation and prevents duplicate admission", () => {
  mocks.query.mockReturnValue({ data: { ...value, state: "building", activeOperationId: "aio_background" }, isPending: false, isError: false });
  show(); expect(mocks.remember).toHaveBeenCalledWith({ id: "aio_background" });
  expect(screen.getByRole("button", { name: "补齐未识别" })).toBeDisabled();
});
it("retains a revision conflict as a visible error without unconditional retry", async () => {
  mocks.save.mockRejectedValue(new ApiError({ status: 412, code: "precondition_failed", message: "changed", requestId: undefined }));
  show(); await userEvent.click(screen.getByRole("switch"));
  expect(await screen.findByText(/设置已变化/)).toBeVisible(); expect(mocks.save).toHaveBeenCalledOnce();
});
it("hides successful status on a failed status refresh and disables writes", () => {
  mocks.query.mockReturnValue({ data: value, isError: true, isPending: false, refetch: vi.fn() });
  show(); expect(screen.getByRole("switch")).toBeDisabled();
  expect(screen.queryByText(/已完成 · 新图片/)).not.toBeInTheDocument();
});

it("requires explicit reprocessing for stopped or degraded work instead of offering missing-only automatic recovery", async () => {
  mocks.query.mockReturnValue({ data: { ...value, state: "degraded" }, isPending: false, isError: false });
  show();
  expect(screen.getByText(/任务已停止.*重新识别/)).toBeVisible();
  expect(screen.getByRole("button", { name: "补齐未识别" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "重新识别" })).toBeEnabled();
  await userEvent.click(screen.getByRole("button", { name: "重新识别" }));
  await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "重新识别" }));
  expect(mocks.job).toHaveBeenCalledWith(expect.objectContaining({ mode: "all" }));
});

it("does not transfer a reprocess confirmation or stale error to another library", async () => {
  const view = show();
  await userEvent.click(screen.getByRole("button", { name: "重新识别" }));
  expect(screen.getByRole("dialog")).toBeVisible();
  mocks.query.mockReturnValue({ data: { ...value, libraryId: "lib_3", etag: '"autotag-library-3-r1"' }, isPending: false, isError: false });
  view.rerender(<MemoryRouter><LocaleProvider><AutomaticTagSettings libraryId="lib_3" csrfToken="csrf" onOperation={mocks.remember} /></LocaleProvider></MemoryRouter>);
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(mocks.job).not.toHaveBeenCalled();
});
