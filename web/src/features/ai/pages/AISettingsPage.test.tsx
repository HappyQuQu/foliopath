import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AISettingsPage } from "./AISettingsPage";
import type { AIModelDownloadPage } from "../../../lib/api/ai";

const mocks = vi.hoisted(() => ({
  activate: vi.fn(),
  cancel: vi.fn(),
  clear: vi.fn(),
  install: vi.fn(),
  job: vi.fn(),
  scan: vi.fn(),
  update: vi.fn(),
  download: vi.fn(),
  downloads: vi.fn(),
  operations: vi.fn(),
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
  useAIOperationQueries: mocks.operations,
  useAIModelsQuery: () => ({ data: { items: [], activeModelId: null, activeFaceModelId: null, revision: 0 }, isError: false, isPending: false, refetch: vi.fn() }),
  useAIModelDownloadsQuery: () => ({ data: mocks.downloads(), isError: false, isPending: false, refetch: vi.fn() }),
  useDownloadAIModelMutation: () => ({ isPending: false, mutateAsync: mocks.download }),
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
  mocks.cancel.mockResolvedValue({ ...operation("semantic_missing"), state: "cancelling" });
  mocks.downloads.mockReturnValue({ items: [], runtimeAvailable: false });
  mocks.operations.mockReturnValue([]);
  mocks.download.mockResolvedValue({ ...operation("semantic_missing"), kind: "model_install", id: "aio_download" });
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

  await user.click(screen.getByText("索引维护"));
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

  await user.click(screen.getByText("离线导入 · 扫描模型目录"));
  expect(screen.getByText(/不接收路径、网址或浏览器上传/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "扫描模型目录" }));
  await user.click(await screen.findByRole("button", { name: "直接使用" }));

  expect(screen.getByRole("heading", { name: "直接使用此模型？" })).toBeVisible();
  expect(screen.getByText(/挂载持续存在且内容哈希不变/)).toBeVisible();
});

it("offers real downloads by catalog ID and surfaces missing inference separately", async () => {
  const user = userEvent.setup();
  const page: AIModelDownloadPage = { runtimeAvailable: false, items: [{
    id: "download_vision", name: "视觉理解测试模型", sourceLabel: "测试来源", purpose: "semantic_image_text", version: "1.0", licenseId: "MIT", packageSizeBytes: 1024 ** 2,
    compatibility: "compatible", installedModelId: null, operation: null,
  }] };
  mocks.downloads.mockReturnValue(page);
  renderPage("/settings/ai?view=models");
  expect(screen.getByText(/当前镜像未包含 AI 推理组件/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "下载模型" }));
  expect(mocks.download).toHaveBeenCalledWith({ downloadId: "download_vision", csrfToken: "csrf-token-that-is-long-enough", idempotencyKey: expect.any(String) });
  expect(await screen.findByText("下载已加入后台队列，可在模型卡片查看进度。")).toBeVisible();
});

it("restores server download progress and cancels with its current validator", async () => {
  const user = userEvent.setup();
  mocks.downloads.mockReturnValue({ runtimeAvailable: true, items: [{
    id: "download_vision", name: "视觉理解测试模型", sourceLabel: "测试来源", purpose: "semantic_image_text", version: "1.0", licenseId: "MIT", packageSizeBytes: 1024 ** 2,
    compatibility: "compatible", installedModelId: null, operation: { ...operation("semantic_missing"), kind: "model_install", state: "running", phase: "copying", id: "aio_server_task", revision: 4, etag: '"aio_server_task-r4"', completedItems: 512 * 1024, totalItems: 1024 ** 2 },
  }] });
  renderPage("/settings/ai?view=models");
  expect(screen.getByRole("progressbar", { name: "视觉理解测试模型 下载进度" })).toHaveAttribute("value", "524288");
  expect(mocks.operations).toHaveBeenCalledWith(expect.arrayContaining(["aio_server_task"]));
  await user.click(screen.getByRole("button", { name: "取消任务" }));
  expect(mocks.cancel).toHaveBeenCalledWith({ csrfToken: "csrf-token-that-is-long-enough", etag: '"aio_server_task-r4"', operationId: "aio_server_task" });
});

it("opens the overview first and supports keyboard navigation to models", async () => {
  const user = userEvent.setup();
  renderPage("/settings/ai");
  expect(screen.getByRole("heading", { name: "让照片更容易找到，也更容易整理" })).toBeVisible();
  screen.getByRole("tab", { name: "概览" }).focus();
  await user.keyboard("{ArrowRight}");
  expect(screen.getByRole("tab", { name: "模型" })).toHaveFocus();
  expect(screen.getByText("暂无可直接下载的兼容模型包")).toBeVisible();
});

it("shows server-owned presets without presenting them as downloadable models", async () => {
  const user = userEvent.setup();
  mocks.downloads.mockReturnValue({ runtimeAvailable: false, items: [], presets: [
    { id: "preset_example", name: "候选语义模型", sourceLabel: "测试来源", purpose: "semantic_image_text", readiness: "package_pending" },
    { id: "preset_face", name: "候选人脸模型", sourceLabel: "测试来源", purpose: "face_detection_embedding", readiness: "adapter_pending" },
  ] });
  renderPage("/settings/ai?view=models");
  expect(screen.getByRole("heading", { name: "项目预设 · 待接入" })).toBeVisible();
  expect(screen.getByText("候选语义模型")).toBeVisible();
  expect(screen.getByText("待准备兼容包")).toBeVisible();
  expect(screen.getByText("待适配")).toBeVisible();
  expect(screen.queryByRole("button", { name: "下载模型" })).not.toBeInTheDocument();
  const advanced = screen.getByText("高级：自定义下载来源");
  expect(advanced.closest("details")).not.toHaveAttribute("open");
  await user.click(advanced);
  expect(advanced.closest("details")).toHaveAttribute("open");
  expect(mocks.download).not.toHaveBeenCalled();
});

it("shows server-restored download tasks even with no browser task history", () => {
  const running = { ...operation("semantic_missing"), kind: "model_install", state: "running", id: "aio_remote_only", completedItems: 512 * 1024, totalItems: 1024 ** 2 };
  mocks.downloads.mockReturnValue({ runtimeAvailable: true, items: [{ id: "download_remote", operation: running }] });
  mocks.operations.mockImplementation((ids: string[]) => ids.map((id) => ({ data: id === running.id ? running : undefined, isPending: false, isError: false })));
  renderPage("/settings/ai?view=tasks");
  expect(screen.getByRole("heading", { name: "安装模型" })).toBeVisible();
  expect(screen.getByText("0.5 MiB / 1 MiB")).toBeVisible();
});

function renderPage(path = "/settings/ai?view=libraries") {
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
