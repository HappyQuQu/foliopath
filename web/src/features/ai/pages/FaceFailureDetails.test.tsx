import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";
import type { FaceSettingsSnapshot } from "../../../lib/api/ai";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { FaceAnalysisStatus, FaceFailureDetails, FaceGenerationHint } from "./FaceFailureDetails";

const library = { id: "lib_4", status: "ready" as const };
const settings: FaceSettingsSnapshot = { libraryId: library.id, enabled: true, state: "degraded", revision: 1, activeGenerationId: "face_gen", etag: '"faces-r1"', coverage: { eligible: 999, completed: 4, failed: 0, stale: 0, degraded: 0, complete: false, revision: 1 }, diagnostics: {
  latestTask: { operationId: "aio_failed", state: "failed", errorCode: "library_not_ready", updatedAt: "2026-10-23T00:00:00Z" }, recordedFailures: 0, unrecordedFailures: 0, reasons: [], samples: [],
} };
beforeEach(() => window.localStorage.setItem("foliopath.preferences.v1", '{"locale":"zh-CN"}'));
function show(value = settings, disabled = false, retry = vi.fn()) {
  return render(<MemoryRouter><LocaleProvider><FaceAnalysisStatus settings={value} library={library} /><FaceFailureDetails settings={value} library={library} disabled={disabled} onRetry={retry} /></LocaleProvider></MemoryRouter>);
}
it("distinguishes interrupted work from zero image failures and exposes a recorded task cause", async () => {
  const retry = vi.fn(); show(settings, false, retry);
  expect(screen.getByText("任务中断 · 尚未处理完成")).toBeVisible();
  expect(screen.queryByText("部分失败")).not.toBeInTheDocument();
  await userEvent.click(screen.getByText("查看失败详情"));
  expect(screen.getByText(/媒体库暂不可处理/)).toBeVisible();
  expect(screen.queryByText("library_not_ready")).not.toBeInTheDocument();
  expect(retry).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "补齐缺失并重试失败" }));
  expect(retry).toHaveBeenCalledOnce();
});
it("keeps historical unknowns explicit and links only a known bounded sample within the library", async () => {
  show({ ...settings, coverage: { ...settings.coverage, failed: 3 }, diagnostics: { ...settings.diagnostics!, recordedFailures: 1, unrecordedFailures: 2, reasons: [{ code: "invalid_media", count: 1 }], samples: [{ assetId: "ast_7", name: "broken-image.jpg", code: "invalid_media", updatedAt: "2026-10-23T00:00:00Z" }] } });
  await userEvent.click(screen.getByText("查看失败详情"));
  expect(screen.getByText(/另有 2 张图片暂时无法查看失败原因/)).toBeVisible();
  expect(screen.getByText(/最多展示 20 张失败样本/)).toBeVisible();
  expect(screen.getByRole("link", { name: "broken-image.jpg" }).getAttribute("href")).toMatch(/^\/libraries\/lib_4\/media\/ast_7\?from=/);
});
it("does not invent causes for legacy responses that lack the new diagnostic contract", async () => {
  const { diagnostics: _diagnostics, ...legacy } = settings;
  show(legacy);
  await userEvent.click(screen.getByText("查看失败详情"));
  expect(screen.getByText(/暂时无法查看具体失败原因/)).toBeVisible();
  expect(screen.queryByText("library_not_ready")).not.toBeInTheDocument();
});
it("prevents retry while unavailable and does not pretend a cancellation is an image failure", async () => {
  show({ ...settings, diagnostics: { ...settings.diagnostics!, latestTask: { ...settings.diagnostics!.latestTask!, state: "cancelled", errorCode: "cancelled" } } }, true);
  expect(screen.getByText("任务已取消 · 结果保留")).toBeVisible();
  await userEvent.click(screen.getByText("查看失败详情"));
  expect(screen.getByRole("button", { name: "补齐缺失并重试失败" })).toBeDisabled();
});
it("labels active work waiting on a scan without reporting failure", () => {
  render(<LocaleProvider><FaceAnalysisStatus library={{ ...library, status: "scanning" }} settings={{ ...settings, state: "building", diagnostics: { ...settings.diagnostics!, latestTask: { ...settings.diagnostics!.latestTask!, state: "running", errorCode: null } } }} /></LocaleProvider>);
  expect(screen.getByText("等待媒体库扫描完成")).toBeVisible();
});
it("labels historical coverage without claiming current-model completion", () => {
  render(<LocaleProvider><FaceAnalysisStatus library={library} settings={{ ...settings, state: "ready", generationCurrent: false }} /><FaceGenerationHint settings={{ ...settings, generationCurrent: false }} /></LocaleProvider>);
  expect(screen.getByText("旧模型结果 · 当前模型待补齐")).toBeVisible();
  expect(screen.getByText(/分析设置已更新/)).toBeVisible();
});
it("identifies a proven zero-byte invalid source, not an absent size or another failure", async () => {
  show({ ...settings, diagnostics: { ...settings.diagnostics!, reasons: [{ code: "invalid_media", count: 2 }], samples: [
    { assetId: "ast_1", name: "empty.jpg", code: "invalid_media", sizeBytes: 0, updatedAt: "2026-10-23T00:00:00Z" },
    { assetId: "ast_2", name: "unknown.jpg", code: "invalid_media", updatedAt: "2026-10-23T00:00:00Z" },
    { assetId: "ast_3", name: "unreadable.jpg", code: "media_unreadable", sizeBytes: 0, updatedAt: "2026-10-23T00:00:00Z" },
  ] } });
  await userEvent.click(screen.getByText("查看失败详情"));
  expect(screen.getAllByText("源文件为空（0 B）")).toHaveLength(1);
  expect(screen.getByText(/不能靠反复重试修复/)).toBeVisible();
});
