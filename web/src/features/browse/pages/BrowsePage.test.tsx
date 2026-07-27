import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { getDirectory, listDirectories } from "../../../lib/api/catalog";
import { getLibrary, listLibraries } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { BrowsePage } from "./BrowsePage";

vi.mock("../../../lib/api/catalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/catalog")>();
  return {
    ...actual,
    getDirectory: vi.fn(),
    listDirectories: vi.fn(),
  };
});

vi.mock("../../../lib/api/libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api/libraries")>();
  return {
    ...actual,
    getLibrary: vi.fn(),
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
  vi.mocked(getLibrary).mockReset();
  vi.mocked(getDirectory).mockReset();
  vi.mocked(listDirectories).mockReset();

  vi.mocked(listLibraries).mockResolvedValue({
    items: [
      {
        assetCount: 8,
        directoryCount: 3,
        displayPath: "/library/family",
        id: "lib_family",
        lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
        latestScanId: "scan_test",
        name: "家庭影像",
        status: "ready",
      },
    ],
    nextCursor: null,
  });
  vi.mocked(getLibrary).mockResolvedValue({
    etag: '"library-v1"',
    library: {
      assetCount: 8,
      directoryCount: 3,
      displayPath: "/library/family",
      id: "lib_family",
      lastSuccessfulScanAt: "2026-07-28T00:00:00Z",
      latestScanId: "scan_test",
      name: "家庭影像",
      status: "ready",
    },
  });
  vi.mocked(getDirectory).mockResolvedValue({
    breadcrumbs: [
      { id: "dir_root", name: "家庭影像", relativePath: "" },
      { id: "dir_travel", name: "旅行", relativePath: "旅行" },
    ],
    directAssetCount: 2,
    hasChildren: true,
    id: "dir_travel",
    libraryId: "lib_family",
    name: "旅行",
    parentId: "dir_root",
    recursiveAssetCount: 8,
    relativePath: "旅行",
  });
  vi.mocked(listDirectories).mockImplementation(async ({ parentId }) => ({
    items:
      parentId === "dir_travel"
        ? [
            {
              directAssetCount: 1,
              hasChildren: false,
              id: "dir_japan",
              libraryId: "lib_family",
              name: "日本",
              parentId: "dir_travel",
              recursiveAssetCount: 4,
              relativePath: "旅行/日本",
            },
          ]
        : [
            {
              directAssetCount: 2,
              hasChildren: true,
              id: "dir_travel",
              libraryId: "lib_family",
              name: "旅行",
              parentId: "dir_root",
              recursiveAssetCount: 8,
              relativePath: "旅行",
            },
          ],
    nextCursor: null,
  }));
});

it("restores a deep directory, exposes breadcrumbs, and lazily expands its tree path", async () => {
  const user = userEvent.setup();

  renderBrowse();

  expect(await screen.findByRole("heading", { name: "旅行", level: 1 })).toBeVisible();
  expect(screen.getByRole("navigation", { name: "目录位置" })).toHaveTextContent(
    "家庭影像旅行",
  );
  expect(
    screen.getByRole("link", { name: /^日本4 项$/ }),
  ).toHaveAttribute("href", "/libraries/lib_family/browse/dir_japan");
  expect(
    screen.getByRole("link", { name: "家庭影像" }),
  ).toHaveAttribute("href", "/libraries/lib_family/browse");

  const collapse = await screen.findByRole("button", { name: "收起目录 旅行" });
  const tree = screen.getByRole("navigation", { name: "媒体库目录" });
  await user.click(collapse);
  expect(within(tree).queryByRole("link", { name: "日本" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "展开目录 旅行" }));
  expect(await within(tree).findByRole("link", { name: "日本" })).toBeVisible();
});

function renderBrowse() {
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
          <MemoryRouter initialEntries={["/libraries/lib_family/browse/dir_travel"]}>
            <BrowsePage
              directoryId="dir_travel"
              libraryId="lib_family"
              session={session}
            />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}
