import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";

import { ToastProvider } from "../../../components/ui";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AISettingsPage } from "./AISettingsPage";
import { ApiError } from "../../../lib/api/errors";
import { readAIOperationIds, writeAIOperationIds } from "../../../lib/storage/preferences";
import asyncStyles from "../../../components/ui/AsyncState/AsyncState.module.css";

const mocks = vi.hoisted(() => ({
  activate: vi.fn(),
  cancel: vi.fn(),
  clear: vi.fn(),
  clearFace: vi.fn(),
  clearManualFace: vi.fn(),
  install: vi.fn(),
  job: vi.fn(),
  faceJob: vi.fn(),
  scan: vi.fn(),
  update: vi.fn(),
  download: vi.fn(),
  downloads: vi.fn(),
  operations: vi.fn(),
  libraries: vi.fn(),
  libraryOptions: vi.fn(),
  fetchMoreLibraries: vi.fn(),
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

const serviceModels = {
  items: [
    { id: "model_service", active: true, state: "available", storageMode: "service", purpose: "semantic_image_text", version: "foliopath-ml-v1", architecture: "arm64", packageSizeBytes: 1024 ** 2, licenseId: "MIT" },
    { id: "face_service", active: true, state: "available", storageMode: "service", purpose: "face_detection_embedding", version: "foliopath-face-v1", architecture: "arm64", packageSizeBytes: 2 * 1024 ** 2, licenseId: "MIT" },
  ],
  activeModelId: "model_service",
  activeFaceModelId: "face_service",
  revision: 1,
};

vi.mock("./AITagSettings", () => ({ AITagSettings: () => null }));
vi.mock("./OCRSettings",()=>({OCRSettings:()=> <div>图片文字 OCR</div>}));
vi.mock("./AutomaticTagSettings", () => ({ AutomaticTagSettings: () => <div>自动标签</div> }));

vi.mock("../../libraries", () => ({
  useLibrariesQuery: () => ({
    data: { pages: [{ items: mocks.libraries() }] },
    isError: false,
    isPending: false,
    refetch: vi.fn(),
    fetchNextPage: mocks.fetchMoreLibraries,
    ...mocks.libraryOptions(),
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
  useClearDerivedFaceDataMutation: () => ({ isPending: false, mutateAsync: mocks.clearFace }),
  useClearManualFaceRelationshipsMutation: () => ({ isPending: false, mutateAsync: mocks.clearManualFace }),
  useFaceSettingsQueries: mocks.faces,
  useInstallAIModelCandidateMutation: () => ({ isPending: false, mutateAsync: mocks.install }),
  useRequestSemanticJobMutation: () => ({ isPending: false, mutateAsync: mocks.job }),
  useRequestFaceJobMutation: () => ({ isPending: false, mutateAsync: mocks.faceJob }),
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
  mocks.faceJob.mockResolvedValue({ ...operation("semantic_missing"), id: "aio_face", kind: "face_missing" });
  mocks.videoJob.mockResolvedValue({ ...operation("semantic_missing"), id: "aio_video", kind: "video_semantic_missing" });
  mocks.clear.mockResolvedValue(operation("semantic_clear"));
  mocks.clearFace.mockResolvedValue(operation("semantic_clear"));
  mocks.clearManualFace.mockResolvedValue(operation("semantic_clear"));
  mocks.cancel.mockResolvedValue({ ...operation("semantic_missing"), state: "cancelling" });
  mocks.downloads.mockReturnValue({ items: [], runtimeAvailable: false });
  mocks.models.mockReturnValue(serviceModels);
  mocks.video.mockReturnValue({ data: { libraryId: library.id, enabled: true, offline: false, activeGenerationId: null, coverage: null }, isPending: false, isError: false, refetch: vi.fn() });
  mocks.operations.mockReturnValue([]);
  mocks.libraries.mockReturnValue([library]);
  mocks.libraryOptions.mockReturnValue({ hasNextPage: false, isFetchingNextPage: false });
  mocks.semantic.mockReturnValue([{ data: settings, isError: false, isPending: false, refetch: vi.fn() }]);
  mocks.faces.mockReturnValue([{ data: { ...settings, enabled: false, state: "disabled" }, isError: false, isPending: false, refetch: vi.fn() }]);
  mocks.download.mockResolvedValue({ ...operation("semantic_missing"), kind: "model_install", id: "aio_download" });
  window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}');
});

it.each(["pending", "error"])("keeps unavailable task rows compact and disables writes (%s)", state => {
  const query = { data: state === "error" ? settings : undefined, isPending: state === "pending", isError: state === "error", refetch: vi.fn() };
  mocks.semantic.mockReturnValue([query]);
  mocks.faces.mockReturnValue([query]);
  mocks.video.mockReturnValue(query);
  renderPage("/settings/ai?view=tasks");
  for (const name of ["图片分析", "人物分析", "视频分析"]) {
    const row = screen.getByRole("region", { name });
    expect(within(row).getByRole(state === "pending" ? "status" : "alert")).toHaveClass(asyncStyles.compact!);
    expect(within(row).getByRole("button", { name: "补齐缺失" })).toBeDisabled();
    expect(within(row).queryByRole("progressbar")).not.toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: /更多/ })).not.toBeInTheDocument();
  }
});

it("retries diagnosed face failures through the canonical missing-job action and retains heading focus", async () => {
  mocks.faces.mockReturnValue([{ data: { ...settings, diagnostics: { latestTask: { operationId: "aio_stopped", state: "failed", errorCode: "library_not_ready", updatedAt: "2026-10-23T00:00:00Z" }, recordedFailures: 0, unrecordedFailures: 1, reasons: [], samples: [] } }, isPending: false, isError: false }]);
  renderPage("/settings/ai?view=tasks");
  await userEvent.click(screen.getByText("查看失败详情"));
  await userEvent.click(screen.getByRole("button", { name: "补齐缺失并重试失败" }));
  expect(mocks.faceJob).toHaveBeenCalledWith(expect.objectContaining({ libraryId: library.id, mode: "missing" }));
  expect(screen.getByRole("heading", { name: "人物分析" })).toHaveFocus();
});

it("shows scan waiting and blocks duplicate face admission for an active task", () => {
  mocks.libraries.mockReturnValue([{ ...library, status: "scanning" }]);
  mocks.faces.mockReturnValue([{ data: { ...settings, state: "building", diagnostics: { latestTask: { operationId: "aio_wait", state: "running", errorCode: null, updatedAt: "2026-10-23T00:00:00Z" }, recordedFailures: 0, unrecordedFailures: 0, reasons: [], samples: [] } }, isPending: false, isError: false }]);
  renderPage("/settings/ai?view=tasks");
  const row = screen.getByRole("region", { name: "人物分析" });
  expect(within(row).getByText("等待媒体库扫描完成")).toBeVisible();
  expect(within(row).getByRole("button", { name: "补齐缺失" })).toBeDisabled();
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
  expect(screen.getByText("1 条旧记录已失效")).toBeVisible();
  await user.click(screen.getByText("1 条旧记录已失效"));
  await user.click(screen.getByRole("button", { name: "移除浏览器记录" }));
  expect(readAIOperationIds()).toEqual(["aio_keep"]);
  expect(screen.getByRole("heading", { name: "本浏览器任务记录" })).toHaveFocus();
  expect(screen.queryByText("1 条旧记录已失效")).not.toBeInTheDocument();
  expect(mocks.cancel).not.toHaveBeenCalled();
  expect(mocks.clear).not.toHaveBeenCalled();
});

it("collapses unavailable records and omits zero-count summaries without pretending the server is idle", async () => {
  writeAIOperationIds(["aio_missing_1", "aio_missing_2"]);
  mocks.operations.mockImplementation((ids: string[]) => ids.map(() => ({ isPending: false, isError: true, error: new ApiError({ code: "ai_operation_not_found", status: 404, message: "missing", requestId: undefined }), refetch: vi.fn() })));
  renderPage("/settings/ai?view=tasks");
  const records = screen.getByRole("region", { name: "本浏览器任务记录" });
  expect(within(records).getByText("2 条旧记录已失效")).toBeVisible();
  expect(within(records).getByText("暂无可用任务记录")).toBeVisible();
  for (const button of within(records).getAllByRole("button", { name: "移除浏览器记录" })) expect(button).not.toBeVisible();
  expect(screen.queryByRole("region", { name: "任务状态摘要" })).not.toBeInTheDocument();
  expect(within(records).getByText(/不是完整的服务器任务历史/)).toBeVisible();
  await userEvent.click(within(records).getByText("2 条旧记录已失效"));
  expect(within(records).getAllByRole("button", { name: "移除浏览器记录" })).toHaveLength(2);
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

it("presents task state as a compact summary and one recent-task list", () => {
  const running = { ...operation("semantic_missing"), id: "aio_running", state: "running" as const, totalItems: 100, completedItems: 25 };
  const completed = { ...operation("semantic_clear"), id: "aio_completed", state: "succeeded" as const, phase: "completed" as const, totalItems: 100, completedItems: 100 };
  const failed = { ...operation("semantic_missing"), id: "aio_failed", state: "failed" as const, errorCode: "internal_error", totalItems: 100, completedItems: 40 };
  writeAIOperationIds([running.id, completed.id, failed.id]);
  mocks.operations.mockImplementation((ids: string[]) => ids.map((id) => ({
    data: [running, completed, failed].find((item) => item.id === id),
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  })));

  renderPage("/settings/ai?view=tasks");

  expect(screen.getByRole("heading", { level: 1, name: "智能分析" })).toBeVisible();
  expect(screen.getByRole("button", { name: "处理任务", pressed: true })).toBeVisible();
  const summary = screen.getByRole("region", { name: "任务状态摘要" });
  expect(within(summary).getByText("进行中").previousSibling).toHaveTextContent("1");
  expect(within(summary).getByText("已完成").previousSibling).toHaveTextContent("1");
  expect(within(summary).getByText("失败").previousSibling).toHaveTextContent("1");
  expect(screen.getByRole("region", { name: "最近任务" })).toBeVisible();
  expect(within(screen.getByRole("region", { name: "最近任务" })).getAllByRole("progressbar")).toHaveLength(3);
});

it("uses a compact actionable empty state when no task is tracked", () => {
  renderPage("/settings/ai?view=tasks");
  expect(screen.getByRole("heading", { level: 1, name: "智能分析" })).toBeVisible();
  expect(screen.getByText("暂无可用任务记录")).toBeVisible();
  expect(screen.queryByRole("region", { name: "任务状态摘要" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "管理智能功能" })).toBeVisible();
});

it("keeps focus on the stable destination tab when an in-panel navigation action disappears", async () => {
  renderPage();
  await userEvent.click(screen.getAllByRole("button", { name: "管理处理任务" })[0]!);
  expect(screen.getByRole("button", { name: "处理任务", pressed: true })).toHaveFocus();
  await userEvent.click(screen.getByRole("button", { name: "管理智能功能" }));
  expect(screen.getByRole("button", { name: "功能设置", pressed: true })).toHaveFocus();
});

it("uses the server ETag when toggling one library and shows truthful coverage", async () => {
  const user = userEvent.setup();
  renderPage();

  expect(within(screen.getByRole("region", { name: "智能搜索" })).getByText("8 / 12")).toBeVisible();
  expect(screen.getByText("部分失败")).toBeVisible();

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
  renderPage("/settings/ai?view=tasks");
  const panel = jobPanel("图片分析");

  await user.click(panel.getByRole("button", { name: /更多/ }));
  await user.click(panel.getByRole("button", { name: "全部重建" }));
  expect(screen.getByRole("heading", { name: "全部重建语义索引？" })).toBeVisible();
  expect(screen.getByText(/重建期间仍可使用现有搜索结果/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "取消" }));

  await user.click(screen.getByRole("button", { name: "清除语义数据" }));
  expect(screen.getByRole("heading", { name: "清除可重建语义数据？" })).toBeVisible();
  expect(screen.getByText(/原文件、收藏和标签保留/)).toBeVisible();
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "清除语义数据" }));

  expect(mocks.clear).toHaveBeenCalledWith(expect.objectContaining({
    csrfToken: "csrf-token-that-is-long-enough",
    etag: '"semantic-r4"',
    libraryId: "lib_test",
  }));
});

