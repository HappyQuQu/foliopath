import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AISettingsPage } from "./AISettingsPage";
import type { AIModelDownloadPage } from "../../../lib/api/ai";
import { ApiError } from "../../../lib/api/errors";
import { readAIOperationIds, writeAIOperationIds } from "../../../lib/storage/preferences";

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
  libraries: vi.fn(),
  semantic: vi.fn(),
  faces: vi.fn(),
  models: vi.fn(),
  video: vi.fn(),
  videoJob: vi.fn(),
  updateFace: vi.fn(),
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
    data: { pages: [{ items: mocks.libraries() }] },
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("../queries", () => ({
  useVideoSemanticStatusQuery: mocks.video,
  useRequestVideoSemanticJobMutation: () => ({ isPending: false, mutateAsync: mocks.videoJob }),
  useAIModelProxyQuery: () => ({ data: { mode: "inherit", proxyUrl: "", revision: 1, deploymentConfigured: false, effectiveMode: "direct" }, isPending: false, isError: false }),
  useSaveAIModelProxyMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useTestAIModelProxyMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  operationIsActive: (operation: { state: string }) => ["queued", "running", "cancelling"].includes(operation.state),
  useActivateAIModelMutation: () => ({ isPending: false, mutateAsync: mocks.activate }),
  useAIOperationQueries: mocks.operations,
  useAIModelsQuery: () => ({ data: mocks.models(), isError: false, isPending: false, refetch: vi.fn() }),
  useAIModelDownloadsQuery: () => ({ data: mocks.downloads(), isError: false, isPending: false, refetch: vi.fn() }),
  useDownloadAIModelMutation: () => ({ isPending: false, mutateAsync: mocks.download }),
  useCancelAIOperationMutation: () => ({ isPending: false, mutateAsync: mocks.cancel }),
  useClearSemanticDataMutation: () => ({ isPending: false, mutateAsync: mocks.clear }),
  useClearDerivedFaceDataMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useClearManualFaceRelationshipsMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useFaceSettingsQueries: mocks.faces,
  useInstallAIModelCandidateMutation: () => ({ isPending: false, mutateAsync: mocks.install }),
  useRequestSemanticJobMutation: () => ({ isPending: false, mutateAsync: mocks.job }),
  useRequestFaceJobMutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useScanAIModelCandidatesMutation: () => ({ isPending: false, mutateAsync: mocks.scan }),
  useSemanticSettingsQueries: mocks.semantic,
  useUpdateSemanticSettingsMutation: () => ({ isPending: false, mutateAsync: mocks.update }),
  useUpdateFaceSettingsMutation: () => ({ isPending: false, mutateAsync: mocks.updateFace }),
}));

beforeEach(() => {
  Object.values(mocks).forEach((mock) => mock.mockReset());
  mocks.update.mockResolvedValue(settings);
  mocks.updateFace.mockResolvedValue(settings);
  mocks.job.mockResolvedValue(operation("semantic_missing"));
  mocks.videoJob.mockResolvedValue({ ...operation("semantic_missing"), id: "aio_video", kind: "video_semantic_missing" });
  mocks.clear.mockResolvedValue(operation("semantic_clear"));
  mocks.cancel.mockResolvedValue({ ...operation("semantic_missing"), state: "cancelling" });
  mocks.downloads.mockReturnValue({ items: [], runtimeAvailable: false });
  mocks.models.mockReturnValue({ items: [], activeModelId: null, activeFaceModelId: null, revision: 0 });
  mocks.video.mockReturnValue({ data: { libraryId: library.id, enabled: true, offline: false, activeGenerationId: null, coverage: null }, isPending: false, isError: false, refetch: vi.fn() });
  mocks.operations.mockReturnValue([]);
  mocks.libraries.mockReturnValue([library]);
  mocks.semantic.mockReturnValue([{ data: settings, isError: false, isPending: false, refetch: vi.fn() }]);
  mocks.faces.mockReturnValue([{ data: { ...settings, enabled: false, state: "disabled" }, isError: false, isPending: false, refetch: vi.fn() }]);
  mocks.download.mockResolvedValue({ ...operation("semantic_missing"), kind: "model_install", id: "aio_download" });
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
});

