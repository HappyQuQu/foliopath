import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import type { LibrarySummary } from "../../../lib/api/libraries";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { libraryKeys } from "../queries";
import { LibrariesPage } from "./LibrariesPage";

const session: AuthenticatedSession = {
  administrator: {
    displayName: "管理员",
    id: "adm_story",
    username: "admin",
  },
  csrfToken: "storybook-csrf-token-that-is-long-enough",
  expiresAt: "2099-01-01T00:00:00Z",
};

const libraries: LibrarySummary[] = [
  {
    assetCount: 6_221,
    automaticDiscoveryErrorCode: null,
    automaticDiscoveryStatus: "active",
    contentRevision: 24,
    directoryCount: 184,
    displayPath: "/library/coser",
    id: "lib_coser",
    lastAutomaticDiscoveryAt: "2026-10-08T07:42:00Z",
    lastSuccessfulScanAt: "2026-10-08T07:40:00Z",
    latestScanId: "scan_coser",
    name: "coser",
    status: "ready",
  },
  {
    assetCount: 18_540,
    automaticDiscoveryErrorCode: "source_unavailable",
    automaticDiscoveryStatus: "degraded",
    contentRevision: 81,
    directoryCount: 526,
    displayPath: "/library/archive",
    id: "lib_archive",
    lastAutomaticDiscoveryAt: "2026-10-07T23:10:00Z",
    lastSuccessfulScanAt: "2026-10-07T23:08:00Z",
    latestScanId: "scan_archive",
    name: "影像归档",
    status: "offline",
  },
];

const meta = {
  title: "Features/Libraries/Management",
  component: LibrariesPage,
  args: { session },
  decorators: [
    (Story) => (
      <LibraryStory>
        <Story />
      </LibraryStory>
    ),
  ],
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Accepted library-management layout rendered by the production component with synthetic contract fixtures.",
      },
    },
  },
} satisfies Meta<typeof LibrariesPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Overview: Story = {};

function LibraryStory({ children }: { children: ReactNode }) {
  const [client] = useState(() => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { enabled: false, retry: false, staleTime: Infinity },
      },
    });
    queryClient.setQueryData(libraryKeys.list(), {
      pages: [{ items: libraries, nextCursor: null }],
      pageParams: [undefined],
    });
    return queryClient;
  });

  return (
    <ThemeProvider>
      <QueryClientProvider client={client}>
        <ToastProvider>
          <MemoryRouter initialEntries={["/settings/libraries"]}>
            {children}
            <aside
              role="note"
              style={{
                background: "var(--color-surface)",
                color: "var(--color-text-muted)",
                fontSize: "var(--font-size-2)",
                padding: "var(--space-2)",
              }}
            >
              组件预览 · 合成数据 / Component preview · synthetic data
            </aside>
          </MemoryRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}