it("routes the retired model-store deep link to the Compose-managed service information", () => {
  renderPage("/settings/ai?view=models");
  expect(screen.getByRole("heading", { name: "全局 AI 服务" })).toBeVisible();
  expect(screen.getByRole("button", { name: "服务与模型", pressed: true })).toBeVisible();
  expect(screen.queryByRole("button", { name: "下载模型" })).not.toBeInTheDocument();
  expect(screen.queryByText("离线导入 · 扫描模型目录")).not.toBeInTheDocument();
  expect(screen.queryByText("高级：自定义下载来源")).not.toBeInTheDocument();
});

it("keeps result entry points and switches in features without repeating deployment details", () => {
  renderPage("/settings/ai");
  expect(screen.getByRole("heading", { name: "当前媒体库的功能" })).toBeVisible();
  expect(screen.queryByRole("heading", { name: "全局 AI 服务" })).not.toBeInTheDocument();
  expect(within(screen.getByRole("region", { name: "智能搜索" })).getByText("8 / 12")).toBeVisible();
  expect(screen.getByRole("group", { name: "智能分析导航" })).toBeVisible();
  expect(screen.queryByText(/foliopath-ml-v1/)).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "下载模型" })).not.toBeInTheDocument();
  expect(screen.queryByText("高级维护")).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: /打开智能搜索/ })).toHaveAttribute("href", "/libraries/lib_test/search?mode=semantic");
  expect(screen.getByRole("link", { name: /查看人物/ })).toHaveAttribute("href", "/intelligence?view=clusters&libraryId=lib_test");
  expect(screen.getByText("自动标签")).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: /整理标签建议/ })).not.toBeInTheDocument();
  expect(screen.getAllByRole("switch")).toHaveLength(2);
  expect(screen.getAllByRole("button", { name: "管理处理任务" })).toHaveLength(2);
  expect(screen.queryByRole("button", { name: "兼容模型维护" })).not.toBeInTheDocument();
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
  mocks.models.mockReturnValue({ items: [], activeModelId: null, activeFaceModelId: null, revision: 0 });
  mocks.semantic.mockReturnValue([{ data: { ...settings, enabled: false }, isPending: false, isError: false }]);
  renderPage("/settings/ai");
  expect(screen.getByRole("switch", { name: "切换 家庭照片 的图片语义搜索" })).toBeDisabled();
  expect(screen.getByText("智能搜索暂不可用，请稍后重试。")).toBeVisible();
});

