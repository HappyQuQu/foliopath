import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AISettingsPage } from "./AISettingsPage";

const mocks = vi.hoisted(() => ({
  activate: vi.fn(),
  cancel: vi.fn(),
  clear: vi.fn(),
  install: vi.fn(),
  job: vi.fn(),
  scan: vi.fn(),
  update: vi.fn(),
}));

const library = {
  assetCount: 12,
  automaticDiscoveryErrorCode: null,
  automaticDiscoveryStatus: "active" as const,
  contentRevision: 2,
  directoryCount: 3,
  displayPath: "家庭照片",
  id: "lib_test",
  lastAutomaticDiscoveryAt: null,
  lastSuccessfulScanAt: "2026-08-01T00:00:00Z",
  latestScanId: "scan_test",
  name: "家庭照片",
  status: "ready" as const,
};

const settings = {
  activeGenerationId: "gen_test",
  coverage: {
    complete: false,
    completed: 8,
    eligible: 12,
    failed: 1,
    revision: 4,
    stale: 3,
  },
  enabled: true,
  etag: '"semantic-r4"',
  libraryId: library.id,
  revision: 4,
  state: "degraded" as const,
};

vi.mock("../../libraries", () => ({
  useLibrariesQuery: () => ({
    data: { pages: [{ items: [library] }] },
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("../queries", () => ({
  operationIsActive: (operation: { state: string }) => ["queued", "running", "cancelling"].includes(operation.state),
  useActivateAIModelMutation: () => ({ isPending: false, mutateAsync: mocks.activate }),
  useAIOperationQueries: () => [],
  useAIModelsQuery: () => ({ data: { items: [], activeModelId: null, activeFaceModelId: null, revision: 0 }, isError: false, isPending: false, refetch: vi.fn() }),
  useCancelAIOperationMutation: () => ({ isPending: false, mutateAsync: mocks.cancel }),
  useClearSemanticDataMutation: () => ({ isPending: false, mutateAsync: mocks.clear }),
  useClearDerivedFaceDataMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useClearManualFaceRelationshipsMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useFaceSettingsQueries: () => [],
  useInstallAIModelCandidateMutation: () => ({ isPending: false, mutateAsync: mocks.install }),
  useRequestSemanticJobMutation: () => ({ isPending: false, mutateAsync: mocks.job }),
  useRequestFaceJobMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useScanAIModelCandidatesMutation: () => ({ isPending: false, mutateAsync: mocks.scan }),
  useSemanticSettingsQueries: () => [{ data: settings, isError: false, isPending: false, refetch: vi.fn() }],
  useUpdateSemanticSettingsMutation: () => ({ isPending: false, mutateAsync: mocks.update }),
  useUpdateFaceSettingsMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

beforeEach(() => {
  Object.values(mocks).forEach((mock) => mock.mockReset());
  mocks.update.mockResolvedValue(settings);
  mocks.job.mockResolvedValue(operation("semantic_missing"));
  mocks.clear.mockResolvedValue(operation("semantic_clear"));
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
});

it("uses the server ETag when toggling one library and shows truthful coverage", async () => {
  const user = userEvent.setup();
  renderPage();

  expect(screen.getByText("8 / 12")).toBeVisible();
  expect(screen.getByText("最近任务存在脱敏失败；可靠结果仍保留，可补齐缺失后再检查。")).toBeVisible();

  await user.click(screen.getByRole("switch", { name: "切换 家庭照片 的图片语义搜索" }));

  expect(mocks.update).toHaveBeenCalledWith({
    csrfToken: "csrf-token-that-is-long-enough",
    enabled: false,
    etag: '"semantic-r4"',
    libraryId: "lib_test",
  });
});

it("keeps full rebuild and semantic clearing behind distinct safety confirmations", async () => {
  const user = userEvent.setup();
  renderPage();

  await user.click(screen.getByRole("button", { name: "全部重建" }));
  expect(screen.getByRole("heading", { name: "全部重建语义索引？" })).toBeVisible();
  expect(screen.getByText(/现有可靠 generation 会继续服务/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "取消" }));

  await user.click(screen.getByRole("button", { name: "清除语义数据" }));
  expect(screen.getByRole("heading", { name: "清除可重建语义数据？" })).toBeVisible();
  expect(screen.getByText(/不会删除模型、原始媒体、收藏或人工标签/)).toBeVisible();
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "清除语义数据" }));

  expect(mocks.clear).toHaveBeenCalledWith(expect.objectContaining({
    csrfToken: "csrf-token-that-is-long-enough",
    etag: '"semantic-r4"',
    libraryId: "lib_test",
  }));
});

it("scans only the fixed model directory and confirms direct-use availability risk", async () => {
  const user = userEvent.setup();
  mocks.scan.mockResolvedValue({
    candidates: [{
      architecture: "arm64",
      compatibility: "compatible",
      id: "candidate_test",
      licenseId: "Apache-2.0",
      packageSizeBytes: 1024 ** 2,
      purpose: "semantic_image_text",
      version: "1.0.0",
    }],
    revision: 1,
    scannedAt: "2026-09-02T00:00:00Z",
    truncated: false,
  });
  renderPage("/settings/ai?view=models");

  expect(screen.getByText(/不接收路径、网址或浏览器上传/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "扫描模型目录" }));
  await user.click(await screen.findByRole("button", { name: "直接使用" }));

  expect(screen.getByRole("heading", { name: "直接使用此模型？" })).toBeVisible();
  expect(screen.getByText(/挂载持续存在且内容哈希不变/)).toBeVisible();
});

function renderPage(path = "/settings/ai") {
  return render(
    <LocaleProvider>
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={[path]}>
            <AISettingsPage
              session={{
                administrator: { displayName: "管理员", id: "adm_test", username: "admin" },
                csrfToken: "csrf-token-that-is-long-enough",
                expiresAt: "2026-09-03T00:00:00Z",
              }}
            />
          </MemoryRouter>
        </ToastProvider>
      </ThemeProvider>
    </LocaleProvider>,
  );
}

function operation(kind: "semantic_missing" | "semantic_clear") {
  return {
    completedItems: 0,
    createdAt: "2026-09-02T00:00:00Z",
    errorCode: null,
    etag: '"operation-r1"',
    id: `operation_${kind}`,
    kind,
    phase: "queued" as const,
    revision: 1,
    state: "queued" as const,
    totalItems: null,
    updatedAt: "2026-09-02T00:00:00Z",
  };
}
