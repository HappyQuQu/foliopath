import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { GeneralSettingsPage } from "./GeneralSettingsPage";

beforeEach(() => {
  window.localStorage.clear();
});

it("keeps the language setting in sync with the global quick switch", async () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    '{"locale":"zh-CN"}',
  );
  const user = userEvent.setup();

  renderGeneralSettings();

  expect(screen.getByDisplayValue("简体中文")).toBeVisible();

  await user.click(screen.getByRole("button", { name: "切换到 English" }));

  expect(screen.getByDisplayValue("English")).toBeVisible();
  expect(document.documentElement).toHaveAttribute("lang", "en");
});

it("defaults video preview autoplay on and saves an explicit opt-out", async () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    '{"locale":"zh-CN"}',
  );
  const user = userEvent.setup();

  renderGeneralSettings();

  const autoplay = screen.getByRole("switch", {
    name: /自动播放视频预览/,
  });
  expect(autoplay).toBeChecked();

  await user.click(autoplay);
  await user.click(screen.getByRole("button", { name: "保存更改" }));

  expect(
    JSON.parse(
      window.localStorage.getItem("foliopath.preferences.v1") ?? "{}",
    ),
  ).toMatchObject({ previewAutoplay: false });
});

it("saves default sorting and grid or masonry layout together", async () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    '{"locale":"zh-CN"}',
  );
  const user = userEvent.setup();

  renderGeneralSettings();

  await user.selectOptions(
    screen.getByRole("combobox", { name: /默认布局/ }),
    "masonry",
  );
  await user.selectOptions(
    screen.getByRole("combobox", { name: /默认排序/ }),
    "size:desc",
  );
  await user.click(screen.getByRole("button", { name: "保存更改" }));

  expect(
    JSON.parse(
      window.localStorage.getItem("foliopath.preferences.v1") ?? "{}",
    ),
  ).toMatchObject({ mediaLayout: "masonry", mediaSort: "size:desc" });
});

function renderGeneralSettings() {
  return render(
    <LocaleProvider>
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={["/settings/general"]}>
            <GeneralSettingsPage
              logoutPending={false}
              onLogout={async () => undefined}
              session={{
                administrator: {
                  displayName: "管理员",
                  id: "adm_test",
                  username: "admin",
                },
                csrfToken: "test-csrf-token-that-is-long-enough",
                expiresAt: "2026-08-04T00:00:00Z",
              }}
            />
          </MemoryRouter>
        </ToastProvider>
      </ThemeProvider>
    </LocaleProvider>,
  );
}