it("keeps unrecorded video coverage unknown instead of complete", async () => {
  renderPage("/settings/ai");
  await userEvent.click(screen.getByRole("button", { name: "处理任务" }));
  expect(jobPanel("视频分析").getByText("无当前进度记录")).toBeVisible();
  expect(screen.queryByRole("progressbar", { name: "家庭照片 视频语义分析进度" })).not.toBeInTheDocument();
});

it("submits video missing work through its own endpoint and confirms full rebuild", async () => {
  const user = userEvent.setup();
  mocks.models.mockReturnValue({ items: [{ id: "model_service", active: true, state: "available", storageMode: "service", purpose: "semantic_image_text", version: "foliopath-ml-v1" }], activeModelId: "model_service", activeFaceModelId: null, revision: 1 });
  renderPage("/settings/ai");
  await user.click(screen.getByRole("button", { name: "处理任务" }));
  await user.click(jobPanel("视频分析").getByRole("button", { name: "补齐缺失" }));
  expect(mocks.videoJob).toHaveBeenCalledWith(expect.objectContaining({ csrfToken: "csrf-token-that-is-long-enough", libraryId: library.id, mode: "missing", idempotencyKey: expect.any(String) }));
  expect(mocks.job).not.toHaveBeenCalled();
  expect(mocks.operations).toHaveBeenLastCalledWith(expect.arrayContaining(["aio_video"]));
  await user.click(jobPanel("视频分析").getByRole("button", { name: /更多/ }));
  await user.click(screen.getByRole("button", { name: "重建视频索引" }));
  expect(screen.getByRole("dialog")).toBeVisible();
  expect(mocks.videoJob).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: "取消" }));
  expect(mocks.videoJob).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: "重建视频索引" }));
  await user.click(screen.getByRole("button", { name: "确认重建视频索引" }));
  expect(mocks.videoJob).toHaveBeenLastCalledWith(expect.objectContaining({ libraryId: library.id, mode: "all" }));
});

