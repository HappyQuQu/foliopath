import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";

import { ToastProvider } from "../../../components/ui";
import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { GeneralSettingsPage } from "./GeneralSettingsPage";

const meta = {
  title: "Features/Settings/General",
  component: GeneralSettingsPage,
  decorators: [
    (Story) => (
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={["/settings/general"]}>
            <Story />
          </MemoryRouter>
        </ToastProvider>
      </ThemeProvider>
    ),
  ],
  parameters: { layout: "fullscreen" },
  args: {
    logoutPending: false,
    onLogout: async () => undefined,
    session: {
      administrator: {
        displayName: "管理员",
        id: "adm_story",
        username: "admin",
      },
      csrfToken: "storybook-csrf-token-that-is-long-enough",
      expiresAt: "2026-08-04T00:00:00Z",
    },
  },
} satisfies Meta<typeof GeneralSettingsPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