it("lets users forget only a confirmed missing browser task without server mutation", async () => {
  const user = userEvent.setup();
  writeAIOperationIds(["aio_missing", "aio_keep"]);
  mocks.operations.mockImplementation((ids: string[]) => ids.map(id => ({
    data: id === "aio_keep" ? operation("semantic_missing") : undefined,
    isPending: false, isError: id !== "aio_keep", refetch: vi.fn(),
    error: new ApiError({ code: "ai_operation_not_found", status: 404, message: "missing", requestId: undefined }),
  })));
  renderPage("/settings/ai?view=tasks");
  expect(screen.getByText("当前服务中找不到此任务")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "移除浏览器记录" }));
  expect(readAIOperationIds()).toEqual(["aio_keep"]);
  expect(screen.queryByText("当前服务中找不到此任务")).not.toBeInTheDocument();
  expect(mocks.cancel).not.toHaveBeenCalled();
  expect(mocks.clear).not.toHaveBeenCalled();
});

it.each([
  new Error("offline"),
  new ApiError({ code: "ai_operation_not_found", status: 500, message: "bad status", requestId: undefined }),
  new ApiError({ code: "authentication_required", status: 401, message: "login", requestId: undefined }),
])("retains retry and local records for unconfirmed missing tasks (%s)", async error => {
  const retry = vi.fn();
  writeAIOperationIds(["aio_retry"]);
  mocks.operations.mockReturnValue([{ data: undefined, isError: true, isPending: false, error, refetch: retry }]);
  renderPage("/settings/ai?view=tasks");
  expect(screen.queryByRole("button", { name: "移除浏览器记录" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "重新尝试" }));
  expect(retry).toHaveBeenCalledOnce();
  expect(readAIOperationIds()).toEqual(["aio_retry"]);
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

it("uses the selected design with truthful status and collapsed compatibility maintenance", async () => {
  const user = userEvent.setup();
  renderPage("/settings/ai");
  expect(screen.getByRole("heading", { name: "AI 能力" })).toBeVisible();
  expect(screen.getByRole("heading", { name: "图片语义分析进度" })).toBeVisible();
  expect(screen.getByText(/不包含视频和人物分析/)).toBeVisible();
  expect(screen.getByRole("progressbar", { name: "家庭照片 图片语义分析进度 覆盖率" })).toHaveAttribute("value", "8");
  expect(screen.getByText("兼容维护：内置运行时未就绪")).toBeVisible();
  expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
  expect(screen.queryByText("CLIP ViT-B-32")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "下载模型" })).not.toBeInTheDocument();
  const details = screen.getByText("部署详情");
  expect(details.closest("details")).not.toHaveAttribute("open");
  await user.click(details);
  expect(screen.getByText(/内置运行时状态仅用于兼容模型维护/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "兼容模型维护" }));
  expect(screen.getByText("暂无可直接下载的兼容模型包")).toBeVisible();
});

it("changes only the URL-selected library with its current ETag", async () => {
  const user = userEvent.setup();
  const second = { ...library, id: "lib_second", name: "旅行照片" };
  const secondSettings = { ...settings, libraryId: second.id, etag: '"second-r7"' };
  mocks.libraries.mockReturnValue([library, second]);
  mocks.semantic.mockReturnValue([settings, secondSettings].map(data => ({ data, isPending: false, isError: false })));
  renderPage("/settings/ai");
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), second.id);
  await user.click(screen.getByRole("switch", { name: "切换 旅行照片 的图片语义搜索" }));
  expect(mocks.update).toHaveBeenCalledWith({ csrfToken: "csrf-token-that-is-long-enough", enabled: false, etag: '"second-r7"', libraryId: second.id });
  expect(mocks.update).toHaveBeenCalledTimes(1);
});