it("disables video work when the active model is missing", async () => {
  mocks.models.mockReturnValue({ items: [], activeModelId: null, activeFaceModelId: null, revision: 0 });
  renderPage("/settings/ai");
  await userEvent.click(screen.getByRole("button", { name: "处理任务" }));
  expect(jobPanel("视频分析").getByRole("button", { name: "补齐缺失" })).toBeDisabled();
  await userEvent.click(jobPanel("视频分析").getByRole("button", { name: /更多/ }));
  expect(screen.getByRole("button", { name: "重建视频索引" })).toBeDisabled();
});

it.each(["/settings/ai", "/settings/ai?view=libraries"])("shows unrecorded image progress honestly in %s", (path) => {
  mocks.semantic.mockReturnValue([{ data: { ...settings, coverageRecorded: false, coverage: { ...settings.coverage, completed: 0, eligible: 0, failed: 0, stale: 0, complete: false } }, isPending: false, isError: false }]);
  renderPage(path);
  expect((path === "/settings/ai" ? screen : jobPanel("图片分析")).getByText(path === "/settings/ai" ? "尚无处理记录" : "无当前进度记录")).toBeVisible();
  expect(screen.queryByText("0 / 0")).not.toBeInTheDocument();
  expect(screen.queryByRole("progressbar", { name: "家庭照片 图片语义分析进度 覆盖率" })).not.toBeInTheDocument();
});

