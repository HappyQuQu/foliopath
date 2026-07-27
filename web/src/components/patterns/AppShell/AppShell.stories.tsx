import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";

import { ThemeProvider } from "../../../lib/theme/ThemeProvider";
import { AppShell } from "./AppShell";

const meta = {
  title: "Patterns/AppShell",
  component: AppShell,
  decorators: [
    (Story) => (
      <ThemeProvider>
        <MemoryRouter>
          <Story />
        </MemoryRouter>
      </ThemeProvider>
    ),
  ],
  parameters: { layout: "fullscreen" },
  args: {
    active: "libraries",
    identity: "家庭管理员",
    librariesHref: "/settings/libraries",
    settingsHref: "/settings/general",
    title: "媒体库",
    children: <div style={{ padding: "2rem" }}>页面内容</div>,
  },
} satisfies Meta<typeof AppShell>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
