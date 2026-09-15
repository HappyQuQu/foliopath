import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";
import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { AIModelProxyPanel } from "./AIModelProxyPanel";

const mocks = vi.hoisted(() => ({ save: vi.fn(), test: vi.fn(), query: vi.fn() }));
vi.mock("../queries", () => ({
  useAIModelProxyQuery: mocks.query,
  useSaveAIModelProxyMutation: () => ({ mutateAsync: mocks.save, isPending: false, isError: false }),
  useTestAIModelProxyMutation: () => ({ mutateAsync: mocks.test, isPending: false }),
}));
beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
  mocks.query.mockReturnValue({ data: { mode: "inherit", proxyUrl: "", revision: 3, deploymentConfigured: true, effectiveMode: "proxy" }, isPending: false, isError: false });
  mocks.save.mockResolvedValue({});
  mocks.test.mockResolvedValue({ available: true });
});
function show() {
  render(<LocaleProvider><ToastProvider><AIModelProxyPanel csrfToken="test-csrf" /></ToastProvider></LocaleProvider>);
}
it("tests an unsaved custom proxy without saving and clears obsolete test results on edit", async () => {
  const user = userEvent.setup(); show();
  expect(screen.getByRole("button", { name: "保存代理设置" })).toBeDisabled();
  await user.selectOptions(screen.getByRole("combobox"), "custom");
  expect(screen.getByRole("button", { name: "测试连接" })).toBeDisabled();
  await user.type(screen.getByRole("textbox", { name: /代理地址/ }), "http://192.168.2.231:7890");
  await user.click(screen.getByRole("button", { name: "测试连接" }));
  await screen.findByText(/测试文件下载及校验成功/);
  expect(mocks.test).toHaveBeenCalledWith({ csrfToken: "test-csrf", body: { mode: "custom", proxyUrl: "http://192.168.2.231:7890", revision: 3 } });
  expect(mocks.save).not.toHaveBeenCalled();
  await user.type(screen.getByRole("textbox", { name: /代理地址/ }), "/");
  expect(screen.queryByText(/测试文件下载及校验成功/)).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "保存代理设置" }));
  await waitFor(() => expect(mocks.save).toHaveBeenCalledWith({ csrfToken: "test-csrf", body: { mode: "custom", proxyUrl: "http://192.168.2.231:7890/", revision: 3 } }));
});
it("direct mode saves an empty URL even after editing a custom address", async () => {
  const user = userEvent.setup(); show();
  await user.selectOptions(screen.getByRole("combobox"), "custom");
  await user.type(screen.getByRole("textbox", { name: /代理地址/ }), "http://192.168.2.231:7890");
  await user.selectOptions(screen.getByRole("combobox"), "direct");
  expect(screen.queryByRole("textbox", { name: /代理地址/ })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "保存代理设置" }));
  expect(mocks.save).toHaveBeenCalledWith({ csrfToken: "test-csrf", body: { mode: "direct", proxyUrl: "", revision: 3 } });
});
it("reports failed probes without claiming success", async () => {
  mocks.test.mockResolvedValue({ available: false });
  show(); await userEvent.click(screen.getByRole("button", { name: "测试连接" }));
  await screen.findByText("测试连接失败，请检查代理和服务器网络。");
  expect(mocks.save).not.toHaveBeenCalled();
});