it("separates recorded video completion, degradation and pending work", async () => {
  mocks.video.mockReturnValue({ data: { libraryId: library.id, enabled: true, offline: true, activeGenerationId: "aig_test", coverage: { eligible: 6, completed: 1, degraded: 1, failed: 1, stale: 1, revision: 2, complete: false } }, isPending: false, isError: false });
  renderPage("/settings/ai");
  await userEvent.click(screen.getByRole("button", { name: "处理任务" }));
  expect(screen.getByRole("progressbar", { name: "家庭照片 视频语义分析进度" })).toHaveAttribute("value", "1");
  expect(screen.getByText("待处理 2 · 部分未完成 1 · 失败 1 · 待更新 1")).toBeVisible();
  expect(screen.getByText("已完成 1 / 6")).toBeVisible();
});

it("hides stale video success when refreshing status fails", () => {
  mocks.video.mockReturnValue({ data: { coverage: { eligible: 1, completed: 1 } }, isPending: false, isError: true, refetch: vi.fn() });
  renderPage("/settings/ai?view=tasks");
  expect(screen.queryByRole("progressbar", { name: "家庭照片 视频语义分析进度" })).not.toBeInTheDocument();
  expect(jobPanel("视频分析").getByRole("button", { name: "补齐缺失" })).toBeDisabled();
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
    expect(mocks.job).toHaveBeenCalledWith(expect.objectContaining({
      csrfToken: "csrf-token-that-is-long-enough",
      idempotencyKey: expect.any(String),
      libraryId: library.id,
      mode: "missing",
    }));
  }
});

it("starts missing face analysis when the simple capability switch is enabled", async () => {
  const user = userEvent.setup();
  mocks.models.mockReturnValue({
    items: [{ id: "face_service", active: true, state: "available", storageMode: "service", purpose: "face_detection_embedding", version: "foliopath-ml-v1" }],
    activeModelId: null,
    activeFaceModelId: "face_service",
    revision: 1,
  });
  renderPage("/settings/ai");
  await user.click(screen.getByRole("switch", { name: "切换 家庭照片 的本地人脸分析" }));
  expect(mocks.updateFace).toHaveBeenCalledWith({
    csrfToken: "csrf-token-that-is-long-enough",
    enabled: true,
    etag: settings.etag,
    libraryId: library.id,
  });
  expect(mocks.faceJob).toHaveBeenCalledWith(expect.objectContaining({
    csrfToken: "csrf-token-that-is-long-enough",
    idempotencyKey: expect.any(String),
    libraryId: library.id,
    mode: "missing",
  }));
  expect(mocks.operations).toHaveBeenLastCalledWith(expect.arrayContaining(["aio_face"]));
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
  expect(within(screen.getByRole("region", { name: "智能搜索" })).getByText("8 / 12")).toBeVisible();
});