it("does not enable missing models", () => {
  mocks.semantic.mockReturnValue([{ data: { ...settings, enabled: false }, isPending: false, isError: false }]);
  renderPage("/settings/ai");
  expect(screen.getByRole("switch", { name: "切换 家庭照片 的图片语义搜索" })).toBeDisabled();
});

it("keeps unrecorded video coverage unknown instead of complete", () => {
  renderPage("/settings/ai");
  expect(screen.getByText("当前模型代次尚无视频进度记录，不能判定分析已完成。")).toBeVisible();
  expect(screen.queryByRole("progressbar", { name: "家庭照片 视频语义分析进度" })).not.toBeInTheDocument();
});

it("submits video missing work through its own endpoint and confirms full rebuild", async () => {
  const user = userEvent.setup();
  mocks.models.mockReturnValue({ items: [{ id: "model_service", active: true, state: "available", storageMode: "service", purpose: "semantic_image_text", version: "foliopath-ml-v1" }], activeModelId: "model_service", activeFaceModelId: null, revision: 1 });
  renderPage("/settings/ai");
  await user.click(screen.getByRole("button", { name: "补齐视频索引" }));
  expect(mocks.videoJob).toHaveBeenCalledWith(expect.objectContaining({ csrfToken: "csrf-token-that-is-long-enough", libraryId: library.id, mode: "missing", idempotencyKey: expect.any(String) }));
  expect(mocks.job).not.toHaveBeenCalled();
  expect(mocks.operations).toHaveBeenLastCalledWith(expect.arrayContaining(["aio_video"]));
  await user.click(screen.getByRole("button", { name: "重建视频索引" }));
  expect(screen.getByRole("dialog")).toBeVisible();
  expect(mocks.videoJob).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: "取消" }));
  expect(mocks.videoJob).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: "重建视频索引" }));
  await user.click(screen.getByRole("button", { name: "确认重建视频索引" }));
  expect(mocks.videoJob).toHaveBeenLastCalledWith(expect.objectContaining({ libraryId: library.id, mode: "all" }));
});

it("disables video work when the active model is missing", () => {
  renderPage("/settings/ai");
  expect(screen.getByRole("button", { name: "补齐视频索引" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "重建视频索引" })).toBeDisabled();
});

it.each(["/settings/ai", "/settings/ai?view=libraries"])("shows unrecorded image progress honestly in %s", (path) => {
  mocks.semantic.mockReturnValue([{ data: { ...settings, coverageRecorded: false, coverage: { ...settings.coverage, completed: 0, eligible: 0, failed: 0, stale: 0, complete: false } }, isPending: false, isError: false }]);
  renderPage(path);
  expect(screen.getByText("当前模型代次尚无图片进度记录，不能判定分析已完成。")).toBeVisible();
  expect(screen.queryByText("0 / 0")).not.toBeInTheDocument();
  expect(screen.queryByRole("progressbar", { name: "家庭照片 图片语义分析进度 覆盖率" })).not.toBeInTheDocument();
});

it("separates recorded video completion, degradation and pending work", () => {
  mocks.video.mockReturnValue({ data: { libraryId: library.id, enabled: true, offline: true, activeGenerationId: "aig_test", coverage: { eligible: 6, completed: 1, degraded: 1, failed: 1, stale: 1, revision: 2, complete: false } }, isPending: false, isError: false });
  renderPage("/settings/ai");
  expect(screen.getByRole("progressbar", { name: "家庭照片 视频语义分析进度" })).toHaveAttribute("value", "1");
  expect(screen.getByText("待处理 2 · 降级 1 · 失败 1 · 待更新 1")).toBeVisible();
  expect(screen.getByText("已完成 1 / 6")).toBeVisible();
});

