import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { expect, it } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { GeneralSettingsPage } from "./GeneralSettingsPage";

it("keeps the language setting in sync with the global quick switch", async () => {
  window.localStorage.setItem(
    "foliopath.preferences.v1",
    '{"locale":"zh-CN"}',
  );
  const user = userEvent.setup();

  render(
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

  expect(screen.getByDisplayValue("简体中文")).toBeVisible();

  await user.click(screen.getByRole("button", { name: "切换到 English" }));

  expect(screen.getByDisplayValue("English")).toBeVisible();
  expect(document.documentElement).toHaveAttribute("lang", "en");
});