it("requires explicit confirmation for both face clears and restores focus on cancel", async () => {
  const user = userEvent.setup();
  renderPage("/settings/ai?view=faces");
  await user.click(jobPanel("人物分析").getByRole("button", { name: /更多/ }));
  for (const name of ["清除可重建人脸数据", "清除人物应用状态"]) {
    const trigger = screen.getByRole("button", { name });
    await user.click(trigger);
    const dialog = screen.getByRole("dialog", { name });
    expect(mocks.clearFace).not.toHaveBeenCalled();
    expect(mocks.clearManualFace).not.toHaveBeenCalled();
    await user.click(within(dialog).getByRole("button", { name: "取消" }));
    expect(trigger).toHaveFocus();
  }
  await user.click(screen.getByRole("button", { name: "清除可重建人脸数据" }));
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "清除可重建人脸数据" }));
  expect(mocks.clearFace).toHaveBeenCalledWith(expect.objectContaining({ libraryId: library.id, etag: settings.etag }));
});

it("keeps failed coverage unknown rather than displaying stale success", () => {
  mocks.semantic.mockReturnValue([{ data: settings, isPending: false, isError: true, refetch: vi.fn() }]);
  renderPage("/settings/ai");
  expect(screen.queryByRole("switch", { name: "切换 家庭照片 的图片语义搜索" })).not.toBeInTheDocument();
  expect(within(screen.getByRole("region", { name: "智能搜索" })).queryByRole("progressbar")).not.toBeInTheDocument();
  expect(within(screen.getByRole("region", { name: "人物分析" })).getByRole("progressbar")).toBeVisible();
});

it("preserves the selected library across fixed tabs, jobs and result links", async () => {
  const user = userEvent.setup();
  const second = { ...library, id: "lib_second", name: "旅行照片" };
  mocks.libraries.mockReturnValue([library, second]);
  mocks.semantic.mockReturnValue([settings, { ...settings, libraryId: second.id }].map(data => ({ data, isPending: false, isError: false })));
  mocks.faces.mockReturnValue([settings, { ...settings, libraryId: second.id }].map(data => ({ data, isPending: false, isError: false })));
  renderPage("/settings/ai");
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), second.id);
  expect(screen.getByRole("link", { name: /打开智能搜索/ })).toHaveAttribute("href", "/libraries/lib_second/search?mode=semantic");
  expect(screen.getByRole("link", { name: /查看人物/ })).toHaveAttribute("href", "/intelligence?view=clusters&libraryId=lib_second");
  await user.click(screen.getByRole("button", { name: "处理任务" }));
  expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  await user.click(jobPanel("图片分析").getByRole("button", { name: "补齐缺失" }));
  await user.click(jobPanel("人物分析").getByRole("button", { name: "补齐缺失" }));
  expect(mocks.job).toHaveBeenCalledWith(expect.objectContaining({ libraryId: second.id, mode: "missing" }));
  expect(mocks.faceJob).toHaveBeenCalledWith(expect.objectContaining({ libraryId: second.id, mode: "missing" }));
  await user.click(screen.getByRole("button", { name: "服务与模型" }));
  expect(screen.getByRole("heading", { name: "全局 AI 服务" })).toBeVisible();
  expect(screen.queryByText(/foliopath-ml-v1/)).not.toBeInTheDocument();
  expect(screen.queryByRole("combobox", { name: "当前媒体库" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "功能设置" }));
  expect(screen.getByRole("combobox", { name: "当前媒体库" })).toHaveValue(second.id);
});

