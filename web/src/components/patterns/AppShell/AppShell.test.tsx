import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { expect, it, vi } from "vitest";

import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
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
            homeHref="/"
            identity="管理员"
            searchHref="/search"
            settingsHref="/settings/general"
            sidebarContent={<div>目录</div>}
            topbarContent={<div>当前位置</div>}
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

it("keeps the directory sidebar without duplicate product navigation", async () => {
  const user = userEvent.setup();
  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            browseHref="/libraries/lib_1/browse"
            homeHref="/"
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
  expect(screen.getByRole("searchbox")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "管理员的账户菜单" }));
  expect(screen.getByRole("menuitem", { name: "设置" })).toHaveAttribute(
    "href",
    "/settings/general",
  );
});

it("uses only the global header on pages without a directory sidebar", async () => {
  const user = userEvent.setup();
  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="settings"
            browseHref="/libraries/lib_1/browse"
            homeHref="/"
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

  expect(screen.queryByRole("link", { name: "返回浏览" })).not.toBeInTheDocument();
  expect(screen.getByRole("searchbox")).toBeVisible();
  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "管理员的账户菜单" }));
  expect(screen.getByRole("menuitem", { name: "设置" })).toHaveAttribute(
    "href",
    "/settings/general?libraryId=lib_1",
  );
});

it("opens the global account menu and supports direct logout", async () => {
  const user = userEvent.setup();
  const onLogout = vi.fn().mockResolvedValue(undefined);

  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            homeHref="/"
            identity="管理员"
            onLogout={onLogout}
            searchHref="/search"
            settingsHref="/settings/general"
            title="浏览"
          >
            <h1>内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  const account = screen.getByRole("button", { name: "管理员的账户菜单" });

  await user.click(account);
  await user.click(screen.getByRole("menuitem", { name: "退出登录" }));

  expect(onLogout).toHaveBeenCalledOnce();
});

it("switches theme directly beside the global account menu", async () => {
  const user = userEvent.setup();

  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            homeHref="/"
            identity="管理员"
            searchHref="/search"
            settingsHref="/settings/general"
            title="浏览"
          >
            <h1>内容</h1>
          </AppShell>
        </MemoryRouter>
      </ToastProvider>
    </ThemeProvider>,
  );

  const toggle = screen.getByRole("button", { name: "切换到深色主题" });
  const account = screen.getByRole("button", { name: "管理员的账户菜单" });

  expect(toggle.compareDocumentPosition(account)).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING,
  );

  await user.click(toggle);

  expect(
    screen.getByRole("button", { name: "切换到浅色主题" }),
  ).toBeVisible();
  expect(document.documentElement).toHaveAttribute("data-theme", "dark");
});

it("switches language directly in the global header", async () => {
  const user = userEvent.setup();
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    '{"locale":"zh-CN"}',
  );

  render(
    <LocaleProvider>
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter>
            <AppShell
              active="browse"
              homeHref="/"
              identity="管理员"
              searchHref="/search"
              settingsHref="/settings/general"
              title="浏览"
            >
              <h1>内容</h1>
            </AppShell>
          </MemoryRouter>
        </ToastProvider>
      </ThemeProvider>
    </LocaleProvider>,
  );

  const languageToggle = screen.getByRole("button", {
    name: "切换到 English",
  });
  const themeToggle = screen.getByRole("button", {
    name: "切换到深色主题",
  });
  const account = screen.getByRole("button", { name: "管理员的账户菜单" });

  expect(languageToggle.compareDocumentPosition(themeToggle)).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING,
  );
  expect(themeToggle.compareDocumentPosition(account)).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING,
  );

  await user.click(languageToggle);

  expect(document.documentElement).toHaveAttribute("lang", "en");
  expect(
    screen.getByRole("button", { name: "Switch to 简体中文" }),
  ).toBeVisible();
});

it("uses the fixed redesign sidebar without a resize control", () => {
  render(
    <ThemeProvider>
      <ToastProvider>
        <MemoryRouter>
          <AppShell
            active="browse"
            homeHref="/"
            identity="管理员"
            searchHref="/search"
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

  expect(
    screen.getByRole("complementary", { name: "媒体库和目录" }),
  ).toBeVisible();
  expect(screen.queryByRole("separator")).not.toBeInTheDocument();
});
