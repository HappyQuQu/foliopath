import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { ThemeProvider, useTheme } from "./ThemeProvider";

function Harness() {
  const theme = useTheme();
  return (
    <button type="button" onClick={() => theme.setPreference("dark")}>
      {theme.preference}:{theme.resolvedTheme}
    </button>
  );
}

describe("ThemeProvider", () => {
  it("applies and persists an explicit theme", async () => {
    render(
      <ThemeProvider>
        <Harness />
      </ThemeProvider>,
    );

    await userEvent.click(screen.getByRole("button", { name: "system:light" }));

    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(document.documentElement.dataset.themePreference).toBe("dark");
    expect(window.localStorage.getItem("foliopath.preferences.v1")).toContain('"dark"');
  });
});