it("shows server-restored download tasks even with no browser task history", () => {
  const running = { ...operation("semantic_missing"), kind: "model_install", state: "running", id: "aio_remote_only", completedItems: 512 * 1024, totalItems: 1024 ** 2 };
  mocks.downloads.mockReturnValue({ runtimeAvailable: true, items: [{ id: "download_remote", operation: running }] });
  mocks.operations.mockImplementation((ids: string[]) => ids.map((id) => ({ data: id === running.id ? running : undefined, isPending: false, isError: false })));
  renderPage("/settings/ai?view=tasks");
  expect(screen.getByRole("heading", { name: "安装模型" })).toBeVisible();
  expect(screen.getByText("0.5 MiB / 1 MiB")).toBeVisible();
});

it.each(["libraries", "faces"])("maps the old %s link to processing without losing the selected library", (view) => {
  const second = { ...library, id: "lib_second", name: "旅行照片" };
  mocks.libraries.mockReturnValue([library, second]);
  mocks.semantic.mockReturnValue([settings, { ...settings, libraryId: second.id, coverage: { ...settings.coverage, completed: 6 } }].map(data => ({ data, isPending: false, isError: false })));
  mocks.faces.mockReturnValue([settings, { ...settings, libraryId: second.id, coverage: { ...settings.coverage, completed: 4 } }].map(data => ({ data, isPending: false, isError: false })));
  renderPage(`/settings/ai?view=${view}&library=${second.id}`);
  expect(screen.getByRole("button", { name: "处理任务", pressed: true })).toBeVisible();
  expect(screen.getByRole("combobox", { name: "当前媒体库" })).toHaveValue(second.id);
  expect(jobPanel("图片分析").getByText(/6 \/ 12 · 失败/)).toBeVisible();
  expect(jobPanel("人物分析").getByText(/4 \/ 12 · 失败/)).toBeVisible();
  expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  expect(screen.getByText(/不是完整的服务器任务历史/)).toBeVisible();
});

it.each(["", "&view=tasks"])("rejects an unknown library instead of exposing another library's actions (%s)", async (view) => {
  const user = userEvent.setup();
  renderPage(`/settings/ai?library=lib_unknown${view}`);
  expect(screen.getByText("所选媒体库不可用")).toBeVisible();
  expect(screen.getByRole("combobox", { name: "当前媒体库" })).toHaveValue("");
  expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "补齐缺失" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: /打开智能搜索/ })).not.toBeInTheDocument();
  expect(mocks.update).not.toHaveBeenCalled();
  expect(mocks.job).not.toHaveBeenCalled();
  expect(mocks.faceJob).not.toHaveBeenCalled();
  await user.selectOptions(screen.getByRole("combobox", { name: "当前媒体库" }), library.id);
  expect(screen.queryByText("所选媒体库不可用")).not.toBeInTheDocument();
  expect(screen.getByRole("heading", { name: view ? "图片分析" : "智能搜索" })).toBeVisible();
});

it("does not show an unrelated library's query error in the selected library", () => {
  const second = { ...library, id: "lib_second", name: "旅行照片" };
  mocks.libraries.mockReturnValue([library, second]);
  const failed = { data: undefined, isPending: false, isError: true, refetch: vi.fn() };
  mocks.semantic.mockReturnValue([{ data: settings, isPending: false, isError: false }, failed]);
  mocks.faces.mockReturnValue([{ data: { ...settings, enabled: false }, isPending: false, isError: false }, failed]);
  renderPage(`/settings/ai?library=${library.id}`);
  expect(screen.getByRole("switch", { name: "切换 家庭照片 的图片语义搜索" })).toBeEnabled();
  expect(within(screen.getByRole("region", { name: "智能搜索" })).getByText("8 / 12")).toBeVisible();
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "重新尝试" })).not.toBeInTheDocument();
});

it("keeps paginated libraries reachable from features and processing", async () => {
  const user = userEvent.setup();
  mocks.libraryOptions.mockReturnValue({ hasNextPage: true, isFetchingNextPage: false });
  renderPage();
  await user.click(screen.getByRole("button", { name: "载入更多媒体库" }));
  expect(mocks.fetchMoreLibraries).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: "处理任务" }));
  await user.click(screen.getByRole("button", { name: "载入更多媒体库" }));
  expect(mocks.fetchMoreLibraries).toHaveBeenCalledTimes(2);
});

