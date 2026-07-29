import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ToastProvider } from "../components/ui/Toast/ToastProvider";
import {
  getAuthenticationStatus,
  getSession,
  login,
  logout,
  setupAdministrator,
  type AuthenticatedSession,
} from "../lib/api/auth";
import { ApiError } from "../lib/api/errors";
import { getSystemReadiness } from "../lib/api/readiness";
import { listLibraries } from "../lib/api/libraries";
import { ThemeProvider } from "../lib/theme/ThemeProvider";
import { AppRoutes } from "./AppRouter";

vi.mock("../lib/api/auth", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api/auth")>();
  return {
    ...actual,
    getAuthenticationStatus: vi.fn(),
    getSession: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    setupAdministrator: vi.fn(),
  };
});

vi.mock("../lib/api/readiness", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api/readiness")>();
  return {
    ...actual,
    getSystemReadiness: vi.fn(),
  };
});

vi.mock("../lib/api/libraries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api/libraries")>();
  return {
    ...actual,
    listLibraries: vi.fn(),
  };
});

vi.mock("../features/browse", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../features/browse")>();
  return {
    ...actual,
    BrowsePage: ({ libraryId }: { libraryId: string }) => (
      <h1>浏览媒体库 {libraryId}</h1>
    ),
  };
});

const session: AuthenticatedSession = {
  administrator: {
    displayName: "家庭管理员",
    id: "01JTESTADMIN00000000000000",
    username: "admin",
  },
  csrfToken: "csrf-token-that-is-long-enough-for-the-contract",
  expiresAt: "2026-08-04T00:00:00Z",
};

describe("authentication routes", () => {
  beforeEach(() => {
    vi.mocked(getSystemReadiness).mockReset();
    vi.mocked(getSystemReadiness).mockResolvedValue({
      status: "ready",
      reasonCode: null,
    });
    vi.mocked(getAuthenticationStatus).mockReset();
    vi.mocked(getSession).mockReset();
    vi.mocked(login).mockReset();
    vi.mocked(logout).mockReset();
    vi.mocked(setupAdministrator).mockReset();
    vi.mocked(listLibraries).mockReset();
    vi.mocked(listLibraries).mockResolvedValue({
      items: [],
      nextCursor: null,
    });
  });

  it("shows a safe application-data failure without exposing a host path", async () => {
    vi.mocked(getSystemReadiness).mockResolvedValue({
      status: "not_ready",
      reasonCode: "application_data_unavailable",
    });

    renderRoutes("/");

    expect(await screen.findByRole("heading", { name: "FolioPath 无法完成启动" })).toBeVisible();
    expect(screen.getByText(/应用数据目录不可用/)).toBeVisible();
    expect(screen.queryByText(/Users|app\/data|stack|sqlite/i)).not.toBeInTheDocument();
    expect(getAuthenticationStatus).not.toHaveBeenCalled();
  });

  it("completes first-administrator setup through the real route flow", async () => {
    const user = userEvent.setup();
    vi.mocked(getAuthenticationStatus).mockResolvedValue({ setupRequired: true });
    vi.mocked(setupAdministrator).mockImplementation(async () => {
      vi.mocked(getAuthenticationStatus).mockResolvedValue({
        setupRequired: false,
      });
      return session;
    });

    renderRoutes("/");

    await user.type(await screen.findByLabelText("显示名称 *"), "家庭管理员");
    await user.type(screen.getByLabelText("用户名 *"), "admin");
    await user.type(screen.getByLabelText("密码 *"), "a-secure-password");
    await user.type(screen.getByLabelText("确认密码 *"), "a-secure-password");
    await user.click(screen.getByRole("button", { name: "创建账户" }));

    expect(setupAdministrator).toHaveBeenCalledWith({
      displayName: "家庭管理员",
      password: "a-secure-password",
      username: "admin",
    });
    expect(await screen.findByRole("heading", { name: "还没有媒体库" })).toBeVisible();
    expect(screen.getByText("家庭管理员")).toBeVisible();
  });

  it("returns an expired protected session to login with a safe notice", async () => {
    vi.mocked(getAuthenticationStatus).mockResolvedValue({ setupRequired: false });
    vi.mocked(getSession).mockRejectedValue(
      new ApiError({
        code: "session_expired",
        message: "expired",
        requestId: "request-test",
        status: 401,
      }),
    );

    renderRoutes("/settings/general");

    expect(await screen.findByRole("heading", { name: "登录 FolioPath" })).toBeVisible();
    expect(screen.getByText("为了保护您的媒体库，会话已过期。请重新登录。")).toBeVisible();
  });

  it("opens the default browse view when an authenticated library exists", async () => {
    vi.mocked(getAuthenticationStatus).mockResolvedValue({
      setupRequired: false,
    });
    vi.mocked(getSession).mockResolvedValue(session);
    vi.mocked(listLibraries).mockResolvedValue({
      items: [
        {
          assetCount: 12,
          directoryCount: 3,
          displayPath: "家庭照片",
          id: "01JTESTLIBRARY0000000000000",
          lastSuccessfulScanAt: "2026-07-29T00:00:00Z",
          latestScanId: "01JTESTSCAN000000000000000",
          name: "家庭照片",
          status: "ready",
        },
      ],
      nextCursor: null,
    });

    renderRoutes("/");

    expect(
      await screen.findByRole("heading", {
        name: "浏览媒体库 01JTESTLIBRARY0000000000000",
      }),
    ).toBeVisible();
  });

  it("does not reveal whether invalid credentials matched an account", async () => {
    const user = userEvent.setup();
    vi.mocked(getAuthenticationStatus).mockResolvedValue({ setupRequired: false });
    vi.mocked(getSession).mockRejectedValue(
      new ApiError({
        code: "authentication_required",
        message: "required",
        requestId: "request-test",
        status: 401,
      }),
    );
    vi.mocked(login).mockRejectedValue(
      new ApiError({
        code: "invalid_credentials",
        message: "invalid",
        requestId: "request-test",
        status: 401,
      }),
    );

    renderRoutes("/login");

    await user.type(await screen.findByLabelText("用户名 *"), "someone");
    await user.type(screen.getByLabelText("密码 *"), "incorrect");
    await user.click(screen.getByRole("button", { name: "登录" }));

    expect(await screen.findByText("用户名或密码不正确。")).toBeVisible();
  });
});

function renderRoutes(initialEntry: string) {
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
          <MemoryRouter initialEntries={[initialEntry]}>
            <AppRoutes />
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
}
