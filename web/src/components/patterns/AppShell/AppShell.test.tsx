import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { expect, it, vi } from "vitest";

import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { ToastProvider } from "../../ui/Toast/ToastProvider";
import { AppShell } from "./AppShell";

it("closes mobile navigation with Escape and restores focus", async () => {
  const user = userEvent.setup();

  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="libraries"
            identity="管理员"
            searchHref="/search"
            settingsHref="/settings/general"
            sidebarContent={<div>目录</div>}
            title="媒体库"
          >
            <h1>内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  const openButton = screen.getByLabelText("打开导航");
  fireEvent.click(openButton);

  expect(openButton).toHaveAttribute("aria-expanded", "true");
  expect(screen.getAllByLabelText("关闭导航")[1]).toHaveFocus();

  await user.keyboard("{Escape}");

  expect(openButton).toHaveAttribute("aria-expanded", "false");
  expect(openButton).toHaveFocus();
});

it("keeps the directory sidebar with redesign navigation at the bottom", () => {
  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            browseHref="/libraries/lib_1/browse"
            identity="管理员"
            searchHref="/search"
            settingsHref="/settings/general"
            sidebarContent={<div>家庭照片目录</div>}
            topbarContent={<div>搜索与浏览工具</div>}
            title="浏览"
          >
            <h1>内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  expect(screen.getByText("家庭照片目录")).toBeVisible();
  expect(
    screen.queryByRole("link", { name: "返回浏览" }),
  ).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "搜索" })).toHaveAttribute(
    "href",
    "/search",
  );
  expect(screen.getByRole("link", { name: "设置" })).toHaveAttribute(
    "href",
    "/settings/general",
  );
});

it("shows contextual routes back to browse and search from settings", () => {
  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="settings"
            browseHref="/libraries/lib_1/browse"
            identity="管理员"
            searchHref="/libraries/lib_1/search"
            settingsHref="/settings/general?libraryId=lib_1"
            title="设置"
          >
            <h1>设置内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  expect(screen.getByRole("link", { name: "返回浏览" })).toHaveAttribute(
    "href",
    "/libraries/lib_1/browse",
  );
  expect(screen.getByRole("link", { name: "搜索" })).toHaveAttribute(
    "href",
    "/libraries/lib_1/search",
  );
  expect(screen.queryByRole("link", { name: "设置" })).not.toBeInTheDocument();
});

it("places the account after theme and supports direct logout", async () => {
  const user = userEvent.setup();
  const onLogout = vi.fn().mockResolvedValue(undefined);

  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            identity="管理员"
            onLogout={onLogout}
            settingsHref="/settings/general"
            title="浏览"
          >
            <h1>内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  const theme = screen.getByRole("button", { name: "切换到深色主题" });
  const account = screen.getByRole("button", { name: "管理员的账户菜单" });
  expect(
    theme.compareDocumentPosition(account) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();

  await user.click(account);
  await user.click(screen.getByRole("menuitem", { name: "退出登录" }));

  expect(onLogout).toHaveBeenCalledOnce();
});

it("uses the fixed redesign sidebar without a resize control", () => {
  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            identity="管理员"
            settingsHref="/settings/general"
            sidebarContent={<div>目录</div>}
            title="浏览"
          >
            <h1>内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  expect(screen.getByRole("complementary", { name: "主导航" })).toBeVisible();
  expect(screen.queryByRole("separator")).not.toBeInTheDocument();
});