it("hides stale video success when refreshing status fails", () => {
  mocks.video.mockReturnValue({ data: { coverage: { eligible: 1, completed: 1 } }, isPending: false, isError: true, refetch: vi.fn() });
  renderPage("/settings/ai");
  expect(screen.queryByRole("progressbar", { name: "家庭照片 视频语义分析进度" })).not.toBeInTheDocument();
});

it.each([
  ["service", "available", "ready", false],
  ["service", "unavailable", "ready", true],
  ["service", "available", "offline", true],
  ["managed", "available", "ready", true],
  ["direct", "available", "ready", true],
])("gates %s / %s / %s independently of legacy runtime", async (storageMode, state, libraryStatus, disabled) => {
  const user = userEvent.setup();
  mocks.models.mockReturnValue({ items: [{ id: "model_service", active: true, state, storageMode, purpose: "semantic_image_text", version: "foliopath-ml-v1" }], activeModelId: "model_service", activeFaceModelId: null, revision: 1 });
  mocks.libraries.mockReturnValue([{ ...library, status: libraryStatus }]);
  mocks.semantic.mockReturnValue([{ data: { ...settings, enabled: false, state: "disabled" }, isPending: false, isError: false }]);
  renderPage("/settings/ai");
  const toggle = screen.getByRole("switch", { name: "切换 家庭照片 的图片语义搜索" });
  if (disabled) {
    expect(toggle).toBeDisabled();
  } else {
    expect(toggle).toBeEnabled();
    await user.click(toggle);
    expect(mocks.update).toHaveBeenCalledWith({ csrfToken: "csrf-token-that-is-long-enough", enabled: true, etag: settings.etag, libraryId: library.id });
  }
});

it("updates face settings with the face ETag rather than the semantic ETag", async () => {
  const user = userEvent.setup();
  mocks.faces.mockReturnValue([{ data: { ...settings, etag: '"face-r9"' }, isPending: false, isError: false }]);
  renderPage("/settings/ai");
  await user.click(screen.getByRole("switch", { name: "切换 家庭照片 的本地人脸分析" }));
  expect(mocks.updateFace).toHaveBeenCalledWith({ csrfToken: "csrf-token-that-is-long-enough", enabled: false, etag: '"face-r9"', libraryId: library.id });
  expect(mocks.update).not.toHaveBeenCalled();
});

it("disables capability changes for an offline library but preserves coverage", () => {
  mocks.libraries.mockReturnValue([{ ...library, status: "offline" }]);
  renderPage("/settings/ai");
  expect(screen.getByRole("switch", { name: "切换 家庭照片 的图片语义搜索" })).toBeDisabled();
  expect(screen.getByRole("progressbar", { name: "家庭照片 图片语义分析进度 覆盖率" })).toHaveAttribute("value", "8");
});

it("keeps failed coverage unknown rather than displaying stale success", () => {
  mocks.semantic.mockReturnValue([{ data: settings, isPending: false, isError: true, refetch: vi.fn() }]);
  renderPage("/settings/ai");
  expect(screen.queryByRole("switch", { name: "切换 家庭照片 的图片语义搜索" })).not.toBeInTheDocument();
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
});

it("preserves the selected library when opening analysis maintenance", async () => {
  const user = userEvent.setup();
  const second = { ...library, id: "lib_second", name: "旅行照片" };
  mocks.libraries.mockReturnValue([library, second]);
  mocks.semantic.mockReturnValue([settings, { ...settings, libraryId: second.id }].map(data => ({ data, isPending: false, isError: false })));
  renderPage("/settings/ai");
  await user.click(screen.getAllByRole("button", { name: "管理分析" })[1]!);
  expect(screen.getByRole("heading", { name: "旅行照片" })).toBeVisible();
  expect(screen.queryByRole("heading", { name: "家庭照片" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "返回智能功能" }));
  expect(screen.getByRole("combobox", { name: "当前媒体库" })).toHaveValue(second.id);
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