it.each([
  { reason: "offline library", libraryStatus: "offline", faceState: "ready", modelAvailable: true },
  { reason: "clearing face data", libraryStatus: "ready", faceState: "clearing", modelAvailable: true },
  { reason: "unavailable model", libraryStatus: "ready", faceState: "ready", modelAvailable: false },
])("disables face processing for $reason", async ({ libraryStatus, faceState, modelAvailable }) => {
  mocks.libraries.mockReturnValue([{ ...library, status: libraryStatus }]);
  mocks.faces.mockReturnValue([{ data: { ...settings, state: faceState }, isPending: false, isError: false }]);
  if (!modelAvailable) mocks.models.mockReturnValue({ ...serviceModels, activeFaceModelId: null });
  renderPage("/settings/ai?view=tasks");
  const panel = jobPanel("人物分析");
  expect(panel.getByRole("button", { name: "补齐缺失" })).toBeDisabled();
  await userEvent.click(panel.getByRole("button", { name: /更多/ }));
  expect(panel.getByRole("button", { name: "全部重建" })).toBeDisabled();
  if (faceState === "clearing") {
    expect(panel.getByRole("button", { name: "清除可重建人脸数据" })).toBeDisabled();
    expect(panel.getByRole("button", { name: "清除人物应用状态" })).toBeDisabled();
  }
  expect(mocks.faceJob).not.toHaveBeenCalled();
});

it("confirms a full face rebuild and scopes it to the selected library", async () => {
  const user = userEvent.setup();
  mocks.faces.mockReturnValue([{ data: settings, isPending: false, isError: false }]);
  renderPage("/settings/ai?view=tasks");
  const panel = jobPanel("人物分析");
  await user.click(panel.getByRole("button", { name: /更多/ }));
  const trigger = panel.getByRole("button", { name: "全部重建" });
  await user.click(trigger);
  expect(mocks.faceJob).not.toHaveBeenCalled();
  const dialog = screen.getByRole("dialog", { name: "重新分析全部人脸？" });
  expect(within(dialog).getByText(library.name)).toBeVisible();
  await user.click(within(dialog).getByRole("button", { name: "取消" }));
  expect(trigger).toHaveFocus();
  expect(mocks.faceJob).not.toHaveBeenCalled();
  await user.click(trigger);
  await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "重新分析全部人脸？" }));
  expect(mocks.faceJob).toHaveBeenCalledWith(expect.objectContaining({ libraryId: library.id, mode: "all", idempotencyKey: expect.any(String) }));
});

it("submits exact impact counts and the face revision for manual relationship clearing", async () => {
  const user = userEvent.setup();
  mocks.faces.mockReturnValue([{ data: { ...settings, etag: '"face-r9"' }, isPending: false, isError: false }]);
  renderPage("/settings/ai?view=tasks");
  await user.click(jobPanel("人物分析").getByRole("button", { name: /更多/ }));
  await user.click(screen.getByRole("button", { name: "清除人物应用状态" }));
  const dialog = screen.getByRole("dialog", { name: "清除人物应用状态" });
  for (const [label, value] of [["人物数", "2"], ["归类数", "3"], ["约束数", "4"]] as const) {
    await user.clear(within(dialog).getByRole("spinbutton", { name: label }));
    await user.type(within(dialog).getByRole("spinbutton", { name: label }), value);
  }
  expect(mocks.clearManualFace).not.toHaveBeenCalled();
  await user.click(within(dialog).getByRole("button", { name: "清除人物应用状态" }));
  expect(mocks.clearManualFace).toHaveBeenCalledWith(expect.objectContaining({
    libraryId: library.id, etag: '"face-r9"', personCount: 2, assignmentCount: 3, constraintCount: 4,
  }));
});

function jobPanel(name: string) {
  const article = screen.getByRole("heading", { name }).closest("section");
  expect(article).not.toBeNull();
  return within(article!);
}

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
