import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { expect, it } from "vitest";

import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AppShell } from "./AppShell";

it("closes mobile navigation with Escape and restores focus", async () => {
  const user = userEvent.setup();

  render(
    <ThemeProvider>
      <MemoryRouter>
        <AppShell
          active="libraries"
          identity="管理员"
          librariesHref="/settings/libraries"
          settingsHref="/settings/general"
          title="媒体库"
        >
          <h1>内容</h1>
        </AppShell>
      </MemoryRouter>
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
