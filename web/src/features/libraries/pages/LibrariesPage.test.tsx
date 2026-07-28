import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { listLibraries } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { LibrariesPage } from "./LibrariesPage";

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/libraries")>();
  return {
    ...actual,
    listLibraries: vi.fn(),
  };
});

const session: AuthenticatedSession = {
  administrator: {
    displayName: "家庭管理员",
    id: "adm_test",
    username: "admin",
  },
  csrfToken: "csrf-token-that-is-long-enough-for-the-contract",
  expiresAt: "2026-08-04T00:00:00Z",
};

beforeEach(() => {
  vi.mocked(listLibraries).mockReset();
  vi.mocked(listLibraries).mockResolvedValue({
    items: [
      {
        assetCount: 9_983,
        directoryCount: 13,
        displayPath: "/library/temp",
        id: "lib_1",
        lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
        latestScanId: "scan_1",
        name: "temp",
        status: "ready",
      },
    ],
    nextCursor: null,
  });
});

it("enables global and per-library browse navigation after indexing", async () => {
  const user = userEvent.setup();
  renderLibraries();

  const browseLinks = await screen.findAllByRole("link", { name: "浏览" });
  expect(browseLinks).toHaveLength(1);
  expect(browseLinks[0]).toHaveAttribute("href", "/libraries/lib_1/browse");

  await user.click(screen.getByRole("button", { name: "浏览" }));
  expect(screen.getByTestId("location")).toHaveTextContent(
    "/libraries/lib_1/browse",
  );
});

function renderLibraries() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  return render(
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <MemoryRouter initialEntries={["/settings/libraries"]}>
            <LibrariesPage session={session} />
            <LocationProbe />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname}</output>;
}
