import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import type { ReleaseInformation, SystemStatus } from "../../../lib/api/release-info";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AboutPage } from "./AboutPage";

const status: SystemStatus = {
  version: "v1.9.0", apiVersion: "v1", runtimeState: "ready", initialized: true, readOnlyMedia: true,
  supportedLocales: ["zh-CN", "en"],
  supportedMedia: { imageMimeTypes: ["image/jpeg", "image/png", "image/webp", "image/gif"], videoMimeTypes: ["video/mp4", "video/quicktime", "video/x-matroska", "video/x-msvideo"], videoTranscoding: false },
};
const releases: ReleaseInformation = {
  currentVersion: "v1.9.0", latestVersion: "v1.9.0", updateAvailable: false, checkedAt: "2026-10-29T09:30:00Z",
  releases: [{ version: "v1.9.0", name: "合成更新记录 · Synthetic release", publishedAt: "2026-10-29T09:00:00Z", summary: "Synthetic release fixture", notes: "### 新功能\n- 支持 7z 图集与文件夹封面\n- 目录与媒体滚动加载", url: "https://github.com/HappyQuQu/foliopath/releases/tag/v1.9.0" }],
};
const originalFetch = globalThis.fetch.bind(globalThis);

function FixtureBoundary({ children, failed }: { children: ReactNode; failed: boolean }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  window.fetch = async (input, init) => {
    const url = new URL(input instanceof Request ? input.url : String(input), window.location.origin);
    if (url.pathname === "/api/v1/status") return Response.json(status);
    if (url.pathname === "/api/v1/releases") return failed
      ? Response.json({ error: { code: "internal_error", message: "Synthetic upstream failure", requestId: "synthetic-about" } }, { status: 503 })
      : Response.json(releases);
    return originalFetch(input, init);
  };
  useEffect(() => () => { window.fetch = originalFetch; }, []);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

const meta = {
  title: "Features/Settings/About",
  component: AboutPage,
  decorators: [(Story, context) => <FixtureBoundary failed={context.parameters.failed === true}><ThemeProvider><ToastProvider><MemoryRouter initialEntries={["/settings/about"]}><Story /></MemoryRouter></ToastProvider></ThemeProvider></FixtureBoundary>],
  parameters: { layout: "fullscreen", docs: { description: { component: "Production AboutPage with explicitly synthetic version/release contracts. Project links are real; no updater or enabled AI capability is simulated." } } },
  args: { session: { administrator: { id: "adm_story", displayName: "合成管理员", username: "synthetic" }, csrfToken: "synthetic-token-long-enough-for-story", expiresAt: "2026-10-30T00:00:00Z" } },
} satisfies Meta<typeof AboutPage>;
export default meta;
type Story = StoryObj<typeof meta>;
export const ProjectOverview: Story = {};
export const UpdateCheckFailed: Story = { parameters: { failed: true } };
