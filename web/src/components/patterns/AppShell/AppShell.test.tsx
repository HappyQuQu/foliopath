import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { expect, it, vi } from "vitest";

import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { ToastProvider } from "../../ui/Toast/ToastProvider";
import { AppShell } from "./AppShell";

it.each([
  ["/libraries/lib_1/search?q=old&mode=semantic", "/libraries/lib_1/search?q=blue+%26+%E8%93%9D&mode=semantic"],
  ["/libraries/lib_1/search?directoryId=dir_1&scope=directory", "/libraries/lib_1/search?directoryId=dir_1&scope=directory&q=blue+%26+%E8%93%9D"],
])("replaces the global query without losing search context: %s", async (searchHref, expected) => {
  const user = userEvent.setup();
  function Location() { const location = useLocation(); return <output data-testid="search-location">{location.pathname}{location.search}</output>; }
  render(<ThemeProvider><ToastProvider><MemoryRouter>
    <AppShell active="search" homeHref="/" identity="管理员" searchHref={searchHref} settingsHref="/settings/general" title="搜索"><Location /></AppShell>
  </MemoryRouter></ToastProvider></ThemeProvider>);
  await user.type(screen.getByRole("searchbox"), "blue & 蓝");
  await user.keyboard("{Enter}");
  expect(screen.getByTestId("search-location")).toHaveTextContent(expected);
});

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

it("forwards search guidance without interpreting feature URL modes in the shared shell", async () => {
  const user = userEvent.setup();
  function Location() { const location = useLocation(); return <output>{location.pathname}{location.search}</output>; }
  render(<ThemeProvider><ToastProvider><MemoryRouter>
    <AppShell active="search" homeHref="/" identity="管理员" searchHref="/libraries/lib_1/search?mode=semantic&scope=library"
      searchInput={{ label: "Describe visual content", placeholder: "Cars by the sea" }} settingsHref="/settings/general" title="搜索"><Location /></AppShell>
  </MemoryRouter></ToastProvider></ThemeProvider>);
  const input = screen.getByRole("searchbox", { name: "Describe visual content" });
  expect(input).toHaveAttribute("placeholder", "Cars by the sea");
  await user.type(input, "汽车");
  await user.keyboard("{Enter}");
  expect(screen.getByRole("status")).toHaveTextContent("/libraries/lib_1/search?mode=semantic&scope=library&q=%E6%B1%BD%E8%BD%A6");
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
  expect(screen.getByRole("menuitem", { name: "管理中心" })).toHaveAttribute(
    "href",
    "/settings/general",
  );
});

it("closes an open drawer when its destination changes without remounting the shell", () => {
  function Workspace() {
    const [destination, setDestination] = useState("library:tags");
    const [search, setSearch] = useState("");
    return <AppShell active="browse" homeHref="/" identity="管理员" searchHref="/search" settingsHref="/settings/general"
      navigationKey={destination} title="标签" topbarContent={<span>标签</span>}
      sidebarContent={<><button onClick={() => setSearch("travel")}>Search tags</button><button onClick={() => setDestination("library:travel")}>Choose travel</button></>}>
      <output>{search}</output>
    </AppShell>;
  }
  render(<ThemeProvider><ToastProvider><MemoryRouter><Workspace /></MemoryRouter></ToastProvider></ThemeProvider>);
  const open = screen.getByLabelText("打开导航");
  fireEvent.click(open);
  fireEvent.click(screen.getByText("Search tags"));
  expect(open).toHaveAttribute("aria-expanded", "true");
  fireEvent.click(screen.getByText("Choose travel"));
  expect(open).toHaveAttribute("aria-expanded", "false");
  expect(screen.getByLabelText("打开导航")).toBe(open);
  expect(open).toHaveFocus();
  expect(screen.getByText("travel")).toBeInTheDocument();
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
  expect(screen.getByRole("menuitem", { name: "管理中心" })).toHaveAttribute(
    "href",
    "/settings/general?libraryId=lib_1",
  );
});

it("exposes people as a direct global destination when configured", () => {
  render(
    <ThemeProvider><ToastProvider><MemoryRouter initialEntries={["/libraries/lib_1/browse"]}>
      <AppShell active="browse" homeHref="/" identity="管理员" peopleHref="/intelligence?view=clusters" searchHref="/search" settingsHref="/settings/general" title="浏览">
        <h1>内容</h1>
      </AppShell>
    </MemoryRouter></ToastProvider></ThemeProvider>,
  );
  expect(screen.getByRole("link", { name: "人物" })).toHaveAttribute("href", "/intelligence?view=clusters");
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

it("exposes the accepted workspace sidebar resize control", () => {
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
  expect(screen.getByRole("separator", { name: "调整目录栏宽度" })).toBeVisible();
});

it("uses feature supplied sidebar names and opens tools without requiring a topbar slot", async () => {
  const user = userEvent.setup();
  render(<ThemeProvider><ToastProvider><MemoryRouter><AppShell active="search" homeHref="/" identity="Admin" searchHref="/search" settingsHref="/settings/general" title="Search" sidebarLabel="Search tools" navigationLabel="Open search tools" sidebarContent={<input aria-label="Sidebar query" />}><h1>Results</h1></AppShell></MemoryRouter></ToastProvider></ThemeProvider>);
  expect(screen.getByRole("complementary", { name: "Search tools" })).toBeVisible();
  const opener = screen.getByLabelText("Open search tools");
  await user.click(opener);
  await user.keyboard("{Escape}");
  expect(opener).toHaveFocus();
  expect(opener).toHaveAttribute("aria-expanded", "false");
});
