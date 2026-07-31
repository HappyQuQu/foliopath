import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { StorageSettingsPage } from "./StorageSettingsPage";

const mocks = vi.hoisted(() => ({
  settings: {
    etag: '"settings-r1"',
    language: "browser" as const,
    resourceProfile: "balanced" as const,
    scheduledScanIntervalHours: 24,
    thumbnailCacheQuotaBytes: 10 * 1024 ** 3,
    updatedAt: "2026-07-31T00:00:00Z",
  },
  update: vi.fn(),
}));

vi.mock("../queries", () => ({
  useSettingsQuery: () => ({
    data: mocks.settings,
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
  useUpdateSettingsMutation: () => ({
    isPending: false,
    mutateAsync: mocks.update,
  }),
}));

vi.mock("../cache-queries", () => ({
  useCacheSummaryQuery: () => ({
    data: {
      availableBytes: 20 * 1024 ** 3,
      cleanup: { status: "idle" },
      quotaBytes: 10 * 1024 ** 3,
      usageBytes: 1024 ** 3,
    },
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
  useStartCacheCleanupMutation: () => ({
    isPending: false,
    mutateAsync: vi.fn(),
  }),
}));

vi.mock("../../libraries", () => ({
  useLibrariesQuery: () => ({
    data: { pages: [{ items: [] }] },
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isError: false,
    isFetchingNextPage: false,
    isPending: false,
    refetch: vi.fn(),
  }),
}));

beforeEach(() => {
  mocks.update.mockReset();
  mocks.update.mockResolvedValue({});
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
});

it("saves the NAS-friendly resource profile with the existing settings validator", async () => {
  const user = userEvent.setup();
  render(
    <LocaleProvider>
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={["/settings/storage"]}>
            <StorageSettingsPage
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

  await user.click(screen.getByRole("radio", { name: /NAS 友好/ }));
  await user.click(screen.getByRole("button", { name: "保存扫描与缓存设置" }));

  expect(mocks.update).toHaveBeenCalledWith(
    expect.objectContaining({
      etag: '"settings-r1"',
      resourceProfile: "eco",
    }),
  );
});
