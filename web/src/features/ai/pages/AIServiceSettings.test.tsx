import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { AIServiceSettings } from "./AISettingsOverview";
import type { AIPreparation } from "../../../lib/api/ai";

function renderState(state: AIPreparation["state"], locale: "en" | "zh-CN" = "zh-CN") {
  localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale }));
  return render(<LocaleProvider><AIServiceSettings models={[]} activeModelId={null} activeFaceModelId={null}
    preparation={{ state, completedFiles: 2, totalFiles: 5, completedBytes: 20, totalBytes: 100 }} /></LocaleProvider>);
}

describe("managed model preparation", () => {
  it("shows real download progress without enabling a library", () => {
    renderState("downloading");
    expect(screen.getByRole("progressbar", { name: "AI 模型下载进度" })).toHaveAttribute("value", "20");
    expect(screen.getByText("已准备 2 / 5 个文件")).toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  });
  it("offers recovery guidance without stale progress", () => {
    renderState("failed");
    expect(screen.getByText(/系统会自动重试/)).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });
  it("keeps manual deployments unchanged", () => {
    renderState("disabled");
    expect(screen.queryByText(/正在下载 AI/)).not.toBeInTheDocument();
  });
  it("explains readiness in English", () => {
    renderState("ready", "en");
    expect(screen.getByText("AI models are ready. Enable the features you need for each library.")).toBeInTheDocument();
  });
});
