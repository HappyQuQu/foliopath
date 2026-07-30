import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";

import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { ToastProvider } from "../../ui/Toast/ToastProvider";
import { AppShell } from "./AppShell";

const meta = {
  title: "Patterns/AppShell",
  component: AppShell,
  decorators: [
    (Story) => (
      <ThemeProvider>
        <ToastProvider>
          <MemoryRouter>
            <Story />
          </MemoryRouter>
        </ToastProvider>
      </ThemeProvider>
    ),
  ],
  parameters: { layout: "fullscreen" },
  args: {
    active: "libraries",
    homeHref: "/",
    identity: "家庭管理员",
    searchHref: "/search",
    settingsHref: "/settings/general",
    title: "媒体库",
    children: <div style={{ padding: "2rem" }}>页面内容</div>,
  },
} satisfies Meta<typeof AppShell>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
